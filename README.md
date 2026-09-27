# appstorys-cli

A command-line tool that finds, checks and fixes AppStorys SDK instrumentation in a mobile app. It reads your source, tells you what's missing or wrong, and can write the fix as a reviewable diff. It also ships an agent skill so an AI coding agent can do the same work by driving the CLI.

It runs entirely on your machine. This version makes no network calls.

## Why

The AppStorys SDK never tells you when it's misplaced. A missing overlay host, a screen that's never tracked, or a reserved event name doesn't crash anything: campaigns just silently stop rendering. Each platform also places init, overlay hosts and screen tracking differently. `appstorys-cli` looks for those mistakes directly and proposes the edits.

## Supported platforms

| Platform | `scan` `lint` `validate` `events list` | `doctor` | `init` | `integrate` |
| --- | --- | --- | --- | --- |
| Android (Kotlin, Compose and XML) | yes | manifest Activities | yes, including Gradle | Compose `NavHost`, Activities |
| Flutter (Dart) | yes | `MaterialApp(routes:)`, `go_router` | `main()` | `initState`, existing `Stack` |
| React Native (TS/JS) | yes | React Navigation, JSX and static API | root `useEffect` | wraps JSX in `<AppStorys.Screen>` |
| iOS (Swift) | not yet | not yet | not yet | not yet |

Only the latest SDK version of each platform is supported. `appstorys-cli version` prints them. On iOS the CLI detects the project and stops with exit code 3.

A project is detected from files at `--root`: `settings.gradle(.kts)` plus `build.gradle(.kts)` (Android), a `pubspec.yaml` mentioning Flutter, a `package.json` depending on `react-native`, or `Package.swift` / a `*.xcodeproj` (iOS). Point `--root` at the app itself, not a monorepo parent.

## Install

**macOS:**

```
brew tap appversal/tap
brew install --cask appstorys-cli
```

**Windows:**

```
scoop bucket add appversal https://github.com/appversal/scoop-bucket.git
scoop install appversal/appstorys-cli
```

