## Where things go

- **Init:** one `AppStorys.initialize(...)` call in `onCreate` of the `Application` subclass named by `android:name` on `<application>` in `AndroidManifest.xml`. Not in an Activity, Fragment or composable.
- **Token:** the generated call passes `BuildConfig.APPSTORYS_API_TOKEN`, fed by a `buildConfigField` that reads the `APPSTORYS_API_TOKEN` environment variable at build time. `init.token_expr` overrides the expression.
- **Overlay host:** `overlayElements()` once. With Compose Navigation that is the composable enclosing the `NavHost`; with Activities it is inside `setContent { ... }`. XML layouts use `OverlayLayoutView` as the layout root instead.
- **Screens:** `AppStorys.getScreenCampaigns("Name")`. For a Compose `NavHost` destination it goes in `LaunchedEffect(Unit) { ... }`; for an Activity, in `onResume`.
- **Events:** `AppStorys.trackEvents(event = "Name", metadata = mapOf("key" to value))`. When called positionally the event name is the second argument.
- **Names:** screen names and widget positions are exact-match targeting keys (Android compares case-insensitively). Never rename existing ones.

## What the CLI does

- `init`: creates an `Application` subclass if none exists, registers it in the manifest, inserts the init call into `onCreate` (creating `onCreate` if missing), and edits Gradle Kotlin DSL: the JitPack repository in `settings.gradle.kts`, and the SDK dependency, `buildFeatures { buildConfig = true }` and the `buildConfigField` in the app module's `build.gradle.kts`. When the manifest has no `package` attribute it reads `namespace` from the app module's `android { }` block.
- `doctor`: checks the init call sits in the manifest's `Application` class `onCreate`, and for every Activity declared in the manifest whether its own source file tracks a screen and has an overlay host (matched per file, so the result is a heuristic).
- `integrate`: for a Compose `NavHost`, adds the overlay host to the enclosing composable and a tracked `LaunchedEffect` to each untracked destination whose route is a string literal. For manifest Activities it adds `getScreenCampaigns` to `onResume` (creating it if missing) and, when a `setContent { }` block exists anywhere in the Activity's class body — directly in `onCreate` or in a helper method it calls — the overlay host inside it. An Activity that hosts a `NavHost` is skipped, because its destinations are the screens.

## What stays manual

- Groovy `build.gradle` / `settings.gradle` files: the CLI only edits the `.kts` forms and leaves Groovy alone.
- A `navigateToScreen` lambda: it is required by the SDK and an unset one crashes on the first deep-link click. The CLI does not add it; tell the developer.
- An init call in the wrong place: `init` will not move it.
- `doctor` does not list Compose `NavHost` destinations (only manifest Activities); use `integrate` and `scan` to see those.
- An Activity with no `setContent { }` anywhere in its own class body — a base class's `onCreate`, or XML layouts instead — gets no overlay-host suggestion.
- Missing SDK version pinning, `minSdk` and push (Firebase) setup are not checked.
- The SDK repository's own version strings disagree (tag 5.0.2, POM 3.7.2, runtime 5.0.1). The CLI treats 5.0.2 as the supported version, and that is unconfirmed: if the app pins a nearby version, mention this to the developer rather than calling it a mismatch outright.
