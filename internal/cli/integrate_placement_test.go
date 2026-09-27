package cli

import (
	"testing"

	"github.com/appversal/appstorys-cli/internal/project"
)

func TestPlacementSnippetPerPlatform(t *testing.T) {
	tests := []struct {
		platform       project.Platform
		kind, position string
		want           string
		wantErr        bool
	}{
		{project.Android, "widget", "w1", `Widget(position = "w1")`, false},
		{project.Android, "milestone", "m1", `Milestone(position = "m1")`, false},
		{project.Android, "stories", "", `Stories()`, false},
		{project.Android, "reels", "", `Reels()`, false},
		{project.ReactNative, "widget", "w1", `<AppStorys.Widgets position="w1" />`, false},
		{project.ReactNative, "stories", "", `<AppStorys.Stories />`, false},
		{project.Flutter, "widget", "w1", `AppStorys.widgets(position: "w1"),`, false},
		{project.Flutter, "stories", "", `AppStorys.stories(),`, false},
		{project.Android, "widget", "", "", true},      // position required
		{project.ReactNative, "reels", "", "", true},   // android-only kind
		{project.Flutter, "milestone", "m1", "", true}, // android-only kind
		{project.IOS, "widget", "w1", "", true},        // unsupported platform
		{project.Android, "banner", "", "", true},      // unknown kind
	}
	for _, tt := range tests {
		got, err := placementSnippet(tt.platform, tt.kind, tt.position)
		if (err != nil) != tt.wantErr {
			t.Errorf("placementSnippet(%s, %s, %q) error = %v, wantErr %v", tt.platform, tt.kind, tt.position, err, tt.wantErr)
			continue
		}
		if got != tt.want {
			t.Errorf("placementSnippet(%s, %s, %q) = %q, want %q", tt.platform, tt.kind, tt.position, got, tt.want)
		}
	}
}
