package io.clawdroid.assistant.actions

import android.Manifest
import android.content.BroadcastReceiver
import android.content.ContentUris
import android.content.ContentValues
import android.content.Context
import android.content.Intent
import android.content.IntentFilter
import android.provider.CalendarContract
import android.util.Log
import androidx.core.content.ContextCompat
import io.clawdroid.CalendarPickerActivity
import io.clawdroid.core.data.remote.dto.ToolRequest
import io.clawdroid.core.data.remote.dto.ToolResponse
import kotlinx.coroutines.suspendCancellableCoroutine
import kotlinx.coroutines.withTimeoutOrNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import java.net.HttpURLConnection
import java.net.URL
import java.text.SimpleDateFormat
import java.util.Locale
import java.util.TimeZone
import kotlin.coroutines.resume

class CalendarActionHandler : ActionHandler {
    override val supportedActions = setOf(
        "create_event", "query_events", "update_event",
        "delete_event", "list_calendars", "add_reminder"
    )

    override fun requiredPermissions(action: String): List<PermissionRequirement> = when (action) {
        "query_events", "list_calendars" ->
            listOf(PermissionRequirement.Runtime(Manifest.permission.READ_CALENDAR, "Calendar read access"))
        "create_event", "update_event", "delete_event", "add_reminder" ->
            listOf(PermissionRequirement.Runtime(Manifest.permission.WRITE_CALENDAR, "Calendar write access"))
        else -> emptyList()
    }

    private fun isoFormat(): SimpleDateFormat =
        SimpleDateFormat("yyyy-MM-dd'T'HH:mm:ss", Locale.US).apply {
            timeZone = TimeZone.getDefault()
        }

    private val writeActions = setOf("create_event", "update_event", "add_reminder")

    override suspend fun handle(request: ToolRequest, context: Context): ToolResponse {
        // For write actions without calendar_id, launch picker and save to config
        val resolvedRequest = if (request.action in writeActions && request.stringParam("calendar_id") == null) {
            val picked = pickCalendar(context)
                ?: return ToolResponse(request.requestId, false, error = "Calendar selection cancelled")
            saveCalendarIdToConfig(picked.first)
            request.withParam("calendar_id", picked.first)
        } else {
            request
        }

        return when (resolvedRequest.action) {
            "create_event" -> handleCreateEvent(resolvedRequest, context)
            "query_events" -> handleQueryEvents(resolvedRequest, context)
            "update_event" -> handleUpdateEvent(resolvedRequest, context)
            "delete_event" -> handleDeleteEvent(resolvedRequest, context)
            "list_calendars" -> handleListCalendars(resolvedRequest, context)
            "add_reminder" -> handleAddReminder(resolvedRequest, context)
            else -> ToolResponse(resolvedRequest.requestId, false, error = "Unknown calendar action")
        }
    }

    private suspend fun pickCalendar(context: Context): Pair<String, String>? {
        return withTimeoutOrNull(30_000L) {
            suspendCancellableCoroutine { cont ->
                val receiver = object : BroadcastReceiver() {
                    override fun onReceive(ctx: Context, intent: Intent) {
                        context.unregisterReceiver(this)
                        val cancelled = intent.getBooleanExtra(CalendarPickerActivity.EXTRA_CANCELLED, false)
                        if (cancelled) {
                            if (cont.isActive) cont.resume(null)
                        } else {
                            val id = intent.getStringExtra(CalendarPickerActivity.EXTRA_CALENDAR_ID).orEmpty()
                            val name = intent.getStringExtra(CalendarPickerActivity.EXTRA_CALENDAR_NAME).orEmpty()
                            if (cont.isActive) cont.resume(id to name)
                        }
                    }
                }

                val filter = IntentFilter(CalendarPickerActivity.ACTION_RESULT)
                ContextCompat.registerReceiver(
                    context, receiver, filter, ContextCompat.RECEIVER_NOT_EXPORTED
                )

                cont.invokeOnCancellation {
                    try { context.unregisterReceiver(receiver) } catch (_: IllegalArgumentException) {}
                }

                context.startActivity(CalendarPickerActivity.intent(context))
            }
        }
    }

