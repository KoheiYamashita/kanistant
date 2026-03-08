package io.clawdroid.assistant.actions

import android.content.Context
import android.content.Intent
import android.net.Uri
import io.clawdroid.core.data.remote.dto.ToolRequest
import io.clawdroid.core.data.remote.dto.ToolResponse

class CommunicationActionHandler : ActionHandler {
    companion object {
        private val PHONE_NUMBER_RE = Regex("^[0-9+\\-() #*]+$")
        private val EMAIL_RE = Regex("^[^@\\s]+@[^@\\s]+\\.[^@\\s]+$")
    }

    override val supportedActions = setOf("dial", "compose_sms", "compose_email")

    override suspend fun handle(request: ToolRequest, context: Context): ToolResponse {
        return when (request.action) {
            "dial" -> handleDial(request, context)
            "compose_sms" -> handleComposeSms(request, context)
            "compose_email" -> handleComposeEmail(request, context)
            else -> ToolResponse(request.requestId, false, error = "Unknown communication action")
        }
    }

    private fun handleDial(request: ToolRequest, context: Context): ToolResponse {
        val phone = request.stringParam("phone_number")
            ?: return ToolResponse(request.requestId, false, error = "phone_number required")
        if (!PHONE_NUMBER_RE.matches(phone)) {
            return ToolResponse(request.requestId, false, error = "Invalid phone number: only digits, +, -, (), spaces, #, * are allowed")
        }

        val intent = Intent(Intent.ACTION_DIAL).apply {
            data = Uri.parse("tel:$phone")
            addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
        }
        return launchActivity(request, context, intent, "Dialer opened with: $phone")
    }

    private fun handleComposeSms(request: ToolRequest, context: Context): ToolResponse {
        val phone = request.stringParam("phone_number")
            ?: return ToolResponse(request.requestId, false, error = "phone_number required")
        if (!PHONE_NUMBER_RE.matches(phone)) {
            return ToolResponse(request.requestId, false, error = "Invalid phone number: only digits, +, -, (), spaces, #, * are allowed")
        }

        val intent = Intent(Intent.ACTION_SENDTO).apply {
            data = Uri.parse("smsto:$phone")
            request.stringParam("message")?.let {
                putExtra("sms_body", it)
            }
            addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
        }
        return launchActivity(request, context, intent, "SMS compose opened for: $phone")
    }

    private fun handleComposeEmail(request: ToolRequest, context: Context): ToolResponse {
        val to = request.stringParam("to")
            ?: return ToolResponse(request.requestId, false, error = "to required")
        if (!EMAIL_RE.matches(to)) {
            return ToolResponse(request.requestId, false, error = "Invalid email address: $to")
        }

        val intent = Intent(Intent.ACTION_SENDTO).apply {
            data = Uri.parse("mailto:$to")
            request.stringParam("subject")?.let {
                putExtra(Intent.EXTRA_SUBJECT, it)
            }
            request.stringParam("body")?.let {
                putExtra(Intent.EXTRA_TEXT, it)
            }
            addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
        }
        return launchActivity(request, context, intent, "Email compose opened for: $to")
    }
}
