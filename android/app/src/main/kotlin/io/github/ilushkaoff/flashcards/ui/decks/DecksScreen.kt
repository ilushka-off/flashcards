package io.github.ilushkaoff.flashcards.ui.decks

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.filled.AccountCircle
import androidx.compose.material3.AssistChip
import androidx.compose.material3.Card
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.ExtendedFloatingActionButton
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.pulltorefresh.PullToRefreshBox
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import io.github.ilushkaoff.flashcards.core.api.ApiResult
import io.github.ilushkaoff.flashcards.core.data.DeckRepository
import io.github.ilushkaoff.flashcards.core.model.Deck
import io.github.ilushkaoff.flashcards.ui.common.ErrorBox
import io.github.ilushkaoff.flashcards.ui.common.LoadingBox
import io.github.ilushkaoff.flashcards.ui.common.userMessage
import kotlinx.coroutines.launch

data class DecksUiState(
    val decks: List<Deck>? = null,
    val error: String? = null,
    val refreshing: Boolean = false,
)

class DecksViewModel(private val repository: DeckRepository) : ViewModel() {
    var state by mutableStateOf(DecksUiState())
        private set

    /** Перечитывает список. Вызывается при каждом показе экрана, чтобы подтянуть изменения. */
    fun load() {
        if (state.refreshing) return
        state = state.copy(refreshing = true)
        viewModelScope.launch {
            state = when (val result = repository.list()) {
                is ApiResult.Success -> DecksUiState(decks = result.value)
                is ApiResult.Failure -> state.copy(refreshing = false, error = result.error.userMessage())
            }
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun DecksScreen(
    vm: DecksViewModel,
    onOpenDeck: (Long) -> Unit,
    onCreateDeck: () -> Unit,
    onOpenProfile: () -> Unit,
) {
    LaunchedEffect(Unit) { vm.load() }
    val state = vm.state

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Мои колоды") },
                actions = {
                    IconButton(onClick = onOpenProfile) {
                        Icon(Icons.Filled.AccountCircle, contentDescription = "Профиль")
                    }
                },
            )
        },
        floatingActionButton = {
            ExtendedFloatingActionButton(
                onClick = onCreateDeck,
                icon = { Icon(Icons.Filled.Add, contentDescription = null) },
                text = { Text("Колода") },
            )
        },
    ) { padding ->
        val decks = state.decks
        when {
            decks == null && state.error != null -> ErrorBox(state.error, vm::load, Modifier.padding(padding))
            decks == null -> LoadingBox(Modifier.padding(padding))
            else -> PullToRefreshBox(
                isRefreshing = state.refreshing,
                onRefresh = vm::load,
                modifier = Modifier
                    .fillMaxSize()
                    .padding(padding),
            ) {
                if (decks.isEmpty()) {
                    EmptyDecks()
                } else {
                    DeckList(decks, state.error, onOpenDeck)
                }
            }
        }
    }
}

@Composable
private fun DeckList(decks: List<Deck>, error: String?, onOpenDeck: (Long) -> Unit) {
    LazyColumn(
        contentPadding = PaddingValues(start = 16.dp, end = 16.dp, top = 8.dp, bottom = 96.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp),
        modifier = Modifier.fillMaxSize(),
    ) {
        if (error != null) {
            // Список уже был загружен, а обновить не вышло: показываем старый и ошибку над ним
            item { Text(error, color = MaterialTheme.colorScheme.error) }
        }
        items(decks, key = { it.id }) { deck ->
            DeckItem(deck, onClick = { onOpenDeck(deck.id) })
        }
    }
}

@Composable
private fun DeckItem(deck: Deck, onClick: () -> Unit) {
    Card(
        modifier = Modifier
            .fillMaxWidth()
            .clickable(onClick = onClick),
    ) {
        Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(6.dp)) {
            Text(deck.name, style = MaterialTheme.typography.titleMedium, maxLines = 1, overflow = TextOverflow.Ellipsis)
            if (deck.description.isNotEmpty()) {
                Text(
                    deck.description,
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    maxLines = 2,
                    overflow = TextOverflow.Ellipsis,
                )
            }
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp), verticalAlignment = Alignment.CenterVertically) {
                AssistChip(onClick = onClick, label = { Text(deck.language.title) })
                Text(
                    "Карточек: ${deck.totalCards} · к повторению: ${deck.dueCards}",
                    style = MaterialTheme.typography.bodySmall,
                )
            }
        }
    }
}

@Composable
private fun EmptyDecks() {
    // Скролл нужен, чтобы pull-to-refresh работал и на пустом экране
    Column(
        modifier = Modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(32.dp),
        verticalArrangement = Arrangement.Center,
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Text(
            "Колод пока нет. Создайте первую кнопкой «Колода».",
            style = MaterialTheme.typography.bodyLarge,
            textAlign = TextAlign.Center,
        )
    }
}
