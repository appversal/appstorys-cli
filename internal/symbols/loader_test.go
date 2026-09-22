package symbols

import "testing"

func TestLoadAllPlatforms(t *testing.T) {
	want := map[string]string{
		"android":      "5.0.2",
		"flutter":      "5.0.2",
		"react-native": "5.0.0",
		"ios":          "5.0.0",
	}

	maps, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(maps) != len(want) {
		t.Fatalf("Load() returned %d platforms, want %d", len(maps), len(want))
	}
	for platform, wantVersion := range want {
		m, ok := maps[platform]
		if !ok {
			t.Errorf("missing platform %q", platform)
			continue
		}
		if m.SDKVersion != wantVersion {
			t.Errorf("%s: sdk_version = %q, want %q", platform, m.SDKVersion, wantVersion)
		}
		if m.Platform != platform {
			t.Errorf("%s: platform field = %q, want %q", platform, m.Platform, platform)
		}
	}
}

func TestLoadPlatformUnknown(t *testing.T) {
	if _, err := LoadPlatform("windows"); err == nil {
		t.Fatal("LoadPlatform(\"windows\") expected error, got nil")
	}
}

func TestLoadPlatformSingle(t *testing.T) {
	m, err := LoadPlatform("ios")
	if err != nil {
		t.Fatalf("LoadPlatform(\"ios\") error = %v", err)
	}
	if m.SDKVersion != "5.0.0" {
		t.Errorf("ios sdk_version = %q, want %q", m.SDKVersion, "5.0.0")
	}
}
