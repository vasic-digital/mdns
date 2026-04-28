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
