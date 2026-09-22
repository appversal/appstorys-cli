package com.example.broken

class SettingsActivity {
    fun onResume() {
        // No getScreenCampaigns call at all — exercises the "not
        // tracked" failure path.
    }
}
