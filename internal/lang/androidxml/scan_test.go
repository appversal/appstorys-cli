package androidxml

import "testing"

func TestScan(t *testing.T) {
	src := []byte(`<?xml version="1.0" encoding="utf-8"?>
<com.appversal.appstorys.OverlayLayoutView
    xmlns:android="http://schemas.android.com/apk/res/android"
    xmlns:app="http://schemas.android.com/apk/res-auto"
    android:layout_width="match_parent"
    android:layout_height="match_parent">

    <com.appversal.appstorys.WidgetView
        android:id="@+id/promo_widget"
        app:position="widget_one" />

</com.appversal.appstorys.OverlayLayoutView>
`)

	sites, err := Scan("activity_home.xml", src)
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	if len(sites) != 2 {
		t.Fatalf("Scan() returned %d sites, want 2: %+v", len(sites), sites)
	}

	overlay := sites[0]
	if overlay.Kind != "overlay-host" || overlay.Via != "OverlayLayoutView" {
		t.Errorf("sites[0] = %+v, want overlay-host/OverlayLayoutView", overlay)
	}

	widget := sites[1]
	if widget.Kind != "placement" || widget.Position != "widget_one" {
		t.Errorf("sites[1] = %+v, want placement at widget_one", widget)
	}
	if widget.Line <= overlay.Line {
		t.Errorf("widget Line = %d, want > overlay Line %d", widget.Line, overlay.Line)
	}
}

func TestScanNoMatches(t *testing.T) {
	src := []byte(`<?xml version="1.0" encoding="utf-8"?>
<LinearLayout xmlns:android="http://schemas.android.com/apk/res/android" />
`)
	sites, err := Scan("plain.xml", src)
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	if len(sites) != 0 {
		t.Errorf("Scan() = %+v, want no sites", sites)
	}
}
