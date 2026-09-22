package com.example.broken

class BrokenApp : android.app.Application() {
    override fun onCreate() {
        super.onCreate()
        // No AppStorys.initialize() call here — it's (wrongly) in
        // HomeActivity.onCreate instead. Exercises the init-location
        // failure path.
    }
}
