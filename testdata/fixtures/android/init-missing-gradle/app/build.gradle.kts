plugins {
    id("com.android.application")
}

android {
    namespace = "com.example.gradle"
    compileSdk = 34

    defaultConfig {
        applicationId = "com.example.gradle"
        minSdk = 24
        targetSdk = 34
    }
}

dependencies {
    implementation("androidx.core:core-ktx:1.13.1")
}
