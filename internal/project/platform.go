// Package project detects which AppStorys SDK platform (and, later,
// which navigation library) a customer app uses, so the CLI can pick the
// right language adapter for everything downstream.
package project

// Platform identifies one of the four supported SDK platforms.
type Platform string

const (
	Android     Platform = "android"
	Flutter     Platform = "flutter"
	ReactNative Platform = "react-native"
	IOS         Platform = "ios"
)

// All lists every supported platform, in the priority order the spec
// gives them.
func AllPlatforms() []Platform {
	return []Platform{Android, Flutter, ReactNative, IOS}
}
