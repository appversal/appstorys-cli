package com.example.app

class HomeActivity {
    fun onResume() {
        AppStorys.getInstance().getScreenCampaigns("Home Screen", listOf("widget_one"))
    }
}

@Composable
fun HomeScreen() {
    overlayElements()
    Widget(position = "widget_one")
    Modifier.appstorys("tooltip_one")
}
