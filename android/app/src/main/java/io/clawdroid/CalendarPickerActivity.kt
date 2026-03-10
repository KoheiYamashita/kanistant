package io.clawdroid

import android.content.Context
import android.content.Intent
import android.os.Bundle
import android.provider.CalendarContract
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import io.clawdroid.core.ui.theme.ClawDroidTheme
import io.clawdroid.core.ui.theme.GlassBorder
import io.clawdroid.core.ui.theme.NeonCyan
import io.clawdroid.core.ui.theme.TextPrimary
import io.clawdroid.core.ui.theme.TextSecondary
import io.clawdroid.core.ui.theme.DarkCard

class CalendarPickerActivity : ComponentActivity() {

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)

        val calendars = queryWritableCalendars()
        when {
            calendars.isEmpty() -> {
                broadcastResult(cancelled = true)
                finish()
            }
            calendars.size == 1 -> {
                val (id, name) = calendars.first()
                broadcastResult(calendarId = id, calendarName = name)
                finish()
            }
            else -> setContent {
                ClawDroidTheme {
                    CalendarPickerDialog(
                        calendars = calendars,
                        onSelect = { cal ->
                            broadcastResult(calendarId = cal.id, calendarName = cal.displayName)
                            finish()
                        },
                        onCancel = {
                            broadcastResult(cancelled = true)
                            finish()
                        }
                    )
                }
            }
        }
    }

    private data class CalendarInfo(val id: Long, val displayName: String)

    private fun queryWritableCalendars(): List<CalendarInfo> {
        val projection = arrayOf(
            CalendarContract.Calendars._ID,
            CalendarContract.Calendars.CALENDAR_DISPLAY_NAME,
        )
        val selection =
            "${CalendarContract.Calendars.CALENDAR_ACCESS_LEVEL} >= ${CalendarContract.Calendars.CAL_ACCESS_CONTRIBUTOR}"
        val cursor = contentResolver.query(
            CalendarContract.Calendars.CONTENT_URI, projection, selection, null, null
        ) ?: return emptyList()

        return cursor.use {
            val result = mutableListOf<CalendarInfo>()
            while (it.moveToNext()) {
                result += CalendarInfo(it.getLong(0), it.getString(1) ?: "")
            }
            result
        }
    }

    @Composable
    private fun CalendarPickerDialog(
        calendars: List<CalendarInfo>,
        onSelect: (CalendarInfo) -> Unit,
        onCancel: () -> Unit,
    ) {
        AlertDialog(
            onDismissRequest = onCancel,
            containerColor = DarkCard,
            title = {
                Text(stringResource(R.string.calendar_picker_title), color = NeonCyan)
            },
            text = {
                Surface(
                    shape = androidx.compose.foundation.shape.RoundedCornerShape(8.dp),
                    border = BorderStroke(1.dp, GlassBorder),
                    color = DarkCard,
                ) {
                    Column(
                        modifier = Modifier
                            .fillMaxWidth()
                            .verticalScroll(rememberScrollState())
                    ) {
                        calendars.forEachIndexed { index, cal ->
                            Text(
                                text = cal.displayName,
                                color = TextPrimary,
                                modifier = Modifier
                                    .fillMaxWidth()
                                    .clickable { onSelect(cal) }
                                    .padding(horizontal = 16.dp, vertical = 14.dp)
                            )
                            if (index < calendars.lastIndex) {
                                HorizontalDivider(color = GlassBorder)
                            }
                        }
                    }
                }
            },
            confirmButton = {},
            dismissButton = {
                TextButton(onClick = onCancel) {
                    Text(stringResource(R.string.action_cancel), color = TextSecondary)
                }
            },
        )
    }

    private fun broadcastResult(
        calendarId: Long = -1,
        calendarName: String = "",
        cancelled: Boolean = false,
    ) {
        sendBroadcast(
            Intent(ACTION_RESULT)
                .setPackage(packageName)
                .putExtra(EXTRA_CALENDAR_ID, calendarId.toString())
                .putExtra(EXTRA_CALENDAR_NAME, calendarName)
                .putExtra(EXTRA_CANCELLED, cancelled)
        )
    }

    companion object {
        const val ACTION_RESULT = "io.clawdroid.CALENDAR_PICKER_RESULT"
        const val EXTRA_CALENDAR_ID = "calendar_id"
        const val EXTRA_CALENDAR_NAME = "calendar_name"
        const val EXTRA_CANCELLED = "cancelled"

        fun intent(context: Context): Intent =
            Intent(context, CalendarPickerActivity::class.java)
                .addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
    }
}
