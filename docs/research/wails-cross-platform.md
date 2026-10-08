# Wails v2 vs v3 and cross-platform build, signing and release

Resolves wayfinder ticket #3 (child of map #2). Research date: 2026-10-08. Primary sources only; each claim cites its source. Where a conclusion is my inference rather than a source statement, it is marked **(inference)**. No product decisions are made here.

## Answer in brief

1. **Versions.** Wails v2 is the stable line (latest release `v2.14.0`, 2026-08-10). Wails v3 is still in **beta** (`v3.0.0-beta.28`, 2026-10-05; nightly-automated betas); its own docs call the API stable but the software pre-release. Zenith's feature needs (tray, global hotkey) exist in v3 and **not in v2**. See section 1 and 5.
2. **CGO.** `mattn/go-sqlite3` needs `CGO_ENABLED=1` plus gcc. Wails v3 already needs CGO on macOS and Linux (the webview), so mattn adds no new burden there. On Windows v3 builds without CGO by default, so mattn would be the only reason to need a C toolchain there; `modernc.org/sqlite` (pure Go) avoids that. See section 3.
3. **Release pipeline.** The existing GoReleaser setup (OSS features, `CGO_ENABLED=0`, ubuntu runner, Go 1.22) cannot build a Wails app. Several GoReleaser features that would help (prebuilt builder, split/merge, DMG, app bundles, NSIS, MSI, native notarization) are **GoReleaser Pro only**. See section 4.
4. **`zenith update` hazard.** The self-updater picks the first release asset whose name ends in `<goos>_<goarch>[.ext]`, with no check on the binary name. Desktop installers attached to the same release can be picked up by mistake. See section 2.
5. **Signing.** macOS needs a paid Apple Developer account and a Mac; Windows signing needs a CA-issued cert or Microsoft's Artifact Signing service (country/entity restrictions apply); Linux packages are PGP-signed. See section 4.

## 1. Version status

