package com.example.app

class LoginViewModel {
    fun submit() {
        AppStorys.trackEvents(event = "Login", metadata = mapOf("method" to "email"))
    }

    fun clicked() {
        // Reserved event name — should be flagged by lint.
        AppStorys.trackEvents(event = "clicked")
    }
}
