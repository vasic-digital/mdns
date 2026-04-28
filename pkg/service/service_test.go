package service

import (
	"strings"
	"testing"
)

// Unit tests for validation logic. No network, no mDNS.

func TestAnnouncementValidate(t *testing.T) {
	t.Parallel()
	good := Announcement{
		Name:        "Lava API",
		ServiceType: "_lava-api._tcp",
		Port:        8443,
	}
	if err := good.Validate(); err != nil {
		t.Fatalf("good announcement rejected: %v", err)
	}

	cases := []struct {
		name    string
		mutate  func(*Announcement)
		wantErr string
	}{
		{
			name:    "missing Name",
			mutate:  func(a *Announcement) { a.Name = "" },
			wantErr: "Name is required",
		},
		{
			name:    "missing ServiceType",
			mutate:  func(a *Announcement) { a.ServiceType = "" },
			wantErr: "ServiceType is required",
		},
		{
			name:    "ServiceType wrong shape (no proto)",
			mutate:  func(a *Announcement) { a.ServiceType = "_lava-api" },
			wantErr: "must be of the form",
		},
		{
			name:    "ServiceType wrong proto label",
			mutate:  func(a *Announcement) { a.ServiceType = "_lava-api._raw" },
			wantErr: "must be _tcp or _udp",
		},
		{
			name:    "ServiceType missing leading underscore",
			mutate:  func(a *Announcement) { a.ServiceType = "lava-api._tcp" },
			wantErr: "must start with '_'",
		},
		{
			name:    "Port zero",
			mutate:  func(a *Announcement) { a.Port = 0 },
			wantErr: "out of range",
		},
		{
			name:    "Port negative",
			mutate:  func(a *Announcement) { a.Port = -1 },
			wantErr: "out of range",
		},
		{
			name:    "Port too high",
			mutate:  func(a *Announcement) { a.Port = 70000 },
			wantErr: "out of range",
		},
		{
			name:    "TXT key with =",
			mutate:  func(a *Announcement) { a.TXT = map[string]string{"k=bad": "v"} },
			wantErr: "TXT key",
		},
		{
			name:    "TXT key with NUL",
			mutate:  func(a *Announcement) { a.TXT = map[string]string{"k\x00": "v"} },
			wantErr: "TXT key",
		},
		{
			name:    "TXT value with NUL",
			mutate:  func(a *Announcement) { a.TXT = map[string]string{"k": "v\x00"} },
			wantErr: "TXT value",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a := good
			a.TXT = nil
			tc.mutate(&a)
			err := a.Validate()
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected error containing %q, got %v", tc.wantErr, err)
			}
		})
	}
}

func TestBrowseConfigValidate(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		typ     string
		wantErr string
	}{
		{"empty", "", "is required"},
		{"valid tcp", "_lava-api._tcp", ""},
		{"valid udp", "_x._udp", ""},
		{"missing leading underscore", "lava-api._tcp", "must start with '_'"},
		{"too many labels", "_a._b._tcp", "must be of the form"},
		{"bad proto", "_a._foo", "must be _tcp or _udp"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := BrowseConfig{ServiceType: tc.typ}.Validate()
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("expected nil, got %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected error containing %q, got %v", tc.wantErr, err)
			}
		})
	}
}

func TestParseTXT(t *testing.T) {
	t.Parallel()
	got := parseTXT([]string{"k=v", "engine=go", "version=1.0.0", "flag"})
	if got["k"] != "v" {
		t.Errorf("k = %q, want v", got["k"])
	}
	if got["engine"] != "go" {
		t.Errorf("engine = %q, want go", got["engine"])
	}
	if got["version"] != "1.0.0" {
		t.Errorf("version = %q, want 1.0.0", got["version"])
	}
	if v, ok := got["flag"]; !ok || v != "" {
		t.Errorf("flag entry = (%q, %v), want (\"\", true)", v, ok)
	}
}

func TestParseTXTHandlesEqualsInValue(t *testing.T) {
	t.Parallel()
	got := parseTXT([]string{"path=/api/v1?foo=bar"})
	if got["path"] != "/api/v1?foo=bar" {
		t.Errorf("path = %q, want %q", got["path"], "/api/v1?foo=bar")
	}
}

func TestStopBeforeAnnounceIsSafe(t *testing.T) {
	t.Parallel()
	s := &Service{}
	s.Stop() // no panic
	s.Stop()
}
