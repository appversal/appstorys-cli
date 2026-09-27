package com.example.viewsoverlayhelper

class HomeActivity {
    fun onCreate() {
        setupContent()
    }

    private fun setupContent() {
        setContent {
            HomeScreen()
        }
    }

    fun onResume() {
        AppStorys.getScreenCampaigns("Home")
    }
}
