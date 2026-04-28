// Package service provides RFC 6762/6763 mDNS service announcement and
// discovery for any TCP/UDP service on a local network.
//
// The package wraps github.com/grandcat/zeroconf with a narrower, more
// opinionated API: callers configure an Announcement struct or a Browser
// and forget about the underlying mDNS plumbing. Service-type strings
// are validated, TXT records are typed as map[string]string, and both
// announcement and browsing have explicit Start/Stop lifecycles.
//
// Wire layout matches what Android NsdManager and Apple Bonjour expect:
// service types are of the form "_<name>._<proto>" (e.g. "_lava-api._tcp"
// or "_http._udp"), and instance names are the human-friendly bit before
// that (e.g. "Lava API on living-room-pi").
package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/grandcat/zeroconf"
)

// DefaultDomain is the mDNS domain. The empty string and "local." both
// resolve to the standard ".local." multicast scope; we use the empty
// form internally and let zeroconf substitute the default.
const DefaultDomain = "local."

// Announcement describes a service we want to advertise on the LAN.
//
// Required fields: Name, ServiceType, Port. Optional: Domain (default
// "local."), Hostname (default the OS hostname), TXT (key/value records
// — values must NOT contain '=' or '\0'), IPs (default: all v4/v6 of the
// active interfaces).
type Announcement struct {
	Name        string
	ServiceType string
	Port        int
	Domain      string
	Hostname    string
	TXT         map[string]string
	IPs         []string
}

// Validate returns an error if the Announcement is incomplete or
// malformed. Validate is called inside Announce.
func (a Announcement) Validate() error {
	if a.Name == "" {
		return errors.New("mdns: Announcement.Name is required")
	}
	if err := validateServiceType(a.ServiceType); err != nil {
		return err
	}
	if a.Port <= 0 || a.Port > 65535 {
		return fmt.Errorf("mdns: Announcement.Port %d out of range (1..65535)", a.Port)
	}
	for k, v := range a.TXT {
		if strings.ContainsAny(k, "=\x00") {
			return fmt.Errorf("mdns: TXT key %q contains '=' or NUL", k)
		}
		if strings.ContainsAny(v, "\x00") {
			return fmt.Errorf("mdns: TXT value for %q contains NUL", k)
		}
	}
	return nil
}

// Service is an active mDNS announcement. Stop terminates it.
type Service struct {
	srv  *zeroconf.Server
	mu   sync.Mutex
	done bool
}

// Announce registers the service via mDNS and returns immediately.
// The service remains advertised until Stop is called.
func Announce(a Announcement) (*Service, error) {
	if err := a.Validate(); err != nil {
		return nil, err
	}
	domain := a.Domain
	if domain == "" {
		domain = DefaultDomain
	}
	txt := make([]string, 0, len(a.TXT))
	// zeroconf wants TXT entries as "key=value" strings; iterate in any
	// order — mDNS does not specify TXT-record ordering.
	for k, v := range a.TXT {
		txt = append(txt, k+"="+v)
	}
	srv, err := zeroconf.Register(a.Name, a.ServiceType, domain, a.Port, txt, nil)
	if err != nil {
		return nil, fmt.Errorf("mdns: register %q: %w", a.ServiceType, err)
	}
	return &Service{srv: srv}, nil
}

// Stop terminates the announcement. Idempotent.
func (s *Service) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.done {
		return
	}
	s.done = true
	if s.srv != nil {
		s.srv.Shutdown()
	}
}

// Discovered describes a service found by the Browser.
type Discovered struct {
	Instance     string
	ServiceType  string
	Domain       string
	HostName     string
	Port         int
	TXT          map[string]string
	AddrV4       []string
	AddrV6       []string
	Discovered   time.Time
}

// BrowseConfig configures a Browse call.
type BrowseConfig struct {
	ServiceType string        // required, e.g. "_lava-api._tcp"
	Domain      string        // optional, default DefaultDomain
	Timeout     time.Duration // total time to wait for replies; default 5 s
}

// Validate returns an error if the BrowseConfig is malformed.
func (b BrowseConfig) Validate() error {
	return validateServiceType(b.ServiceType)
}

// Browse performs a one-shot mDNS browse for ServiceType. It returns
// every service that responded before Timeout (or before the context
// is cancelled, whichever comes first).
func Browse(ctx context.Context, cfg BrowseConfig) ([]Discovered, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	domain := cfg.Domain
	if domain == "" {
		domain = DefaultDomain
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		return nil, fmt.Errorf("mdns: resolver: %w", err)
	}
	entries := make(chan *zeroconf.ServiceEntry, 32)
	browseCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	if err := resolver.Browse(browseCtx, cfg.ServiceType, domain, entries); err != nil {
		return nil, fmt.Errorf("mdns: browse: %w", err)
	}

	var out []Discovered
	for entry := range entries {
		out = append(out, fromZeroconf(entry))
	}
	return out, nil
}

func fromZeroconf(e *zeroconf.ServiceEntry) Discovered {
	d := Discovered{
		Instance:    e.Instance,
		ServiceType: e.Service,
		Domain:      e.Domain,
		HostName:    e.HostName,
		Port:        e.Port,
		TXT:         parseTXT(e.Text),
		Discovered:  time.Now(),
	}
	for _, ip := range e.AddrIPv4 {
		d.AddrV4 = append(d.AddrV4, ip.String())
	}
	for _, ip := range e.AddrIPv6 {
		d.AddrV6 = append(d.AddrV6, ip.String())
	}
	return d
}

func parseTXT(records []string) map[string]string {
	out := make(map[string]string, len(records))
	for _, r := range records {
		idx := strings.IndexByte(r, '=')
		if idx < 0 {
			out[r] = ""
		} else {
			out[r[:idx]] = r[idx+1:]
		}
	}
	return out
}

// validateServiceType enforces the RFC 6763 syntax: "_<service>._<proto>"
// where <proto> is "tcp" or "udp". Empty is rejected. Multi-label types
// (e.g. "_sub._lava-api._tcp" subtypes) are not currently supported.
func validateServiceType(s string) error {
	if s == "" {
		return errors.New("mdns: ServiceType is required")
	}
	parts := strings.Split(s, ".")
	if len(parts) != 2 {
		return fmt.Errorf("mdns: ServiceType %q must be of the form _service._proto (got %d labels)", s, len(parts))
	}
	for i, p := range parts {
		if !strings.HasPrefix(p, "_") || len(p) < 2 {
			return fmt.Errorf("mdns: ServiceType label %d (%q) must start with '_' and have content", i, p)
		}
	}
	proto := parts[1]
	if proto != "_tcp" && proto != "_udp" {
		return fmt.Errorf("mdns: ServiceType protocol label %q must be _tcp or _udp", proto)
	}
	return nil
}