| Fact | Source |
| --- | --- |
| Latest v2 release is `v2.14.0`, published 2026-08-10, not a prerelease; `v2.13.0` on 2026-07-06. | [wailsapp/wails releases](https://github.com/wailsapp/wails/releases) (queried via `gh release list`) |
| v2 has an automated weekly release workflow, i.e. the v2 line is actively maintained. | [`.github/workflows/weekly-release-v2.yml`](https://github.com/wailsapp/wails/blob/master/.github/workflows/weekly-release-v2.yml) |
| Latest v3 release is `v3.0.0-beta.28` (2026-10-05); beta.22 through beta.28 were published between 2026-09-14 and 2026-10-05. | [GitHub Releases API](https://api.github.com/repos/wailsapp/wails/releases) |
| v3 releases are "automated nightly release[s]" and carry the text: "The API is stable, but you may still encounter issues before the final 3.0 release." and "This is pre-release software." | [v3.0.0-beta.28 release notes](https://github.com/wailsapp/wails/releases/tag/v3.0.0-beta.28) |
| v3 roadmap page: "Current Status: Beta". Beta contract covers desktop apps on Windows (amd64, arm64; WebView2), macOS (Intel, Apple Silicon), Linux (amd64, arm64; GTK4 + WebKitGTK 6.0 default). Go 1.25+ required for development. Android/iOS are experimental. "prerelease defects and explicitly announced changes may still be corrected before v3.0.0." | [docs/mpress/content/status.md](https://github.com/wailsapp/wails/blob/master/docs/mpress/content/status.md) |
| Linux legacy path: GTK3 + WebKit2GTK 4.1 via `-tags gtk3` is supported through v3.0.x and **removed in v3.1**. It exists for Ubuntu 22.04, Debian 12, Fedora <= 39, RHEL 9.x. | [guides/build/linux.md](https://github.com/wailsapp/wails/blob/master/docs/mpress/content/guides/build/linux.md) |
| v2 supported platforms: Windows 10/11 amd64/arm64, macOS 10.15+ (amd64) / 11.0+ (arm64), Linux amd64/arm64. Needs Go 1.21+, Node 15+, WebView2 on Windows, `xcode-select --install` on macOS, gcc + libgtk3 + libwebkit on Linux (`-tags webkit2_41` on distros without webkit2gtk-4.0, e.g. Ubuntu 24.04). | [website/docs/gettingstarted/installation.mdx](https://github.com/wailsapp/wails/blob/master/website/docs/gettingstarted/installation.mdx) |
| v2 module declares `go 1.25.0` in `v2/go.mod` (same as Zenith's `go.mod`). | [v2/go.mod](https://github.com/wailsapp/wails/blob/master/v2/go.mod) |

### Feature gap relevant to Zenith's later extension points (map #2 "Not yet specified")

| Capability | v2 | v3 |
| --- | --- | --- |
| System tray / menu bar | No tray option in `options.App` (`v2/pkg/options/options.go` contains no tray field); the v2 runtime reference lists browser, clipboard, dialog, drag-and-drop, events, log, menu, notification, screen, window only. A `TrayMenu` struct exists in `v2/pkg/menu/tray.go` but is not exposed through the documented options. **(inference: not a supported feature in v2.)** | Documented "unified system tray APIs", macOS menu bar, Windows notification area, Linux StatusNotifierItem. |
| Global (system-wide) hotkeys | Not in the v2 runtime reference. | `app.GlobalShortcut.Register(...)`, built on native OS facilities, no third-party deps. |
| Autostart at login | Not in the v2 runtime reference. | `app.Autostart` (macOS SMAppService/LaunchAgent, Windows `HKCU\...\Run`, Linux `.desktop`). |
| Single-instance lock | Not in the v2 runtime reference. | Documented (Windows mutex, Linux dbus). |
| Notifications | `notification.mdx` in v2 runtime reference. | Documented, with per-platform degradation. |
| In-app updater | none documented | `app.Updater` with a GitHub Releases provider, SHA-256 verification, optional signature key (example `v3/examples/updater`). |

Sources: [v2 options.go](https://github.com/wailsapp/wails/blob/master/v2/pkg/options/options.go), [v2 runtime docs listing](https://github.com/wailsapp/wails/tree/master/website/docs/reference/runtime), [v2 tray.go](https://github.com/wailsapp/wails/blob/master/v2/pkg/menu/tray.go), [systray.md](https://github.com/wailsapp/wails/blob/master/docs/mpress/content/features/menus/systray.md), [global-shortcuts.md](https://github.com/wailsapp/wails/blob/master/docs/mpress/content/features/keyboard/global-shortcuts.md), [autostart/basics.md](https://github.com/wailsapp/wails/blob/master/docs/mpress/content/features/autostart/basics.md), [guides/single-instance.md](https://github.com/wailsapp/wails/blob/master/docs/mpress/content/guides/single-instance.md), [notifications/overview.md](https://github.com/wailsapp/wails/blob/master/docs/mpress/content/features/notifications/overview.md), [v3/examples/updater](https://github.com/wailsapp/wails/blob/master/v3/examples/updater/README.md).

### Tray and global-hotkey behaviour per OS (v3)

| | macOS | Windows | Linux |
| --- | --- | --- | --- |
| Tray | Menu bar item; use template icon (`SetTemplateIcon`), 18-22 px. | Notification area, 16/32 px, survives Explorer restarts. | StatusNotifierItem. GNOME needs an extension for a top-bar tray; KDE, XFCE fine. Docs: "Don't assume tray support - Some Linux DEs don't support it"; there is no `SystemTraySupported()` helper. |
| Global hotkey API | Carbon hot key API; no Accessibility permission needed. Several apps may register the same key (no error). | Win32 `RegisterHotKey`; fails if another app owns it. | X11: direct grab, fails if owned. Wayland: XDG Desktop Portal `GlobalShortcuts`; accelerator is only a *preferred* trigger, the compositor/user decides, `IsRegistered`/`GetAll` report the request, and the portal needs a DE that implements it (recent GNOME, KDE Plasma). |
| Gotcha | `window.Hide()` + `ApplicationShouldTerminateAfterLastWindowClosed: true` quits the app; leave it unset for hide/show hotkeys. | | Right-click on tray icon also ran the click handler until the fix noted in beta.28 (#6018). |

Sources: [systray.md](https://github.com/wailsapp/wails/blob/master/docs/mpress/content/features/menus/systray.md), [global-shortcuts.md](https://github.com/wailsapp/wails/blob/master/docs/mpress/content/features/keyboard/global-shortcuts.md), [beta.28 notes](https://github.com/wailsapp/wails/releases/tag/v3.0.0-beta.28).

## 2. GoReleaser and `zenith update` compatibility

### Current setup (from repo)

- `.goreleaser.yaml`: `CGO_ENABLED=0`; goos linux and darwin (windows commented out); amd64+arm64; archive `{{ .ProjectName }}_{{ .Os }}_{{ .Arch }}` as `tar.gz`; `checksums.txt`; nfpm deb/rpm.
- `.github/workflows/release.yml`: single `ubuntu-latest` job, Go `1.22`, `goreleaser-action@v5`, `version: "~> v2"`, triggered on `v*` tags.
- `cmd/update.go`: `selfupdate.UpdateSelf(v, "yalcinumut/zenith-cli")` using `github.com/rhysd/go-github-selfupdate v1.2.3`, no asset filters configured.

### How the self-updater selects an asset

From the library source (`selfupdate/detect.go`) and README:

- It iterates the repo's releases, skipping drafts and prereleases, and requires a semver-parsable tag.
- For each release it takes the **first asset whose name has suffix** `<runtime.GOOS><sep><runtime.GOARCH><ext>`, where `sep` is `_` or `-` and `ext` is one of `.zip .tar.gz .tgz .gzip .gz .tar.xz .xz` or empty (plus `.exe` variants on Windows). It does **not** check the command-name prefix, unless the caller passes `Config.Filters` regexes.
- The README states the expected naming `{cmd}_{goos}_{goarch}{.ext}`; if archived, the archive must contain an executable named `{cmd}`, `{cmd}_{os}_{arch}` or `{cmd}-{os}-{arch}`.
- The upstream repo's last push is 2023-09-19 (not archived).

Sources: [detect.go](https://github.com/rhysd/go-github-selfupdate/blob/master/selfupdate/detect.go), [README naming rules](https://github.com/rhysd/go-github-selfupdate#naming-rules-of-released-binaries), `cmd/update.go` in this repo.

### Consequences for naming **(inference from the above)**

- If desktop artifacts ship in the **same GitHub release** as the CLI, any desktop asset ending in e.g. `darwin_arm64.zip` or `linux_amd64.tar.gz` can be selected by `zenith update` ahead of the CLI archive, because matching is suffix-only and first-found wins. Names that do not end in `<os>_<arch>[.ext]` (for example `Zenith-1.2.0-macos-arm64.dmg`, `Zenith_1.2.0_amd64.deb`, `.msi`, `-setup.exe`, `.AppImage`) are not matched. Safe options are therefore: keep desktop asset names off that suffix pattern, or put desktop releases in a separate repo/tag stream, or set `Config.Filters` in `cmd/update.go`.
- Wails v3's own tooling defaults produce: NSIS `build/windows/nsis/<AppName>-installer.exe` or MSIX `bin/<AppName>-<arch>.msix`; Linux AppImage, DEB, RPM, Arch under `bin/`; macOS `bin/<AppName>.app` (sources in section 4). Whether those default names collide depends on template output and was not verified.
- The CLI archive name must stay `zenith-cli_<os>_<arch>.tar.gz` (project name `zenith-cli` is the `{{ .ProjectName }}`), per `CLAUDE.md`.
- Wails v3 ships its own updater (`app.Updater`, GitHub Releases provider, SHA256SUMS). Its asset-naming expectations were not read in detail; see open questions.

### GoReleaser capabilities relevant to a Wails app

| Capability | Edition | Source |
| --- | --- | --- |
| `builds` for Go with `CGO_ENABLED`, per-target `overrides`, custom `targets`; no built-in cross C-toolchain | OSS | [builds docs](https://goreleaser.com/customization/builds/go/) |
| `release.extra_files` (attach pre-existing files by glob to the GitHub release) | OSS | [scm/_index.md](https://github.com/goreleaser/goreleaser/blob/main/www/content/customization/publish/scm/_index.md) |
| `builder: prebuilt` (import binaries built elsewhere; documented use case: "build your binaries in different machines due to CGO") | **Pro** | [prebuilt.md](https://github.com/goreleaser/goreleaser/blob/main/www/content/customization/builds/builders/prebuilt.md) |
| `release --split` / merge (build each OS on its own runner; documented for CGO and native Windows/macOS packaging) | **Pro** | [general/partial.md](https://github.com/goreleaser/goreleaser/blob/main/www/content/customization/general/partial.md) |
| `app_bundles` (macOS `.app`, v2.4), `dmg`, `msi`, `nsis` (v2.14) | **Pro** | [app_bundles.md](https://github.com/goreleaser/goreleaser/blob/main/www/content/customization/package/app_bundles.md), [dmg.md](https://github.com/goreleaser/goreleaser/blob/main/www/content/customization/package/dmg.md), [msi.md](https://github.com/goreleaser/goreleaser/blob/main/www/content/customization/package/msi.md), [nsis.md](https://github.com/goreleaser/goreleaser/blob/main/www/content/customization/package/nsis.md) |
| macOS notarize, cross-platform (`notarize.macos`, uses anchore/quill): binaries and universal binaries only; "putting a signed and notarized binary inside a non-notarized `.app` does not work!" | Not labelled Pro on the page (its example uses the Pro distribution) | [notarize docs](https://goreleaser.com/customization/notarize/) |
| macOS notarize, native (`notarize.macos_native`, `codesign`+`xcrun`, needed for `.app` in DMG/PKG) | **Pro** | [notarize docs](https://goreleaser.com/customization/notarize/) |

### Implication **(inference)**

With OSS GoReleaser, the CLI keeps releasing exactly as today. The desktop app would be built by Wails' own tooling (`wails3 build` / `wails3 package`) on per-OS runners and its outputs uploaded to a release by a separate job (for example via `extra_files` or the GitHub CLI), not by GoReleaser's `builds`. Using GoReleaser Pro would enable a single-pipeline approach (prebuilt + split); that is a cost/licensing decision not made here.

## 3. CGO and SQLite

| Fact | Source |
| --- | --- |
| `mattn/go-sqlite3` is a cgo package: "you are required to set the environment variable `CGO_ENABLED=1` and have a `gcc` compiler present within your path." (Zenith currently depends on `v1.14.37`.) | [go-sqlite3 README](https://github.com/mattn/go-sqlite3#readme), repo `go.mod` |
| `modernc.org/sqlite` is described as "Pure-Go SQLite, no cgo". | [cznic/sqlite README](https://gitlab.com/cznic/sqlite) (canonical; GitHub is a mirror) |
| Wails v3 CGO needs: **Windows** none by default (Docker only if you set `CGO_ENABLED=1` on a non-Windows host); **macOS** required (webview integration); **Linux** required (webview integration). | [cross-platform.md](https://github.com/wailsapp/wails/blob/master/docs/mpress/content/guides/build/cross-platform.md) |
| Wails v3 cross-compile matrix: Windows target is "Native Go" from any host (`CGO_ENABLED=0`); macOS target from Linux/Windows and Linux target from macOS/Windows use a Docker image with the Zig cross-compiler and a macOS SDK; the macOS SDK is downloaded from `wailsapp/macosx-sdks` and users must review Apple's SDK license ("Wails does not distribute the macOS SDK"). Cross-compiled macOS binaries "are not code-signed". | [cross-platform.md](https://github.com/wailsapp/wails/blob/master/docs/mpress/content/guides/build/cross-platform.md) |
| Wails v3's own `v3/go.mod` already requires `modernc.org/sqlite` (v1.44.3) as a dependency of the module (purpose not investigated). | [v3/go.mod](https://github.com/wailsapp/wails/blob/master/v3/go.mod) |

**Implications (inference):**
- On macOS and Linux, CGO is unavoidable for the Wails app itself, so keeping mattn there costs nothing extra beyond the toolchain already needed.
- On Windows, v3 can be built CGO-free, so keeping `mattn/go-sqlite3` introduces a requirement for a C compiler (and means Windows can no longer be cross-built "Native Go" from macOS/Linux; per the same page you would use the Docker route with `CGO_ENABLED=1`). A pure-Go driver would preserve the CGO-free Windows path.
- The shared core serves both the CLI (currently built with `CGO_ENABLED=0` in GoReleaser although it imports a cgo driver; see open question 1) and the desktop app. A driver swap touches only `internal/storage` (`Store` interface unchanged), but is a data-layer decision outside this ticket.
- Wails v2 does not use CGO on Windows either; v2 macOS/Linux builds use the platform toolchain per the v2 installation page (Xcode CLI tools, gcc + gtk/webkit).

## 4. Build, sign, notarize, release per OS (Wails v3 unless stated)

### CI runners

Wails' own v3 CI matrix runs `windows-latest`, `ubuntu-latest`, `macos-latest` with Go 1.25; Ubuntu installs `libgtk-4-dev libwebkitgtk-6.0-dev libwayland-dev build-essential pkg-config` (GTK4 default) or `libgtk-3-dev libwebkit2gtk-4.1-dev` (legacy). Source: [build-and-test-v3.yml](https://github.com/wailsapp/wails/blob/master/.github/workflows/build-and-test-v3.yml). Repo's release workflow currently pins Go 1.22 while `go.mod` requires Go 1.25 (already noted in `CLAUDE.md`); Wails v3 requires Go 1.25+ ([status.md](https://github.com/wailsapp/wails/blob/master/docs/mpress/content/status.md)).

### macOS

- Packaging: `wails3 package GOOS=darwin` produces `bin/<AppName>.app` (binary, icon, `Info.plist`). DMG: "copy ... or create a DMG". Source: [macos.md](https://github.com/wailsapp/wails/blob/master/docs/mpress/content/guides/build/macos.md).
- Signing prerequisites: Apple Developer Account ($99/year per Wails docs), Developer ID Application certificate, Xcode CLI tools. Tasks: `wails3 task darwin:sign` and `wails3 task darwin:sign:notarize`; notarization credentials via `xcrun notarytool store-credentials`; verify with `spctl --assess`. Ticket is stapled automatically. Entitlements via `wails3 setup entitlements` (production preset: network only). Source: [signing.md](https://github.com/wailsapp/wails/blob/master/docs/mpress/content/guides/build/signing.md).
- "macOS signing requires a Mac due to Apple's tooling requirements" (signing matrix: `.app` signing and notarization only from macOS). Same source.
- Notifications on macOS need user authorization; autostart via `SMAppService` needs a bundled `.app`; bundle resources inside a signed app are immutable (changing them invalidates the signature), so user data must live in Application Support. Sources: [notifications/overview.md](https://github.com/wailsapp/wails/blob/master/docs/mpress/content/features/notifications/overview.md), [autostart/basics.md](https://github.com/wailsapp/wails/blob/master/docs/mpress/content/features/autostart/basics.md), [macos.md](https://github.com/wailsapp/wails/blob/master/docs/mpress/content/guides/build/macos.md).
- Projects in iCloud Drive/Dropbox-style synced folders can break `codesign` (Finder metadata on `.app`); Wails relocates build output to `~/Library/Application Support/Wails/build-output/...` when it detects this. Relevant because the product's sync design uses such folders for *data*, not for the build tree, but the developer's checkout location matters. Source: [macos.md](https://github.com/wailsapp/wails/blob/master/docs/mpress/content/guides/build/macos.md).
- Apple's own notarization pages (`developer.apple.com/documentation/security/notarizing-macos-software-before-distribution`, `/support/compare-memberships/`) could not be read by the fetch tool (content not returned), so Apple-side requirements above are cited via the Wails docs, not Apple. See open questions.
- GoReleaser: `.app`, DMG and native notarize are Pro (section 2).

### Windows

- Packaging: `wails3 package GOOS=windows` builds an NSIS installer (`build/windows/nsis/<AppName>-installer.exe`) and generates a WebView2 bootstrapper; `FORMAT=msix` produces `bin/<AppName>-<arch>.msix` (needs `makeappx.exe`). Source: [windows.md](https://github.com/wailsapp/wails/blob/master/docs/mpress/content/guides/build/windows.md).
- Signing: `signtool.exe` natively on Windows; built-in signer when cross-signing; `osslsigncode` is listed for macOS/Linux hosts. Formats: `.exe`, `.msi`, `.msix/.appx`. Windows signing can be done from any host. Certificate password via keychain or `WAILS_WINDOWS_CERT_PASSWORD` in CI. Self-signed certificates "trigger SmartScreen warnings" and are for testing only. Source: [signing.md](https://github.com/wailsapp/wails/blob/master/docs/mpress/content/guides/build/signing.md).
- Wails' older v2 signing guide states an EV certificate is not required unless writing kernel-level software; a standard certificate is enough for signing an app. Source: [website/docs/guides/signing.mdx](https://github.com/wailsapp/wails/blob/master/website/docs/guides/signing.mdx) (v2 docs, from Wails; not a Microsoft statement about SmartScreen reputation).
- Microsoft's managed option, Artifact Signing (formerly Trusted Signing): public-trust certificates for **organizations** in the US, Canada, EU, UK, Australia, New Zealand, Japan, South Korea, Singapore, Switzerland, Norway and Israel; **individual developers must be located in the US or Canada**; requires an Azure subscription, a Microsoft Entra tenant, identity validation done in the Azure portal; identity validation takes "1 to 20 business days"; Basic and Premium SKUs exist (prices not retrieved). Source: [Microsoft Learn quickstart](https://learn.microsoft.com/en-us/azure/artifact-signing/quickstart), [overview](https://learn.microsoft.com/en-us/azure/artifact-signing/overview).
- Windows runtime requirement: WebView2 (Wails' NSIS flow bootstraps it).

### Linux

- Packaging: `wails3 package GOOS=linux` creates AppImage, DEB, RPM and Arch packages in `bin/`; DEB/RPM metadata via `build/linux/nfpm/nfpm.yaml`. Source: [linux.md](https://github.com/wailsapp/wails/blob/master/docs/mpress/content/guides/build/linux.md).
- Signing: DEB/RPM signed with a PGP key (`wails3 task linux:sign:deb|rpm|packages`); this proves package origin, not OS-level trust. Source: [signing.md](https://github.com/wailsapp/wails/blob/master/docs/mpress/content/guides/build/signing.md).
- Runtime deps: GTK4 + WebKitGTK 6.0 by default; legacy GTK3/WebKit2GTK 4.1 via `-tags gtk3` until v3.1. ARM64 from x86_64 hosts uses Docker. Source: [linux.md](https://github.com/wailsapp/wails/blob/master/docs/mpress/content/guides/build/linux.md).

### Wails v2 equivalents (for comparison)

v2 uses `wails build`, NSIS for Windows installers (optional dependency), UPX optional, and its signing guide documents manual `signtool` and `gon` (notarization) steps inside GitHub Actions rather than built-in `sign` tasks; the guide notes Apple's `altool` syntax is deprecated in favour of `notarytool`. Sources: [installation.mdx](https://github.com/wailsapp/wails/blob/master/website/docs/gettingstarted/installation.mdx), [signing.mdx](https://github.com/wailsapp/wails/blob/master/website/docs/guides/signing.mdx).

## 5. Constraints on the shared-core design (what the sources imply)

These are inferences tying the sources above to `CLAUDE.md`; none is a decided design.

1. **Process model / DB access.** The desktop app, CLI, TUI and (today) `daemon` all open `~/.zenith/data.db`. v3 offers single-instance locking, autostart and tray, which enable a long-lived GUI process owning timers/notifications, while the CLI remains short-lived. The sources do not describe SQLite multi-process concurrency; that is outside Wails docs and was not researched here.
2. **Notifications.** `checkExpiredTimersPassive()` currently shells out to macOS `osascript`. v3 provides cross-platform native notifications (macOS needs authorization and a bundled app; scheduled delivery is native only on macOS, in-process timer on Windows/Linux, so scheduled notifications on Windows/Linux only fire while the app process is running).
3. **Tray/hotkey features are v3-only** and Linux behaviour varies by desktop environment and by X11 vs Wayland, so any "menu-bar timer" or "quick capture hotkey" extension point must tolerate absence (the docs say to degrade gracefully).
4. **Frontend toolchain.** Both versions need Node/npm on the build machine (v2 installation page; v3 Docker image installs nodejs/npm). React + TS + Tailwind + shadcn is not constrained by either source.
5. **Go version.** Wails v3 needs Go 1.25+, matching `go.mod`; the release workflow's Go 1.22 must change for any pipeline that compiles v3 code.
6. **Module layout.** Wails' project template expects its own Taskfile-based build (`build/<os>/Taskfile.yml`) and `wails3` CLI. Whether it can live in a sub-directory of the existing module (for example `desktop/`) alongside the root `main.go` was not verified.

## 6. Open questions (flagged, not decided)

1. **CGO inconsistency in the current CLI release.** `.goreleaser.yaml` sets `CGO_ENABLED=0` yet the CLI imports `mattn/go-sqlite3`, which is a cgo package. The go-sqlite3 README says CGO is required; what a `CGO_ENABLED=0` build does at runtime was not verified or reproduced here. Worth verifying that published CLI binaries actually work; this directly affects whether to switch drivers. Not investigated further because it is outside the question.
2. **Version choice depends on product tolerance for a beta.** Sources establish v3 is beta with an "API is stable" statement and no GA date; they do not say when 3.0.0 final ships. v2 cannot provide tray/global hotkeys/autostart per its docs. Decision belongs to the owner.
3. **Apple primary sources not retrievable.** Apple's notarization and membership/fee-waiver pages returned no readable content to the fetch tool. Still to confirm from Apple directly: the annual fee, whether any fee waiver applies to an open-source, donation-funded project, and Developer ID/notarization terms.
4. **Windows signing cost/eligibility.** Artifact Signing pricing (Basic/Premium) was not retrieved. Eligibility is by geography of the signer; the maintainer's location and legal entity status decide which path applies. No Microsoft statement about SmartScreen reputation building was read.
5. **GoReleaser Pro vs OSS-only pipeline.** Not priced here. If OSS-only, the desktop build/sign/notarize/upload steps live in separate workflow jobs outside GoReleaser (inference).
6. **Asset-name collisions with `zenith update`.** Confirm the exact asset names Wails v3 packaging emits, then either keep them off the `<os>_<arch>[.ext]` suffix, split the release stream, or add `Config.Filters` in `cmd/update.go`. Also unread: Wails v3 `app.Updater` provider's asset naming rules.
7. **Linux target matrix.** Ubuntu 22.04 / Debian 12 need the GTK3 legacy path, which Wails removes in v3.1; GTK4/WebKitGTK 6.0 availability on the distributions Zenith wants to support was not checked.
8. **Wayland global shortcuts** depend on a DE-provided portal; which environments Zenith must support is a product decision.
9. **Not covered:** sync-folder interaction with SQLite (belongs to the sync tickets), Wails v3 bindings/IPC design, code-signing for the AppImage, Flatpak/Snap/winget/Homebrew distribution.

## Source list

Wails: [repo](https://github.com/wailsapp/wails) (docs under `docs/mpress/content/` for v3, `website/docs/` for v2), [releases](https://github.com/wailsapp/wails/releases), [status](https://github.com/wailsapp/wails/blob/master/docs/mpress/content/status.md), [cross-platform build](https://github.com/wailsapp/wails/blob/master/docs/mpress/content/guides/build/cross-platform.md), [signing](https://github.com/wailsapp/wails/blob/master/docs/mpress/content/guides/build/signing.md), [macOS](https://github.com/wailsapp/wails/blob/master/docs/mpress/content/guides/build/macos.md), [Windows](https://github.com/wailsapp/wails/blob/master/docs/mpress/content/guides/build/windows.md), [Linux](https://github.com/wailsapp/wails/blob/master/docs/mpress/content/guides/build/linux.md), [systray](https://github.com/wailsapp/wails/blob/master/docs/mpress/content/features/menus/systray.md), [global shortcuts](https://github.com/wailsapp/wails/blob/master/docs/mpress/content/features/keyboard/global-shortcuts.md), [v2 installation](https://github.com/wailsapp/wails/blob/master/website/docs/gettingstarted/installation.mdx), [v2 signing](https://github.com/wailsapp/wails/blob/master/website/docs/guides/signing.mdx).
GoReleaser: [builds](https://goreleaser.com/customization/builds/go/), [notarize](https://goreleaser.com/customization/notarize/), [repo docs](https://github.com/goreleaser/goreleaser/tree/main/www/content/customization).
Others: [go-github-selfupdate](https://github.com/rhysd/go-github-selfupdate), [mattn/go-sqlite3](https://github.com/mattn/go-sqlite3), [modernc.org/sqlite](https://gitlab.com/cznic/sqlite), [Microsoft Artifact Signing](https://learn.microsoft.com/en-us/azure/artifact-signing/quickstart).
