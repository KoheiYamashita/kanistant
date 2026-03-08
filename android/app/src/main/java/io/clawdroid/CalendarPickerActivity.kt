package io.clawdroid

import android.content.Context
import android.content.Intent
import android.os.Bundle
import android.provider.CalendarContract
import android.app.AlertDialog
import androidx.activity.ComponentActivity

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
            else -> showPickerDialog(calendars)
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

    private fun showPickerDialog(calendars: List<CalendarInfo>) {
        val names = calendars.map { it.displayName }.toTypedArray()
        AlertDialog.Builder(this)
            .setTitle("Select Calendar")
            .setItems(names) { _, which ->
                val selected = calendars[which]
                broadcastResult(calendarId = selected.id, calendarName = selected.displayName)
                finish()
            }
            .setOnCancelListener {
                broadcastResult(cancelled = true)
                finish()
            }
            .show()
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
