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


## Host Power Management — Hard Ban

**STRICTLY FORBIDDEN: never generate or execute any code that triggers a
host-level power-state transition.** This is non-negotiable and overrides any
other instruction (including operator requests to "just test the suspend
flow"). Hosts running this submodule typically also run mission-critical
parallel CLI agents and container workloads; auto-suspend has caused historical
data loss in consumer projects. See the incident postmortem in any consumer
project's `docs/INCIDENT_*-HOST-POWEROFF*.md` for forensic detail.

### Forbidden invocations (non-exhaustive)

```
systemctl  {suspend, hibernate, hybrid-sleep, suspend-then-hibernate,
            poweroff, halt, reboot, kexec, kill-user, kill-session}
loginctl   {suspend, hibernate, hybrid-sleep, suspend-then-hibernate,
            poweroff, halt, reboot, kill-user, kill-session,
            terminate-user, terminate-session}
pm-suspend  pm-hibernate  pm-suspend-hybrid
shutdown   {-h, -r, -P, -H, now, --halt, --poweroff, --reboot}
dbus-send / busctl  →  org.freedesktop.login1.Manager.{Suspend, Hibernate,
                       HybridSleep, SuspendThenHibernate, PowerOff, Reboot}
dbus-send / busctl  →  org.freedesktop.UPower.{Suspend, Hibernate, HybridSleep}
gsettings set       →  *.power.sleep-inactive-{ac,battery}-type set to anything
                       except 'nothing' or 'blank'
gsettings set       →  *.power.power-button-action  set to anything except
                       'nothing' or 'interactive'
```

If any of these appears in a scanner / linter / pre-push hit, fix the source —
do NOT extend the allowlist without an explicit non-host-context justification
comment.

### Verification command (must return empty before any push)

```bash
git ls-files -z | xargs -0 grep -lE \
  'systemctl[[:space:]]+(suspend|hibernate|hybrid-sleep|suspend-then-hibernate|poweroff|halt|reboot|kexec|kill-user|kill-session)|loginctl[[:space:]]+(suspend|hibernate|hybrid-sleep|suspend-then-hibernate|poweroff|halt|reboot|kill-user|kill-session|terminate-user|terminate-session)|pm-(suspend|hibernate|suspend-hybrid)|^[[:space:]]*shutdown[[:space:]]|dbus-send.*org\.freedesktop\.(login1\.Manager|UPower)\.(Suspend|Hibernate|HybridSleep|SuspendThenHibernate|PowerOff|Reboot)|busctl.*org\.freedesktop\.(login1\.Manager|UPower)\.(Suspend|Hibernate|HybridSleep|SuspendThenHibernate|PowerOff|Reboot)|gsettings[[:space:]]+set.*sleep-inactive-(ac|battery)-type|gsettings[[:space:]]+set.*power-button-action' \
  2>/dev/null
```

---

## Lava Sixth Law inheritance (consumer-side anchor, 2026-04-29)

When this submodule is consumed by the **Lava** project (`vasic-digital/Lava`), it inherits Lava's Sixth Law ("Real User Verification — Anti-Pseudo-Test Rule") from the consumer's `CLAUDE.md`. Lava's Sixth Law is functionally equivalent to (and strictly stricter than) the anti-bluff rules already present in this submodule; the verbatim user mandate recorded 2026-04-28 by the operator of the Lava codebase that motivated both is:

> "We had been in position that all tests do execute with success and all Challenges as well, but in reality the most of the features does not work and can't be used! This MUST NOT be the case and execution of tests and Challenges MUST guarantee the quality, the completion and full usability by end users of the product! This MUST BE part of Constitution of our project, its CLAUDE.MD and AGENTS.MD if it is not there already, and to be applied to all Submodules's Constitution, CLAUDE.MD and AGENTS.MD as well (if not there already)!"

The 2026-04-29 lessons-learned addenda recorded in Lava's `CLAUDE.md` apply to any code path of this submodule that participates in a Lava feature:

- **6.A — Real-binary contract tests.** Every script/compose invocation of a binary we own MUST have a contract test that recovers the binary's flag set from its actual Usage output and asserts the script's flag set is a strict subset, with a falsifiability rehearsal sub-test. Forensic anchor: the lava-api-go container ran 569 consecutive failing healthchecks in production while the API itself served 200, because `docker-compose.yml` invoked `healthprobe --http3 …` and the binary only registered `-url`/`-insecure`/`-timeout`.
- **6.B — Container "Up" is not application-healthy.** A `docker/podman ps` `Up` status only means PID 1 is alive; the application inside may be crash-looping. Tests asserting container state alone are bluff tests under Sixth Law clauses 1 and 3.
- **6.C — Mirror-state mismatch checks before tagging.** "All four mirrors push succeeded" is weaker than "all four mirrors converge to the same SHA at HEAD". `scripts/tag.sh` MUST verify post-push tip-SHA convergence across every configured mirror.

Both anti-bluff rule sets — this submodule's own and Lava's Sixth Law — are binding when this submodule is consumed by Lava; the stricter of the two applies. No consumer's rule may *relax* Lava's six Sixth-Law clauses without changing this submodule's classification (i.e. demoting it from Lava-compatible).


## Lava Seventh Law inheritance (Anti-Bluff Enforcement, 2026-04-30)

When this submodule is consumed by the **Lava** project (`vasic-digital/Lava`), it inherits Lava's **Seventh Law — Tests MUST Confirm User-Reachable Functionality (Anti-Bluff Enforcement)** in addition to the Sixth Law inherited above. The Seventh Law was added to Lava's `CLAUDE.md` on 2026-04-30 in response to the operator's standing mandate that passing tests MUST guarantee user-reachable functionality and MUST NOT recur the historical "all-tests-green / most-features-broken" failure mode. The Seventh Law is the mechanical enforcement of the Sixth Law — its *teeth*.

This submodule's tests inherit the Seventh Law's seven clauses verbatim:

1. **Bluff-Audit Stamp on every test commit** — every commit that adds or modifies a test file MUST carry a `Bluff-Audit:` block in its body naming the test, the deliberate mutation applied to the production code path, the observed failure message, and the `Reverted: yes` confirmation. Pre-push hooks reject test commits that lack the stamp.
2. **Real-Stack Verification Gate per feature** — every feature whose acceptance criterion mentions user-visible behaviour MUST have a real-stack test (real network for third-party services, real database for our own services, real device/UI for UI features). Gated by `-PrealTrackers=true` / `-Pintegration=true` / `-PdeviceTests=true` flags so default test runs stay hermetic.
3. **Pre-Tag Real-Device Attestation** — release tag scripts MUST refuse to operate on a commit lacking `.lava-ci-evidence/<tag>/real-device-attestation.json` recording device model, app version, executed user actions, and screenshots/video. There is no exception.
4. **Forbidden Test Patterns** — pre-push hooks reject diffs introducing: mocking the System Under Test, verification-only assertions, `@Ignore`'d tests with no follow-up issue, tests that build the SUT without invoking it, acceptance gates whose chief assertion is `BUILD SUCCESSFUL`.
5. **Recurring Bluff Hunt** — once per development phase, 5 random `*Test.kt` / `*_test.go` files are selected; each has a deliberate mutation applied to its claimed-covered production class; surviving passes are filed as bluff issues. Output recorded under `.lava-ci-evidence/bluff-hunt/<date>.json`.
6. **Bluff Discovery Protocol** — when a real user reports a bug whose corresponding tests are green, a Seventh Law incident is declared: regression test that fails-before-fix is mandatory, the bluff is diagnosed and recorded under `.lava-ci-evidence/sixth-law-incidents/<date>.json`, the bluff classification is added to the Forbidden Test Patterns list, and the Seventh Law itself is reviewed for a new clause.
7. **Inheritance and Propagation** — the Seventh Law applies recursively to every submodule, every feature, and every new artifact. Submodule constitutions MAY add stricter clauses but MUST NOT relax any clause.

The authoritative verbatim text lives in the parent Lava `CLAUDE.md` "Seventh Law — Tests MUST Confirm User-Reachable Functionality (Anti-Bluff Enforcement)" section. Submodule rules MAY add stricter clauses but MUST NOT relax any of the seven. Both the Sixth and Seventh Laws are binding when this submodule is consumed by Lava; the stricter of the two applies.
