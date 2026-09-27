plugins {
    id("com.android.application")
}

android {
    namespace = "com.example.agpns"
    compileSdk = 34

    defaultConfig {
        applicationId = "com.example.agpns"
        minSdk = 24
        targetSdk = 34
    }
}

dependencies {
    implementation("androidx.core:core-ktx:1.13.1")
}
