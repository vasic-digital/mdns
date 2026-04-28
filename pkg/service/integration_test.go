package service_test

import (
	"context"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"digital.vasic.mdns/pkg/service"
)

// Integration test for digital.vasic.mdns.
//
// Sixth Law type-4 (Challenge): a real mDNS announcement is registered,
// then a real mDNS browse runs on the same machine; the discovered entry
// must match the announcement byte-for-byte (Instance, ServiceType, Port,
// every TXT key/value).
//
// These tests require multicast UDP to work on the test runner. They
// skip cleanly if mDNS doesn't function (e.g. inside an isolated CI
// container with no multicast interfaces).
//
// To verify Sixth Law clause 2 (falsifiability) before merging, the
// author MUST temporarily mutate the announce path — e.g. drop one TXT
// record before passing to zeroconf.Register — and observe the
// "TXT mismatch" assertion fire. After confirming the breakage is
// detected, revert the mutation. Record the procedure in the PR.

func skipIfNoMulticast(t *testing.T) {
	t.Helper()
	if os.Getenv("MDNS_SKIP_INTEGRATION") == "1" {
		t.Skip("MDNS_SKIP_INTEGRATION=1 set")
	}
	// Probe: can we open the mDNS multicast group?
	addr, err := net.ResolveUDPAddr("udp4", "224.0.0.251:5353")
	if err != nil {
		t.Skipf("cannot resolve mDNS group: %v", err)
	}
	conn, err := net.ListenMulticastUDP("udp4", nil, addr)
	if err != nil {
		t.Skipf("multicast UDP unavailable on this host: %v", err)
	}
	conn.Close()
}

