package com.example.viewsoverlay

class HomeActivity {
    fun onCreate() {
        setContent {
            HomeScreen()
        }
    }

    fun onResume() {
        AppStorys.getScreenCampaigns("Home")
    }
}
