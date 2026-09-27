package com.example.viewsoverlay

class App : android.app.Application() {
    override fun onCreate() {
        super.onCreate()
        AppStorys.initialize(token = BuildConfig.APPSTORYS_API_TOKEN)
    }
}