    private fun saveCalendarIdToConfig(calendarId: String) {
        try {
            val url = URL("http://127.0.0.1:18790/api/config")
            val conn = url.openConnection() as HttpURLConnection
            conn.requestMethod = "PUT"
            conn.setRequestProperty("Content-Type", "application/json")
            conn.doOutput = true
            // Patch only the calendar_id field
            val body = """{"tools":{"android":{"calendar":{"calendar_id":"$calendarId"}}}}"""
            conn.outputStream.use { it.write(body.toByteArray()) }
            conn.responseCode // trigger the request
            conn.disconnect()
        } catch (e: Exception) {
            Log.w("CalendarActionHandler", "Failed to save calendar_id to config", e)
        }
    }

    private fun ToolRequest.withParam(key: String, value: String): ToolRequest {
        val currentParams = params ?: JsonObject(emptyMap())
        val newParams = JsonObject(currentParams + (key to JsonPrimitive(value)))
        return ToolRequest(requestId = requestId, action = action, params = newParams)
    }

    private fun handleCreateEvent(request: ToolRequest, context: Context): ToolResponse {
        val title = request.stringParam("title")
            ?: return ToolResponse(request.requestId, false, error = "title required")
        val startTimeStr = request.stringParam("start_time")
            ?: return ToolResponse(request.requestId, false, error = "start_time required")

        val fmt = isoFormat()
        val startMillis = try { fmt.parse(startTimeStr)?.time ?: 0L } catch (e: Exception) {
            return ToolResponse(request.requestId, false, error = "Invalid start_time format: ${e.message}")
        }

        val endMillis = request.stringParam("end_time")?.let {
            try { fmt.parse(it)?.time } catch (_: Exception) { null }
        } ?: (startMillis + 3600_000L) // default 1 hour

        val calendarId = request.stringParam("calendar_id")?.toLongOrNull()
            ?: getDefaultCalendarId(context)
            ?: return ToolResponse(request.requestId, false, error = "No calendar found. Please add a Google account first.")

        val values = ContentValues().apply {
            put(CalendarContract.Events.CALENDAR_ID, calendarId)
            put(CalendarContract.Events.TITLE, title)
            put(CalendarContract.Events.DTSTART, startMillis)
            put(CalendarContract.Events.DTEND, endMillis)
            put(CalendarContract.Events.EVENT_TIMEZONE, TimeZone.getDefault().id)
            request.stringParam("description")?.let {
                put(CalendarContract.Events.DESCRIPTION, it)
            }
            request.stringParam("location")?.let {
                put(CalendarContract.Events.EVENT_LOCATION, it)
            }
            request.boolParam("all_day")?.let {
                if (it) put(CalendarContract.Events.ALL_DAY, 1)
            }
        }

        return try {
            val uri = context.contentResolver.insert(CalendarContract.Events.CONTENT_URI, values)
            val eventId = uri?.lastPathSegment
            ToolResponse(request.requestId, true, result = "Event created: $title (ID: $eventId)")
        } catch (e: SecurityException) {
            ToolResponse(request.requestId, false, error = "Calendar permission not granted. Please grant WRITE_CALENDAR permission.")
        } catch (e: Exception) {
            ToolResponse(request.requestId, false, error = "Failed to create event: ${e.message}")
        }
    }

    private fun getDefaultCalendarId(context: Context): Long? {
        val projection = arrayOf(CalendarContract.Calendars._ID)
        val selection = "${CalendarContract.Calendars.IS_PRIMARY} = 1"
        val cursor = context.contentResolver.query(
            CalendarContract.Calendars.CONTENT_URI, projection, selection, null, null
        )
        cursor?.use {
            if (it.moveToFirst()) return it.getLong(0)
        }
        // Fallback: first writable calendar
        val fallbackCursor = context.contentResolver.query(
            CalendarContract.Calendars.CONTENT_URI,
            arrayOf(CalendarContract.Calendars._ID),
            "${CalendarContract.Calendars.CALENDAR_ACCESS_LEVEL} >= ${CalendarContract.Calendars.CAL_ACCESS_CONTRIBUTOR}",
            null, null
        )
        fallbackCursor?.use {
            if (it.moveToFirst()) return it.getLong(0)
        }
        return null
    }

