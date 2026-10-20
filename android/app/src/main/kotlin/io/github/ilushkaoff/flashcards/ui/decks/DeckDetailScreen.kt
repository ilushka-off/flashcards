package io.github.ilushkaoff.flashcards.ui.decks

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Delete
import androidx.compose.material.icons.filled.Edit
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Card
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import io.github.ilushkaoff.flashcards.core.api.ApiResult
import io.github.ilushkaoff.flashcards.core.data.DeckRepository
import io.github.ilushkaoff.flashcards.core.model.Deck
import io.github.ilushkaoff.flashcards.ui.common.ErrorBox
import io.github.ilushkaoff.flashcards.ui.common.LoadingBox
import io.github.ilushkaoff.flashcards.ui.common.formatDate
import io.github.ilushkaoff.flashcards.ui.common.userMessage
import kotlinx.coroutines.launch

data class DeckDetailUiState(
    val deck: Deck? = null,
    val error: String? = null,
    val deleting: Boolean = false,
    val deleted: Boolean = false,
)

class DeckDetailViewModel(
    private val id: Long,
    private val repository: DeckRepository,
) : ViewModel() {
    var state by mutableStateOf(DeckDetailUiState())
        private set

    fun load() {
        viewModelScope.launch {
            state = when (val result = repository.get(id)) {
                is ApiResult.Success -> state.copy(deck = result.value, error = null)
                is ApiResult.Failure -> state.copy(error = result.error.userMessage())
            }
        }
    }

    fun delete() {
        if (state.deleting) return
        state = state.copy(deleting = true, error = null)
        viewModelScope.launch {
            state = when (val result = repository.delete(id)) {
                is ApiResult.Success -> state.copy(deleting = false, deleted = true)
                is ApiResult.Failure -> state.copy(deleting = false, error = result.error.userMessage())
            }
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun DeckDetailScreen(
    vm: DeckDetailViewModel,
    onBack: () -> Unit,
    onEdit: () -> Unit,
    onDeleted: () -> Unit,
) {
    // Перечитываем при каждом показе: после редактирования данные уже другие
    LaunchedEffect(Unit) { vm.load() }
    val state = vm.state
    LaunchedEffect(state.deleted) { if (state.deleted) onDeleted() }
    var confirmDelete by remember { mutableStateOf(false) }

    Scaffold(
        topBar = {
            TopAppBar(
                title = {
                    Text(state.deck?.name ?: "Колода", maxLines = 1, overflow = TextOverflow.Ellipsis)
                },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Назад")
                    }
                },
                actions = {
                    if (state.deck != null) {
                        IconButton(onClick = onEdit) {
                            Icon(Icons.Filled.Edit, contentDescription = "Изменить")
                        }
                        IconButton(onClick = { confirmDelete = true }, enabled = !state.deleting) {
                            Icon(Icons.Filled.Delete, contentDescription = "Удалить")
                        }
                    }
                },
            )
        },
    ) { padding ->
        val deck = state.deck
        when {
            deck == null && state.error != null -> ErrorBox(state.error, vm::load, Modifier.padding(padding))
            deck == null -> LoadingBox(Modifier.padding(padding))
            else -> DeckInfo(deck, state.error, Modifier.padding(padding))
        }
    }

    if (confirmDelete) {
        AlertDialog(
            onDismissRequest = { confirmDelete = false },
            title = { Text("Удалить колоду?") },
            text = { Text("Колода, все её карточки и история повторений удалятся без возможности восстановления.") },
            confirmButton = {
                TextButton(onClick = {
                    confirmDelete = false
                    vm.delete()
                }) { Text("Удалить", color = MaterialTheme.colorScheme.error) }
            },
            dismissButton = {
                TextButton(onClick = { confirmDelete = false }) { Text("Отмена") }
            },
        )
    }
}

@Composable
private fun DeckInfo(deck: Deck, error: String?, modifier: Modifier = Modifier) {
    Column(
        modifier = modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(16.dp),
        verticalArrangement = Arrangement.spacedBy(16.dp),
    ) {
        error?.let { Text(it, color = MaterialTheme.colorScheme.error) }
        Card {
            Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                InfoRow("Язык", deck.language.title)
                InfoRow("Карточек", deck.totalCards.toString())
                InfoRow("К повторению", deck.dueCards.toString())
                InfoRow("Создана", formatDate(deck.createdAt))
            }
        }
        if (deck.description.isNotEmpty()) {
            Text("Описание", style = MaterialTheme.typography.titleSmall)
            Text(deck.description, style = MaterialTheme.typography.bodyLarge)
        }
        Text(
            "Карточки и повторение появятся в следующей версии приложения.",
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
    }
}

@Composable
private fun InfoRow(label: String, value: String) {
    Text(
        "$label: $value",
        style = MaterialTheme.typography.bodyLarge,
    )
}