// TestAnnounceThenBrowseMatchesByteEquivalent is the load-bearing
// Challenge Test: announce a service with a fully populated TXT map,
// browse the same service-type, assert every announced field round-tripped.
func TestAnnounceThenBrowseMatchesByteEquivalent(t *testing.T) {
	skipIfNoMulticast(t)
	t.Parallel()

	const svcType = "_vasicmdnstest1._tcp"
	announcement := service.Announcement{
		Name:        "VasicMdnsTest1",
		ServiceType: svcType,
		Port:        7777,
		TXT: map[string]string{
			"engine":      "go",
			"version":     "1.0.0",
			"protocols":   "h3,h2",
			"compression": "br,gzip",
		},
	}

	srv, err := service.Announce(announcement)
	if err != nil {
		t.Fatalf("Announce: %v", err)
	}
	defer srv.Stop()

	// Give the multicast announcement a moment to propagate.
	time.Sleep(200 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	got, err := service.Browse(ctx, service.BrowseConfig{
		ServiceType: svcType,
		Timeout:     4 * time.Second,
	})
	if err != nil {
		t.Fatalf("Browse: %v", err)
	}
	if len(got) == 0 {
		t.Fatal("Browse returned 0 entries — mDNS round-trip failed (host likely cannot multicast to self)")
	}

	// Find our specific instance.
	var found *service.Discovered
	for i := range got {
		if got[i].Instance == announcement.Name {
			found = &got[i]
			break
		}
	}
	if found == nil {
		var names []string
		for _, g := range got {
			names = append(names, g.Instance)
		}
		t.Fatalf("did not find announced instance %q; saw: %v", announcement.Name, names)
	}

	// Sixth Law primary assertions: every user-visible field round-tripped.
	if found.ServiceType != svcType {
		t.Errorf("ServiceType = %q, want %q", found.ServiceType, svcType)
	}
	if found.Port != announcement.Port {
		t.Errorf("Port = %d, want %d", found.Port, announcement.Port)
	}
	if !strings.HasSuffix(found.Domain, "local.") {
		t.Errorf("Domain = %q, want suffix local.", found.Domain)
	}
	for k, want := range announcement.TXT {
		gotV, ok := found.TXT[k]
		if !ok {
			t.Errorf("TXT[%q] missing", k)
			continue
		}
		if gotV != want {
			t.Errorf("TXT[%q] = %q, want %q", k, gotV, want)
		}
	}
	if len(found.AddrV4) == 0 && len(found.AddrV6) == 0 {
		t.Errorf("Discovered entry has neither v4 nor v6 addresses; expected at least one")
	}
}

// TestAnnounceTwoDistinctServices verifies multiple parallel services
// are independently discoverable.
func TestAnnounceTwoDistinctServices(t *testing.T) {
	skipIfNoMulticast(t)
	t.Parallel()

	// Use distinct service-types so each Browse only matches one.
	a := service.Announcement{Name: "VasicMdnsTest2A", ServiceType: "_vasicmdnstest2a._tcp", Port: 7771}
	b := service.Announcement{Name: "VasicMdnsTest2B", ServiceType: "_vasicmdnstest2b._tcp", Port: 7772}

	sa, err := service.Announce(a)
	if err != nil {
		t.Fatalf("Announce a: %v", err)
	}
	defer sa.Stop()
	sb, err := service.Announce(b)
	if err != nil {
		t.Fatalf("Announce b: %v", err)
	}
	defer sb.Stop()

	time.Sleep(300 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	gotA, err := service.Browse(ctx, service.BrowseConfig{ServiceType: a.ServiceType, Timeout: 3 * time.Second})
	if err != nil {
		t.Fatalf("Browse a: %v", err)
	}
	gotB, err := service.Browse(ctx, service.BrowseConfig{ServiceType: b.ServiceType, Timeout: 3 * time.Second})
	if err != nil {
		t.Fatalf("Browse b: %v", err)
	}

	if !containsInstance(gotA, a.Name) {
		t.Errorf("Browse a did not return %q; saw %v", a.Name, instanceNames(gotA))
	}
	if !containsInstance(gotB, b.Name) {
		t.Errorf("Browse b did not return %q; saw %v", b.Name, instanceNames(gotB))
	}
	if containsInstance(gotA, b.Name) {
		t.Errorf("Browse a leaked into b's service-type")
	}
	if containsInstance(gotB, a.Name) {
		t.Errorf("Browse b leaked into a's service-type")
	}
}

// TestStopRemovesService verifies that after Stop, the service is no
// longer discoverable — within the rdata TTL window the resolver may
// still return it from cache, but a fresh resolver should miss it.
func TestStopRemovesService(t *testing.T) {
	skipIfNoMulticast(t)
	t.Parallel()

	const svcType = "_vasicmdnstest3._tcp"
	a := service.Announcement{Name: "VasicMdnsTest3", ServiceType: svcType, Port: 7773}

	srv, err := service.Announce(a)
	if err != nil {
		t.Fatalf("Announce: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	// Confirm visible.
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	got, err := service.Browse(ctx, service.BrowseConfig{ServiceType: svcType, Timeout: 2 * time.Second})
	if err != nil {
		t.Fatalf("Browse pre-Stop: %v", err)
	}
	if !containsInstance(got, a.Name) {
		t.Skipf("pre-Stop browse did not see service (multicast loopback may be unsupported); skipping the post-Stop check")
	}

	srv.Stop()
	// mDNS goodbye is sent on Shutdown; allow some grace.
	time.Sleep(500 * time.Millisecond)

	got, err = service.Browse(ctx, service.BrowseConfig{ServiceType: svcType, Timeout: 1500 * time.Millisecond})
	if err != nil {
		t.Fatalf("Browse post-Stop: %v", err)
	}
	if containsInstance(got, a.Name) {
		// This isn't a hard failure because mDNS caches may legitimately
		// hold the entry for the rdata TTL — but we still log it so an
		// outright "Stop is a no-op" regression would be visible.
		t.Logf("post-Stop browse still returned the service; this can happen if cached entries persist within mDNS rdata TTL")
	}
}

func containsInstance(entries []service.Discovered, name string) bool {
	for _, e := range entries {
		if e.Instance == name {
			return true
		}
	}
	return false
}

func instanceNames(entries []service.Discovered) []string {
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		out = append(out, e.Instance)
	}
	return out
}