    private fun handleQueryEvents(request: ToolRequest, context: Context): ToolResponse {
        val startTimeStr = request.stringParam("start_time")
            ?: return ToolResponse(request.requestId, false, error = "start_time required")
        val endTimeStr = request.stringParam("end_time")
            ?: return ToolResponse(request.requestId, false, error = "end_time required")

        val fmt = isoFormat()
        val startMillis = try { fmt.parse(startTimeStr)?.time ?: 0L } catch (e: Exception) {
            return ToolResponse(request.requestId, false, error = "Invalid start_time: ${e.message}")
        }
        val endMillis = try { fmt.parse(endTimeStr)?.time ?: 0L } catch (e: Exception) {
            return ToolResponse(request.requestId, false, error = "Invalid end_time: ${e.message}")
        }

        val projection = arrayOf(
            CalendarContract.Instances.EVENT_ID,
            CalendarContract.Instances.TITLE,
            CalendarContract.Instances.BEGIN,
            CalendarContract.Instances.END,
            CalendarContract.Instances.EVENT_LOCATION,
            CalendarContract.Instances.DESCRIPTION,
            CalendarContract.Instances.ALL_DAY
        )

        val uri = CalendarContract.Instances.CONTENT_URI.buildUpon()
            .appendPath(startMillis.toString())
            .appendPath(endMillis.toString())
            .build()

        val query = request.stringParam("query")
        val selection = if (query != null) "${CalendarContract.Instances.TITLE} LIKE ?" else null
        val selectionArgs = if (query != null) arrayOf("%$query%") else null

        return try {
            val cursor = context.contentResolver.query(uri, projection, selection, selectionArgs, "${CalendarContract.Instances.BEGIN} ASC")
            val events = buildString {
                appendLine("Events found: ${cursor?.count ?: 0}")
                cursor?.use {
                    while (it.moveToNext()) {
                        val eventId = it.getLong(0)
                        val title = it.getString(1) ?: "(no title)"
                        val begin = it.getLong(2)
                        val end = it.getLong(3)
                        val location = it.getString(4) ?: ""
                        val description = it.getString(5) ?: ""
                        val allDay = it.getInt(6) == 1

                        appendLine("---")
                        appendLine("ID: $eventId")
                        appendLine("Title: $title")
                        appendLine("Start: ${fmt.format(begin)}")
                        appendLine("End: ${fmt.format(end)}")
                        if (allDay) appendLine("All day: true")
                        if (location.isNotEmpty()) appendLine("Location: $location")
                        if (description.isNotEmpty()) appendLine("Description: $description")
                    }
                }
            }
            ToolResponse(request.requestId, true, result = events)
        } catch (e: SecurityException) {
            ToolResponse(request.requestId, false, error = "Calendar permission not granted. Please grant READ_CALENDAR permission.")
        } catch (e: Exception) {
            ToolResponse(request.requestId, false, error = "Failed to query events: ${e.message}")
        }
    }

    private fun handleUpdateEvent(request: ToolRequest, context: Context): ToolResponse {
        val eventId = request.stringParam("event_id")
            ?: return ToolResponse(request.requestId, false, error = "event_id required")

        val fmt = isoFormat()
        val values = ContentValues()
        request.stringParam("title")?.let {
            values.put(CalendarContract.Events.TITLE, it)
        }
        request.stringParam("description")?.let {
            values.put(CalendarContract.Events.DESCRIPTION, it)
        }
        request.stringParam("location")?.let {
            values.put(CalendarContract.Events.EVENT_LOCATION, it)
        }
        request.stringParam("start_time")?.let {
            try { fmt.parse(it)?.time } catch (_: Exception) { null }
        }?.let { values.put(CalendarContract.Events.DTSTART, it) }
        request.stringParam("end_time")?.let {
            try { fmt.parse(it)?.time } catch (_: Exception) { null }
        }?.let { values.put(CalendarContract.Events.DTEND, it) }

        if (values.size() == 0) {
            return ToolResponse(request.requestId, false, error = "No fields to update")
        }

        return try {
            val uri = ContentUris.withAppendedId(CalendarContract.Events.CONTENT_URI, eventId.toLong())
            val rows = context.contentResolver.update(uri, values, null, null)
            ToolResponse(request.requestId, true, result = "Updated $rows event(s)")
        } catch (e: SecurityException) {
            ToolResponse(request.requestId, false, error = "Calendar permission not granted. Please grant WRITE_CALENDAR permission.")
        } catch (e: Exception) {
            ToolResponse(request.requestId, false, error = "Failed to update event: ${e.message}")
        }
    }

