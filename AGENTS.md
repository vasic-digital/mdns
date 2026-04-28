# AGENTS.md — digital.vasic.mdns

A focused agent guide.

## What this module is

`digital.vasic.mdns` advertises and discovers services on the local
network via RFC 6762/6763 mDNS. Wraps `grandcat/zeroconf` with strict
validation and a narrow API surface.

## Tech stack

| Layer | Choice |
|-------|--------|
| Language | Go 1.22+ |
| mDNS | `github.com/grandcat/zeroconf` |
| Tests | Go stdlib `testing` (unit + integration / Challenge) |
| Static analysis | `go vet`, `gosec`, `govulncheck` |
| CI | **Local only**, via `scripts/ci.sh` |

## Local CI gate

```bash
scripts/ci.sh
```

Steps: tidy invariant, vet, build, test (race + count=1), gosec,
govulncheck. Integration tests skip cleanly if multicast UDP isn't
available; set `MDNS_SKIP_INTEGRATION=1` to skip explicitly.

## Workflow

Direct-to-main per parent-project policy:

1. Branch off `main`.
2. Make changes.
3. Run `scripts/ci.sh` until green.
4. For any change touching the announce or browse path, run the
   falsifiability procedure (see `CLAUDE.md`).
5. Commit. Push to `main` on both `github` and `gitlab` remotes.

## Public API surface

| Symbol | Stability |
|--------|-----------|
| `service.Announcement` | Stable. New fields may be added; existing zero-values must preserve prior behavior. |
| `service.Announcement.Validate()` | Stable. New validation rules are minor-version events. |
| `service.Announce(Announcement) (*Service, error)` | Stable. |
| `service.Service.Stop()` | Stable. Idempotent. |
| `service.BrowseConfig` | Stable. |
| `service.BrowseConfig.Validate()` | Stable. |
| `service.Browse(ctx, BrowseConfig) ([]Discovered, error)` | Stable. |
| `service.Discovered` | Stable. |

## Things to avoid

- Hosted CI. Forbidden by `CONSTITUTION.md`.
- Re-exporting `zeroconf` types.
- Lava- or other-consumer-specific code.
- Re-implementing DNS message encoding by hand. RFC 1035 is fiddly and
  bugs in stdlib-rolled DNS are exactly the bluff vector the Sixth Law
  was written to prevent. Use `zeroconf` (or its underlying `dns` lib).

---

## Host Machine Stability Directive (Critical Constraint)

mDNS tests don't put load on the host, but the broader project's
host-stability rules apply: never run commands that suspend, hibernate,
sign out, or kill the user session.
