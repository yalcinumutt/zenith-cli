---
status: accepted
---

# The desktop app lives in this module under `desktop/`, released on its own tag stream

The Wails shell and the React app live in `desktop/` (shell at `desktop/`, frontend at `desktop/frontend/`) inside the existing Go module, because `internal/core` can only be imported from within the module. The CLI keeps the `v*` tag and its GoReleaser OSS release. The desktop app releases on `desktop-v*` tags as separate GitHub releases, built by a three-OS matrix (`wails3 package` on macOS, Windows and Linux runners) and uploaded with `gh release upload`. GoReleaser Pro is not used.

`zenith update` picks the first release asset ending in `<os>_<arch>`, so desktop installers must never be selectable by it. We split the release streams and also add a `Config.Filters` regex (`^zenith-cli_`) in `cmd/update.go`, so a future asset name cannot break the CLI updater.

v1 is a free beta from the public MIT repo with no payment or licensing code. Any commercial model is decided after v1 and does not need relicensing.

## Considered options

- **Separate repo for the desktop app:** would force `internal/core` into a public package for no gain.
- **One release for CLI and desktop:** the `zenith update` asset-matching hazard, and a single failing platform would block both products.
- **GoReleaser Pro (prebuilt + split):** one pipeline, but a paid dependency for a free OSS project.

## Consequences

- The Wails v3 build from a sub-directory of an existing module is unverified; the first implementation ticket is a spike that proves it.
- The GitHub owner is `yalcinumutt`; the module path, README links and the self-update slug are corrected to match.