    private fun handleDeleteEvent(request: ToolRequest, context: Context): ToolResponse {
        val eventId = request.stringParam("event_id")
            ?: return ToolResponse(request.requestId, false, error = "event_id required")

        return try {
            val uri = ContentUris.withAppendedId(CalendarContract.Events.CONTENT_URI, eventId.toLong())
            val rows = context.contentResolver.delete(uri, null, null)
            ToolResponse(request.requestId, true, result = "Deleted $rows event(s)")
        } catch (e: SecurityException) {
            ToolResponse(request.requestId, false, error = "Calendar permission not granted. Please grant WRITE_CALENDAR permission.")
        } catch (e: Exception) {
            ToolResponse(request.requestId, false, error = "Failed to delete event: ${e.message}")
        }
    }

    private fun handleListCalendars(request: ToolRequest, context: Context): ToolResponse {
        val projection = arrayOf(
            CalendarContract.Calendars._ID,
            CalendarContract.Calendars.CALENDAR_DISPLAY_NAME,
            CalendarContract.Calendars.ACCOUNT_NAME,
            CalendarContract.Calendars.ACCOUNT_TYPE,
            CalendarContract.Calendars.CALENDAR_COLOR
        )

        return try {
            val cursor = context.contentResolver.query(
                CalendarContract.Calendars.CONTENT_URI, projection, null, null, null
            )
            val calendars = buildString {
                appendLine("Calendars found: ${cursor?.count ?: 0}")
                cursor?.use {
                    while (it.moveToNext()) {
                        appendLine("---")
                        appendLine("ID: ${it.getLong(0)}")
                        appendLine("Name: ${it.getString(1)}")
                        appendLine("Account: ${it.getString(2)} (${it.getString(3)})")
                    }
                }
            }
            ToolResponse(request.requestId, true, result = calendars)
        } catch (e: SecurityException) {
            ToolResponse(request.requestId, false, error = "Calendar permission not granted. Please grant READ_CALENDAR permission.")
        } catch (e: Exception) {
            ToolResponse(request.requestId, false, error = "Failed to list calendars: ${e.message}")
        }
    }

    private fun handleAddReminder(request: ToolRequest, context: Context): ToolResponse {
        val eventId = request.stringParam("event_id")
            ?: return ToolResponse(request.requestId, false, error = "event_id required")
        val minutes = request.intParam("minutes")
            ?: return ToolResponse(request.requestId, false, error = "minutes required")

        val values = ContentValues().apply {
            put(CalendarContract.Reminders.EVENT_ID, eventId.toLong())
            put(CalendarContract.Reminders.MINUTES, minutes)
            put(CalendarContract.Reminders.METHOD, CalendarContract.Reminders.METHOD_ALERT)
        }

        return try {
            context.contentResolver.insert(CalendarContract.Reminders.CONTENT_URI, values)
            ToolResponse(request.requestId, true, result = "Reminder added: $minutes minutes before event")
        } catch (e: SecurityException) {
            ToolResponse(request.requestId, false, error = "Calendar permission not granted. Please grant WRITE_CALENDAR permission.")
        } catch (e: Exception) {
            ToolResponse(request.requestId, false, error = "Failed to add reminder: ${e.message}")
        }
    }
}
