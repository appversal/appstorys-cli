package project

// All returns one adapter instance per supported platform.
func All() []Adapter {
	return []Adapter{
		NewAndroidAdapter(),
		NewFlutterAdapter(),
		NewReactNativeAdapter(),
		NewIOSAdapter(),
	}
}

// ByPlatform returns the adapter for p, if p is supported.
func ByPlatform(p Platform) (Adapter, bool) {
	for _, a := range All() {
		if a.Platform() == p {
			return a, true
		}
	}
	return nil, false
}
