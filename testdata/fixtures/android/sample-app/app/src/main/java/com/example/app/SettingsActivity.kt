package com.example.app

class SettingsActivity {
    fun onResume() {
        AppStorys.getInstance().getScreenCampaigns("Settings Screen", listOf("widget_settings"))
    }
}

@Composable
fun SettingsScreen() {
    overlayElements()
}
