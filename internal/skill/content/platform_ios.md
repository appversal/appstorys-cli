## The CLI does not support iOS yet

`appstorys-cli` detects an iOS project but has no scanner, doctor, init or integrate for it (the SDK is binary-only, so there is no source to derive symbols from). Every command that needs one exits with code 3 and an "aren't supported" message. Do not try to work around that. Instead, integrate by hand from the facts below, which come from the project's SDK notes and are **not verified by the CLI**, and tell the developer that nothing here was checked automatically.

## SDK facts (SwiftUI and UIKit)

- **Install:** Swift Package Manager product `AppStorys_iOS`. Deployment target iOS 15 or newer. **Never edit `project.pbxproj`**: print the manual Xcode steps (add the package, link the product) and let the developer do them.
- **Init:** `AppStorys.shared.appstorys(...)` (async) or `AppStorys.initialize(...)`, once, from the `@main` App's `init` (`Task { await AppStorys.shared.appstorys(...) }`) or from `application(_:didFinishLaunchingWithOptions:)`. Not from a destination view or view controller.
- **Token:** read a build-time value, for example `Bundle.main.object(forInfoDictionaryKey: "AppStorysAPIToken") as? String`, mapped from an `.xcconfig` build setting to that Info.plist key. Never put a token in source.
- **Overlay host:** `.withAppStorysOverlays()` once on the root view (SwiftUI); `attachOverlaysToKeyWindow()` or the same modifier on the root view (UIKit).
- **Screens:** `.trackAppStorysScreen("Name")` on each destination view (SwiftUI). In UIKit, `trackAppStorysScreen("Name")` in `viewDidAppear` and `untrackAppStorysScreen("Name")` (or `hideAllCampaignsForDisappearingScreen`) in `viewDidDisappear`.
- **Events:** `triggerEvent("Name", metadata: [...])`, with campaign-ID, async and instance variants.
- **Placements and tags:** `AppStorys.Stories`, `Widget(position:)`, `Widgets`, `Milestone(position:)`; tag an element with `.captureAppStorysTag("id")` or `UIView.appStorysTag`.
- **Names:** screen names and widget positions are exact-match targeting keys. Never rename existing ones.

Ask the developer to confirm every screen name before you add it, and hand over the list of edits you made.
