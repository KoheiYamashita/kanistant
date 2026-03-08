package io.clawdroid.assistant

import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.content.IntentFilter
import android.content.pm.PackageManager
import androidx.core.content.ContextCompat
import io.clawdroid.PermissionRequestActivity
import kotlinx.coroutines.suspendCancellableCoroutine
import kotlinx.coroutines.withTimeoutOrNull
import java.util.concurrent.atomic.AtomicBoolean
import kotlin.coroutines.resume

class PermissionRequester(private val context: Context) {

    suspend fun request(permission: String): Boolean {
        if (ContextCompat.checkSelfPermission(context, permission)
            == PackageManager.PERMISSION_GRANTED
        ) {
            return true
        }

        return withTimeoutOrNull(12_000L) {
            suspendCancellableCoroutine { cont ->
                val unregistered = AtomicBoolean(false)
                val receiver = object : BroadcastReceiver() {
                    override fun onReceive(ctx: Context, intent: Intent) {
                        val perm = intent.getStringExtra(PermissionRequestActivity.EXTRA_PERMISSION)
                        if (perm == permission) {
                            if (!unregistered.compareAndSet(false, true)) return
                            context.unregisterReceiver(this)
                            val granted = intent.getBooleanExtra(
                                PermissionRequestActivity.EXTRA_GRANTED, false
                            )
                            if (cont.isActive) cont.resume(granted)
                        }
                    }
                }

                val filter = IntentFilter(PermissionRequestActivity.ACTION_RESULT)
                ContextCompat.registerReceiver(
                    context, receiver, filter, ContextCompat.RECEIVER_NOT_EXPORTED
                )

                cont.invokeOnCancellation {
                    if (unregistered.compareAndSet(false, true)) {
                        try {
                            context.unregisterReceiver(receiver)
                        } catch (_: IllegalArgumentException) {
                            // already unregistered
                        }
                    }
                }

                context.startActivity(
                    PermissionRequestActivity.intent(context, permission)
                )
            }
        } == true
    }
}
