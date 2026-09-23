plugins {
    id("com.android.application")
    kotlin("android")
    id("org.jetbrains.kotlin.plugin.compose")
}

kotlin {
    compilerOptions {
        jvmTarget.set(org.jetbrains.kotlin.gradle.dsl.JvmTarget.JVM_17)
    }
}

android {
    namespace = "io.gosicp.tableauxeink"
    compileSdk = 34

    defaultConfig {
        applicationId = "io.gosicp.tableauxeink"
        minSdk = 26
        targetSdk = 33  // Android 13, per project requirement
        versionCode = 1
        versionName = "0.1.0"

        ndk {
            abiFilters += listOf("arm64-v8a")
        }
    }

    buildFeatures {
        compose = true
    }
    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }
    sourceSets {
        getByName("main") {
            // androidsvc Java glue lives outside this app module — pull
            // from the sibling project so we don't have to duplicate it.
            java.srcDirs("src/main/kotlin", "../../../androidsvc/java")
        }
    }
}

dependencies {
    // einkapp.aar is the output of `gomobile bind` (see Taskfile.yaml).
    // Place the file in android-host/app/libs/ before building.
    implementation(fileTree(mapOf("dir" to "libs", "include" to listOf("*.aar"))))

    val composeBom = platform("androidx.compose:compose-bom:2024.10.01")
    implementation(composeBom)
    implementation("androidx.compose.material3:material3")
    implementation("androidx.compose.ui:ui")
    implementation("androidx.compose.ui:ui-tooling-preview")
    implementation("androidx.activity:activity-compose:1.9.3")
    implementation("androidx.core:core-ktx:1.13.1")
    implementation("androidx.lifecycle:lifecycle-runtime-ktx:2.8.7")
    implementation("androidx.lifecycle:lifecycle-viewmodel-compose:2.8.7")

    debugImplementation("androidx.compose.ui:ui-tooling")
}
