package project

import "testing"

func TestDetectFixtures(t *testing.T) {
	tests := []struct {
		name string
		root string
		want []Platform
	}{
		{"android minimal", "../../testdata/fixtures/android/minimal", []Platform{Android}},
		{"flutter minimal", "../../testdata/fixtures/flutter/minimal", []Platform{Flutter}},
		{"react native minimal", "../../testdata/fixtures/reactnative/minimal", []Platform{ReactNative}},
		{"ios minimal", "../../testdata/fixtures/ios/minimal", []Platform{IOS}},
		{"empty", "../../testdata/fixtures/none/empty", nil},
		{"react native with native subdirs", "../../testdata/fixtures/reactnative/with-native-subdirs", []Platform{ReactNative}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			infos, err := Detect(tt.root, nil)
			if err != nil {
				t.Fatalf("Detect(%q) error = %v", tt.root, err)
			}
			var got []Platform
			for _, info := range infos {
				got = append(got, info.Platform)
			}
			if !samePlatforms(got, tt.want) {
				t.Errorf("Detect(%q) = %v, want %v", tt.root, got, tt.want)
			}
		})
	}
}

func samePlatforms(a, b []Platform) bool {
	if len(a) != len(b) {
		return false
	}
	seen := make(map[Platform]int)
	for _, p := range a {
		seen[p]++
	}
	for _, p := range b {
		seen[p]--
	}
	for _, count := range seen {
		if count != 0 {
			return false
		}
	}
	return true
}