**Linux, or building from source:** download a prebuilt archive from the [releases page](https://github.com/appversal/appstorys-cli/releases), or build with Go 1.25 or newer and a C compiler (tree-sitter needs cgo):

```
CGO_ENABLED=1 go build -o appstorys-cli ./cmd/appstorys-cli
./appstorys-cli version
```

Because of cgo, build once per OS and architecture.

## Quick start

```
appstorys-cli doctor --root ./my-app
```

```
CHECK   SCREEN    FILE:LINE                         TRACKED AS   OVERLAY  STATUS
init    -         src/App.tsx:12                    -            -        pass
screen  Home      src/screens/HomeScreen.tsx:5      Home Screen  root     pass
screen  Settings  src/screens/SettingsScreen.tsx:4  -            missing  fail: not tracked
```

`doctor` exits 1 when a check fails. To fix what it found:

```
appstorys-cli init --root ./my-app                     # diff of the init call; add --apply to write it
appstorys-cli integrate --root ./my-app                # list suggestions
appstorys-cli integrate --root ./my-app --diff         # show every change
appstorys-cli integrate --root ./my-app --suggestion c0ce8dbed5d7 --apply
```

```
ID            KIND    FILE                            CONFIDENCE  REASON
c0ce8dbed5d7  screen  src/screens/SettingsScreen.tsx  0.90        Wrap SettingsScreen's returned JSX in <AppStorys.Screen name="Settings">, which also tracks the screen
```

Then check the result:

```
appstorys-cli lint --root ./my-app
appstorys-cli doctor --root ./my-app
```

## Commands

| Command | What it does |
| --- | --- |
| `scan` | Lists every init, screen, event, overlay-host, placement and tag call site. `--kind`, `--name`, `--platform`, `--group-by name\|file`, `--format table\|json\|sarif\|md` |
| `lint` | Checks tracking mistakes: reserved event names, near-duplicate names, property keys that look like personal data, missing or duplicate init, and more. `--format table\|json\|sarif\|md` |
| `validate` | The CI gate: `lint`, but exits 1 on findings at or above `--fail-on error\|warning` |
| `events list` | The deduplicated events, screens and widget positions the app uses. `--only events\|screens\|positions` |
| `doctor` | Checks init placement, and screen tracking and overlay hosts on every screen it can discover |
| `init` | Adds the init call for the detected platform. A diff by default; `--apply` writes it |
| `integrate` | Suggests overlay hosts and screen tracking. `--only overlay\|screen`, `--min-confidence`, `--suggestion <id>`, `--diff`, `--apply` |
| `integrate placement` | Inserts one stories or widget placement exactly where you say: `--kind`, `--position`, `--at file:line` |
| `skill export` | Writes the agent skill (see below) |
| `config init` | Writes a starter `.appstorys-cli.yaml` |
| `version` | The CLI version and the SDK version supported for each platform |

Every command takes `--root <dir>` (default: the current directory) and `--config <file>`. `--format json` is available on `scan`, `lint`, `validate`, `doctor`, `integrate` and `events list`.

## How it edits your code

- **Diff first.** `init` and `integrate` print what they would change and write nothing until you pass `--apply`.
- **Clean git tree required.** `--apply` refuses (exit 5) if the tree has uncommitted changes, so any edit is one `git checkout` from undone.
- **Insert only, and idempotent.** It adds lines; it never deletes, renames or moves existing code. Running it again after applying proposes nothing.
- **Screen names and widget positions are never renamed.** They are campaign-targeting keys that must match the dashboard exactly. New names are derived from route or class names and shown for confirmation.
- **When unsure, it skips.** A screen whose structure it can't safely edit is left to `doctor` to report, not guessed at.
- **What it reads.** Source files and build files (`AndroidManifest.xml`, `build.gradle(.kts)`, `settings.gradle(.kts)`, `pubspec.yaml`, `package.json`). Never `.env*`, keystores, `local.properties` or `Info.plist` values. Event property values and init arguments are never extracted, only property names and inferred types.
- **Tokens.** The generated init call references a build-time value (`BuildConfig.APPSTORYS_API_TOKEN` on Android, `String.fromEnvironment('APPSTORYS_API_TOKEN')` on Flutter, an `APPSTORYS_API_TOKEN` identifier you define on React Native). The CLI never reads or writes a token.

## Agent skill

```
appstorys-cli skill export --root ./my-app
```

writes `.claude/skills/appstorys-integration/` (or use `--dir`): a `SKILL.md` workflow for a coding agent, a reference file per platform, and `reference/output.md` with the JSON schemas and exit codes. It's generated by the binary, so it always matches the CLI and SDK versions that exported it; after upgrading the CLI, re-run with `--force` to refresh it. `SKILL.md` is plain Markdown, so it also works as instructions for agents without skill support.

The skill has the agent run `doctor`, apply `init` and `integrate` suggestions one at a time behind your confirmation, and re-check with `lint` and `doctor`. It forbids reading secrets, editing `project.pbxproj`, renaming screens, and committing or stashing your changes.

## Configuration

`.appstorys-cli.yaml` is optional (`appstorys-cli config init` writes a template). These keys are used:

```yaml
receivers:            # extra names for the SDK singleton, e.g. an app-level alias
  - App.appStorys
rules:                # per-rule severity: error | warning | info | off
  pii-property: error
  dynamic-name: off
init:
  token_expr: "Secrets.appstorysToken"   # the expression the generated init call uses
```

Other keys in the template (`platforms`, `include`, `exclude`, `wrappers`, `naming`, `screens`, `integrate`) are parsed but not applied yet.

## JSON output and exit codes

Every `--format json` command writes `{ "schemaVersion": 1, "data": ... }`; an empty result is `[]`. Field-by-field schemas are in the exported skill's `reference/output.md`.

| Code | Meaning |
| --- | --- |
| 0 | Success |
| 1 | `lint`, `validate` or `doctor` found errors or failing checks |
| 2 | Usage or config error |
| 3 | No project detected, or the platform isn't supported by that command (iOS) |
| 4 | Reserved for event registration (not in this version) |
| 5 | Refused to write: dirty git tree, or `skill export` would overwrite a differing file without `--force` |

Failures other than exit 1 print `error: ...` on standard error.

## In CI

```
appstorys-cli validate --root ./my-app --fail-on warning --format sarif > appstorys.sarif
```

`validate` never edits files. SARIF works with code-scanning annotations.

## Limitations

- **iOS isn't supported.** The SDK is binary-only, so there's no source to derive symbols from.
- **`doctor` and `integrate` are heuristics.** They match a screen to its source file, and only see routes registered the common way: manifest Activities and Compose `NavHost` (Android), `MaterialApp(routes:)` and `go_router`'s `GoRoute` including nested routes (Flutter), `<X.Screen component={Y} />` and the static `createXNavigator({ screens: { X: Y } })` API (React Native). Routes registered through `onGenerateRoute`, `Navigator.push`, a render-prop `Screen` child, or a static screen given as an object instead of a plain component, aren't seen.
- **Dependency setup is only automated on Android.** Flutter (`pubspec.yaml`, desugaring) and React Native (npm packages and peer dependencies, Babel plugin, root wrappers, `appstorys.d.ts`) are printed as manual steps after `init --apply`.
- **Flutter `init` puts the call first in `main()`**, which can land above `WidgetsFlutterBinding.ensureInitialized()`; check the order after applying.
- **No event suggestions.** The CLI finds and lints event calls but doesn't propose where to add them, and there's no `events register` yet, so nothing is sent to AppStorys.
- Names are read only from string literals; constants and computed names show up as `<dynamic>`.

## Development

```
CGO_ENABLED=1 go build ./...
CGO_ENABLED=1 go test ./...
```

Small hand-written apps for every platform and scenario live in `testdata/fixtures/`; the tests run the scan, lint, doctor, init, integrate and skill commands against them, including that applied edits still parse and that a second run changes nothing. SDK facts (symbols, reserved event names) are data in `internal/symbols/*.yaml`, not Go code.

The original feature plan, roadmap and open questions are in [`docs/task-context.md`](docs/task-context.md).

## License

[MIT](LICENSE)
