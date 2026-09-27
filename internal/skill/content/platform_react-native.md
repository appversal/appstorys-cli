## Where things go

- **Init:** one `AppStorys.initialize(token)` call in a mount-only `useEffect(() => { ... }, [])` of the component registered with `AppRegistry.registerComponent`. Not in a screen component or the render body.
- **Token:** the generated call passes an identifier `APPSTORYS_API_TOKEN` that the developer must define (for example from `react-native-config` or their own env setup); the CLI never opens `.env` files. `init.token_expr` overrides the expression.
- **Screens and overlay host:** wrap the screen's returned JSX in `<AppStorys.Screen name="...">`. That one wrapper tracks the screen and is its overlay host. The alternative is `<AppStorys.Container>` plus an `AppStorys.trackScreen(name)` call. `doctor`/`integrate` discover screens from React Navigation's JSX route form (`<Stack.Screen name="X" component={Y} />`) and its static-API form (`createNativeStackNavigator({ screens: { X: Y } })` and the `createXNavigator` family generally).
- **Inline placements:** `<AppStorys.Stories />` and `<AppStorys.Widgets position="..." />` only work inside a `Screen` or `Container`; outside one they throw at runtime.
- **Tags:** an `appstorys="id"` prop on any JSX element (needs the SDK's Babel plugin).
- **Events:** `AppStorys.trackEvent('Name', undefined, { key: value })`. The arguments are event, campaign ID, metadata; note it is `trackEvent`, singular.
- **Names:** screen names and widget positions are exact-match targeting keys. Never rename existing ones.

## What the CLI does

- `init`: adds the init call to the registered root component's existing mount-only `useEffect`, or adds a new `useEffect` if it has none (import `useEffect` from `react` if missing).
- `doctor`: checks the single init call's enclosing component is the one registered with `AppRegistry` (component-level, not specifically inside the `useEffect`), and, for every `<X.Screen name="..." component={Y} />` route, whether `Y`'s file has a screen and an overlay host.
- `integrate`: wraps the returned JSX of an untracked route component in `<AppStorys.Screen name="...">`, as two edits under one ID. It only does this when the component has a single plain JSX `return` or expression body; conditional returns, fragments and multiple returns are left alone.

## What stays manual

- The npm dependency and its 14 peer dependencies, using the project's package manager.
- The SDK Babel plugin in `babel.config.js`, `GestureHandlerRootView` and `SafeAreaProvider` above the screens, and `appstorys.d.ts` in the `tsconfig.json` include. React Native 0.70 or newer and React 18 or newer are required.
- Routes registered with a render-prop child (`<Stack.Screen name="X">{() => <Y/>}</Stack.Screen>`), or a static-API screen given as an object instead of a plain component (`Home: { screen: HomeScreen, options: {...} }`), are not seen by `doctor` or `integrate`.
- A screen `integrate` skipped for its return shape: wrap it by hand, with the developer's agreement.
- The generated wrapper is not reformatted; suggest running the project's formatter.
