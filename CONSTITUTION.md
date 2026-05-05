# digital.vasic.mdns — Constitution

This module inherits the non-negotiable `vasic-digital` rules.

## Inherited rules

- **Sixth Law — Real User Verification.** Every test MUST be provably
  falsifiable, assert on user-visible state, and traverse the production
  code path. Challenge Tests are the load-bearing release gate.
- **Local-Only CI/CD.** No GitHub Actions, GitLab pipelines, CircleCI,
  or any other hosted CI service. `scripts/ci.sh` is the canonical
  entry point.
- **Decoupled Reusable Architecture.** This module is product-agnostic.
  No Lava-, HelixAgent-, or other-consumer-specific code or assumptions.

## Module-specific rules

- **Public API surface.** `service.Announcement`, `service.Announce`,
  `service.Service.Stop`, `service.BrowseConfig`, `service.Browse`,
  `service.Discovered`. New fields on `Announcement` and `Discovered`
  may be added (additive); removal/rename is a major-version event.

- **No re-export of `zeroconf` types.** Consumers who need zeroconf-level
  control import zeroconf directly. We don't wrap, alias, or shadow.

- **Service-type validation in Validate, not at announce time.**
  Strict syntax check on `_<service>._<tcp|udp>` form; reject anything
  else with a clear error before any network operation. RFC 6763
  subtypes (`_sub._main._proto`) are not supported in v0.

- **TXT keys MUST NOT contain `=` or NUL.** Validated in Announcement.Validate
  and rejected by Validate; TXT values MUST NOT contain NUL. Empty TXT
  is fine.

## Verification before tagging

Before any release tag is cut on this module:

1. `scripts/ci.sh` MUST be run on the exact commit being tagged and pass.
2. `TestAnnounceThenBrowseMatchesByteEquivalent` MUST be falsified at
   least once (e.g. drop a TXT key in the announce path; observe the
   "TXT[…] missing" assertion fire). The PR description MUST document
   this rehearsal.
3. The integration tests SHOULD run on a host with working multicast
   (skip is permitted but not preferred).

## Mirror policy

- `git@github.com:vasic-digital/Mdns.git` (primary)
- `git@gitlab.com:vasic-digital/Mdns.git` (mirror)

GitFlic and GitVerse are not currently in this module's mirror set.


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

## Seventh Law inheritance (Anti-Bluff Enforcement, 2026-04-30)

In addition to the Sixth Law above, this submodule inherits Lava's **Seventh Law — Tests MUST Confirm User-Reachable Functionality (Anti-Bluff Enforcement)** when consumed by the Lava project (`vasic-digital/Lava`). The Seventh Law was added to Lava's `CLAUDE.md` on 2026-04-30 to mechanically enforce the Sixth Law: every test commit MUST carry a Bluff-Audit stamp (mutation/observed-failure/reverted protocol); every feature MUST pass a real-stack verification gate; release tags MUST be preceded by a real-device attestation; forbidden test patterns (mocking the SUT, verification-only assertions, ignored tests without follow-up, build-success-as-only-assertion) are pre-push-rejected; a recurring bluff hunt and a bluff discovery protocol apply.

The authoritative verbatim text lives in the parent Lava `CLAUDE.md` under "Seventh Law — Tests MUST Confirm User-Reachable Functionality (Anti-Bluff Enforcement)". This submodule MAY add stricter clauses but MUST NOT relax any of the seven Seventh-Law clauses. Both the submodule's own anti-bluff rules and Lava's Sixth + Seventh Laws are binding when consumed by Lava; the stricter of the two applies.

## Clause 6.L — Anti-Bluff Functional Reality Mandate (Operator's Standing Order)

Inherited verbatim from parent Lava `/CLAUDE.md` §6.L. The operator has invoked this mandate **NINE TIMES** across two working days. The 9th invocation (2026-05-05 late evening): "Make sure that all existing tests and Challenges do work in anti-bluff manner — they MUST confirm that all tested codebase really works as expected!"

Every test, every Challenge Test, every CI gate added to or maintained in this submodule MUST do exactly one job: confirm the feature it claims to cover actually works for an end user, end-to-end, on the gating matrix. CI green is necessary, NEVER sufficient. Tests must guarantee the product works — anything else is theatre.

Inheritance is recursive. Sub-submodules MAY paste this clause verbatim; they MUST NOT abbreviate or relax it.
