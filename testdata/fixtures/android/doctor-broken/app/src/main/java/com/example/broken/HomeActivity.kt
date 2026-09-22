package com.example.broken

class HomeActivity {
    fun onCreate() {
        // Wrong location: init belongs in BrokenApp.onCreate.
        AppStorys.initialize(token = "hardcoded-token-literal")
    }

    fun onResume() {
        // Tracked, but no overlayElements() anywhere in this file —
        // exercises the "no overlay host" failure path.
        AppStorys.getInstance().getScreenCampaigns("Home", listOf())
    }
}
