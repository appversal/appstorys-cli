## Where things go

- **Init:** one `AppStorys.initialize(...)` call in `main()`, after `WidgetsFlutterBinding.ensureInitialized()` and before `runApp`, not awaited (or in the root widget's `initState`).
- **Token:** the generated call passes `const String.fromEnvironment('APPSTORYS_API_TOKEN')`. The developer supplies the value with `--dart-define=APPSTORYS_API_TOKEN=...` or `--dart-define-from-file` in their build scripts. `init.token_expr` overrides the expression.
- **Screens:** `AppStorys.trackScreen('Name', context)` in the screen's `State.initState`. `doctor`/`integrate` discover screens from `MaterialApp(routes: {...})` and `go_router`'s `GoRoute(path:, builder:)`.
- **Overlay host:** `...AppStorys.overlayElements()` spread into the `children` of a `Stack` in that screen's `build`. It must be inside a `Stack`: outside one, its `Positioned` children fail.
- **Names:** screen names and widget positions are exact-match targeting keys. Never rename existing ones.

## What the CLI does

- `init`: inserts the init call as the **first statement** of `main()`. **After applying, check that `WidgetsFlutterBinding.ensureInitialized()` is not below it; if it is, move the init call under it** (a manual edit, with the developer's agreement).
- `doctor`: checks the init call is in `main()` or in the `initState` of the `State` for the widget passed to `runApp`, and, for every route in a `MaterialApp(routes: { ... })` map, whether the route widget's file (its `State` class for a `StatefulWidget`) tracks a screen and has an overlay host.
- `integrate`: adds `trackScreen(name, context)` to the route widget's `State.initState` (creating `initState` if missing), and adds the overlay spread to a `Stack` the screen already has.

## What stays manual

- `pubspec.yaml`: adding `appstorys_sdk_3_0`, and Dart SDK 3.0 or newer.
- Android core library desugaring in `android/app/build.gradle(.kts)`, which the SDK's `flutter_local_notifications` dependency requires.
- Wrapping a screen's body in a `Stack` when it has none: the CLI never does this, so a screen without a `Stack` gets no overlay-host suggestion and `doctor` keeps reporting "no overlay host". Show the developer a proposed edit and get approval first.
- Routes registered through `onGenerateRoute`, `MaterialPageRoute(builder:)` or `Navigator.push` are not seen by `doctor` or `integrate`. `go_router`'s `GoRoute` is, including nested routes, as long as its `builder` returns a plain `Widget()` call.
- Stateless route widgets cannot be tracked from `initState`; tracking them needs a `NavigatorObserver`, which the CLI does not generate.
- Inline placements: `integrate placement` inserts a list element with a trailing comma (for example `AppStorys.widgets(position: "x"),`), so use it only inside a `children: [ ... ]` list.
- The SDK's own version strings disagree (pubspec 5.0.2, runtime 5.0.1). The CLI treats 5.0.2 as the supported version, and that is unconfirmed, so treat a nearby pinned version as worth mentioning, not as an error.
