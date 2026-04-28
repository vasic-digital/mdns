# CLAUDE.md — digital.vasic.mdns

This file guides Claude Code and other agents working in this repository.

## Module purpose

RFC 6762/6763 mDNS service announcement and discovery. Wraps
`grandcat/zeroconf` with strict validation and a narrow public surface
so consumers don't need to deal with TXT-string formatting, multicast
plumbing, or service-type syntax mistakes.

## Inherited rules (non-negotiable)

See `CONSTITUTION.md`.

- Sixth Law — Real User Verification.
- Local-Only CI/CD (`scripts/ci.sh` is the entry point).
- Decoupled Reusable Architecture.

## Layout

```
.
├── pkg/service/                    # public API (Announcement, Service, Browse, Discovered)
│   ├── service.go
│   ├── service_test.go             # unit tests (validation only — no network)
│   └── integration_test.go         # mDNS round-trip Challenge Test
├── scripts/ci.sh                   # local CI gate
├── CONSTITUTION.md
├── README.md
├── CLAUDE.md                       # this file
└── AGENTS.md
```

## Things to avoid

- Re-exporting `zeroconf` types in our public API.
- Adding "convenience" wrappers around stdlib `net` for non-mDNS work.
- Adding hosted CI configuration. Use `scripts/ci.sh`.
- Importing anything Lava-specific or other-consumer-specific.

## When changing the announce or browse path

The Sixth Law makes Challenge Tests load-bearing. If you modify the
announce/browse logic in `pkg/service/service.go`, you MUST:

1. Confirm `TestAnnounceThenBrowseMatchesByteEquivalent` passes.
2. Deliberately mutate the production code (e.g. drop one TXT entry
   before passing to `zeroconf.Register`, or zero out the Port field).
3. Re-run the integration test and confirm a clear assertion failure.
4. Revert the mutation.
5. Document the mutation and observed failure in the PR description.

Skipping this falsifiability rehearsal is a Sixth Law violation.

## Multicast in CI

Integration tests need multicast UDP on at least one network interface.
On hosts that block multicast (isolated containers, some Wi-Fi NICs),
the tests skip cleanly — they MUST NOT fail just because multicast is
unavailable. The skip message points the operator at how to fix it
(e.g. enable a loopback bridge, use host networking).
