package io.gosicp.tableauxeink

import android.Manifest
import android.content.pm.PackageManager
import android.os.Build
import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.runtime.snapshots.SnapshotStateList
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.core.content.ContextCompat
import einkapp.Einkapp
import einkapp.LogObserver
import io.gosicp.androidsvc.GoServiceBridge

/**
 * Minimal host activity. Boots the Go bridge, binds the in-memory log
 * buffer to a Compose state list, and offers Start/Stop server controls.
 */
class MainActivity : ComponentActivity() {

    private val logs: SnapshotStateList<String> = mutableStateListOf()

    private val notifPermLauncher = registerForActivityResult(
        ActivityResultContracts.RequestPermission()
    ) { /* user response — we just continue, the user can re-trigger */ }

    private val observer = object : LogObserver {
        override fun onEntry(unixMillis: Long, level: Int, msg: String) {
            // Compose state list mutations must run on the main thread.
            runOnUiThread {
                logs.add("$msg")
                while (logs.size > 500) logs.removeAt(0)
            }
        }
    }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)

        // 1) Bring up the JNI bridge so the foreground service can talk back to Go.
        GoServiceBridge.init(applicationContext)

        // 2) Bootstrap the Go side. filesDir is writable for cert/token persistence.
        Einkapp.init(filesDir.absolutePath)
        Einkapp.registerLogObserver(observer)

        // 3) Android 13 runtime permission for the foreground notification.
        if (Build.VERSION.SDK_INT >= 33) {
            val granted = ContextCompat.checkSelfPermission(
                this, Manifest.permission.POST_NOTIFICATIONS
            ) == PackageManager.PERMISSION_GRANTED
            if (!granted) {
                notifPermLauncher.launch(Manifest.permission.POST_NOTIFICATIONS)
            }
        }

        setContent {
            MaterialTheme(colorScheme = lightColorScheme()) {
                AppScreen(logs, ::startServer, ::stopServer)
            }
        }
    }

    private fun startServer() {
        try { Einkapp.start() } catch (t: Throwable) {
            // Surface the error in the log buffer so the user sees it.
            logs.add("start failed: ${t.message}")
        }
    }

    private fun stopServer() {
        try { Einkapp.stop() } catch (t: Throwable) {
            logs.add("stop failed: ${t.message}")
        }
    }
}

@Composable
private fun AppScreen(
    logs: List<String>,
    onStart: () -> Unit,
    onStop: () -> Unit,
) {
    var running by remember { mutableStateOf(false) }
    var port by remember { mutableStateOf(0) }
    var token by remember { mutableStateOf("") }
    var fingerprint by remember { mutableStateOf("") }
    var connectUri by remember { mutableStateOf("") }

    // Cheap polling — gomobile bind isn't trivially Flow-friendly.
    LaunchedEffect(Unit) {
        while (true) {
            running = einkapp.Einkapp.isRunning()
            port = einkapp.Einkapp.port()
            token = einkapp.Einkapp.token()
            fingerprint = einkapp.Einkapp.fingerprint()
            connectUri = einkapp.Einkapp.connectURI("")
            kotlinx.coroutines.delay(800)
        }
    }

    Surface(modifier = Modifier.fillMaxSize()) {
        Column(modifier = Modifier.padding(16.dp)) {
            Text("tableaux-eink-go", style = MaterialTheme.typography.titleLarge)
            Spacer(Modifier.height(12.dp))
            Row {
                Button(onClick = onStart, enabled = !running) { Text("Start") }
                Spacer(Modifier.width(8.dp))
                Button(onClick = onStop, enabled = running) { Text("Stop") }
            }
            Spacer(Modifier.height(12.dp))
            Text("running: $running")
            Text("port: $port")
            Text("fp: $fingerprint", maxLines = 2, overflow = TextOverflow.Ellipsis)
            Text("token: $token", maxLines = 1, overflow = TextOverflow.Ellipsis)
            Text("connect: $connectUri", maxLines = 2, overflow = TextOverflow.Ellipsis)
            Divider(Modifier.padding(vertical = 12.dp))
            Text("Logs", style = MaterialTheme.typography.titleMedium)
            LazyColumn(modifier = Modifier.weight(1f)) {
                items(logs) { line ->
                    Text(line, style = MaterialTheme.typography.bodySmall, maxLines = 1, overflow = TextOverflow.Ellipsis)
                }
            }
        }
    }
}
