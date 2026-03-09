package io.clawdroid.setup

import android.net.Uri
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.IconButtonDefaults
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import io.clawdroid.R
import io.clawdroid.core.ui.theme.DeepBlack
import io.clawdroid.core.ui.theme.NeonCyan
import io.clawdroid.core.ui.theme.TextPrimary
import io.clawdroid.core.ui.theme.TextSecondary

@Composable
fun SetupStep3WorkspaceScreen(viewModel: SetupViewModel) {
    val uiState by viewModel.uiState.collectAsState()

    val workspacePicker = rememberLauncherForActivityResult(
        ActivityResultContracts.OpenDocumentTree(),
    ) { uri: Uri? ->
        uri?.let { viewModel.onWorkspaceChange(uriToPath(it)) }
    }

    val dataDirPicker = rememberLauncherForActivityResult(
        ActivityResultContracts.OpenDocumentTree(),
    ) { uri: Uri? ->
        uri?.let { viewModel.onDataDirChange(uriToPath(it)) }
    }

    Column(
        modifier = Modifier
            .fillMaxSize()
            .padding(24.dp)
            .verticalScroll(rememberScrollState()),
        verticalArrangement = Arrangement.spacedBy(16.dp),
    ) {
        Spacer(Modifier.height(32.dp))

        Text(stringResource(R.string.setup_step_3_of_4), style = MaterialTheme.typography.labelMedium, color = TextSecondary)
        Text(stringResource(R.string.setup_workspace_title), style = MaterialTheme.typography.headlineMedium, color = TextPrimary)
        Text(
            stringResource(R.string.setup_workspace_description),
            style = MaterialTheme.typography.bodyMedium,
            color = TextSecondary,
        )

        Spacer(Modifier.height(8.dp))

        DirectoryField(
            value = uiState.workspace,
            onValueChange = viewModel::onWorkspaceChange,
            label = stringResource(R.string.setup_workspace_label),
            placeholder = "~/.clawdroid/workspace",
            supportingText = stringResource(R.string.setup_workspace_hint),
            onBrowse = { workspacePicker.launch(null) },
        )

        DirectoryField(
            value = uiState.dataDir,
            onValueChange = viewModel::onDataDirChange,
            label = stringResource(R.string.setup_data_dir_label),
            placeholder = "~/.clawdroid/data",
            supportingText = stringResource(R.string.setup_data_dir_hint),
            onBrowse = { dataDirPicker.launch(null) },
        )

        Spacer(Modifier.weight(1f))

        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
        ) {
            TextButton(onClick = { viewModel.skipStep(3) }) {
                Text(stringResource(R.string.setup_set_up_later), color = TextSecondary)
            }
            Button(
                onClick = { viewModel.nextStep(3) },
                colors = ButtonDefaults.buttonColors(
                    containerColor = NeonCyan,
                    contentColor = DeepBlack,
                ),
            ) {
                Text(stringResource(R.string.btn_next))
            }
        }
    }
}

@Composable
private fun DirectoryField(
    value: String,
    onValueChange: (String) -> Unit,
    label: String,
    placeholder: String,
    supportingText: String = "",
    onBrowse: () -> Unit,
) {
    OutlinedTextField(
        value = value,
        onValueChange = onValueChange,
        label = { Text(label, color = TextSecondary) },
        placeholder = { Text(placeholder, color = TextSecondary.copy(alpha = 0.5f)) },
        supportingText = if (supportingText.isNotEmpty()) {
            { Text(supportingText) }
        } else {
            null
        },
        singleLine = true,
        trailingIcon = {
            IconButton(
                onClick = onBrowse,
                colors = IconButtonDefaults.iconButtonColors(contentColor = NeonCyan),
            ) {
                Icon(
                    painter = painterResource(android.R.drawable.ic_menu_agenda),
                    contentDescription = stringResource(R.string.btn_browse),
                    modifier = Modifier.size(20.dp),
                )
            }
        },
        colors = setupFieldColors(),
        modifier = Modifier.fillMaxWidth(),
    )
}

private fun uriToPath(uri: Uri): String {
    // content://com.android.externalstorage.documents/tree/primary%3ADocuments
    // → /storage/emulated/0/Documents
    val docId = uri.lastPathSegment ?: return uri.toString()
    val parts = docId.split(":")
    return if (parts.size == 2 && parts[0] == "primary") {
        "/storage/emulated/0/${parts[1]}"
    } else {
        uri.path ?: uri.toString()
    }
}
