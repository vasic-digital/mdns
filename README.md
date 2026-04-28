# digital.vasic.mdns

A generic Go module for **RFC 6762/6763 mDNS** service announcement and
discovery. Drop-in LAN service registration for any TCP/UDP service —
HTTP/3 servers, gRPC services, custom binary protocols. Compatible with
Android `NsdManager`, Apple Bonjour, and any other mDNS client.

## Why

The Go ecosystem has no first-party mDNS library. `grandcat/zeroconf` is
the canonical third-party choice, but its surface is broad and its
defaults aren't always what consumers want (TXT records as parallel
string slices, no validation, no integrated lifecycle). This module
narrows the API to a single `Announce` / `Browse` pair with strict
validation of service-type strings (`_<service>._<tcp|udp>`), TXT records
typed as `map[string]string`, and explicit Start/Stop lifecycle.

## Installation

```bash
go get digital.vasic.mdns
```

## Quick start

```go
package main

import (
    "context"
    "log"
    "time"

    "digital.vasic.mdns/pkg/service"
)

func main() {
    // Announce ourselves on the LAN.
    srv, err := service.Announce(service.Announcement{
        Name:        "Lava API on living-room-pi",
        ServiceType: "_lava-api._tcp",
        Port:        8443,
        TXT: map[string]string{
            "engine":      "go",
            "version":     "2.0.0",
            "protocols":   "h3,h2",
            "compression": "br,gzip",
        },
    })
    if err != nil {
        log.Fatal(err)
    }
    defer srv.Stop()

    // Browse the LAN for any other Lava API instances.
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    found, err := service.Browse(ctx, service.BrowseConfig{
        ServiceType: "_lava-api._tcp",
        Timeout:     3 * time.Second,
    })
    if err != nil {
        log.Fatal(err)
    }
    for _, d := range found {
        log.Printf("found %s at %s:%d (engine=%s)", d.Instance, d.HostName, d.Port, d.TXT["engine"])
    }

    select {} // block until SIGINT
}
```

## Constitutional discipline

Inherits the `vasic-digital` rules — see `CONSTITUTION.md`:

- **Sixth Law (Real User Verification).** Every test is provably
  falsifiable and asserts on user-visible state. The Challenge Test in
  `integration_test.go::TestAnnounceThenBrowseMatchesByteEquivalent`
  registers a real mDNS service, performs a real mDNS browse, and asserts
  that every announced TXT key/value, port, and service-type round-tripped.
- **Local-Only CI/CD.** No hosted CI configuration ships in this repo.
  `scripts/ci.sh` is the single local entry point.
- **Decoupled Reusable Architecture.** No consumer-specific code. The
  module's only third-party runtime dependency is `grandcat/zeroconf`.

## Testing

```bash
scripts/ci.sh                         # full local CI gate
go test ./...                         # unit tests only (fast)
go test -race -count=1 ./...          # all tests with race detector
MDNS_SKIP_INTEGRATION=1 go test ./... # skip multicast-dependent tests
```

The integration tests require multicast UDP on at least one interface.
On hosts where multicast is unavailable (some isolated container
runtimes), they skip with a clear message.

## License

MIT — see `LICENSE`.
