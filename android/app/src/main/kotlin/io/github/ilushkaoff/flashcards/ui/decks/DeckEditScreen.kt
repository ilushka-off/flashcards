package io.github.ilushkaoff.flashcards.ui.decks

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.FilterChip
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import io.github.ilushkaoff.flashcards.core.api.ApiResult
import io.github.ilushkaoff.flashcards.core.data.DeckRepository
import io.github.ilushkaoff.flashcards.core.model.Deck
import io.github.ilushkaoff.flashcards.core.model.Language
import io.github.ilushkaoff.flashcards.core.validation.DECK_DESCRIPTION_MAX
import io.github.ilushkaoff.flashcards.core.validation.DECK_NAME_MAX
import io.github.ilushkaoff.flashcards.core.validation.validateDeckDescription
import io.github.ilushkaoff.flashcards.core.validation.validateDeckName
import io.github.ilushkaoff.flashcards.ui.common.ErrorBox
import io.github.ilushkaoff.flashcards.ui.common.LoadingBox
import io.github.ilushkaoff.flashcards.ui.common.userMessage
import kotlinx.coroutines.launch

data class DeckEditUiState(
    val original: Deck? = null,
    val loading: Boolean = false,
    val loadError: String? = null,
    val name: String = "",
    val description: String = "",
    val language: Language = Language.EN,
    val nameError: String? = null,
    val descriptionError: String? = null,
    val error: String? = null,
    val saving: Boolean = false,
    val savedId: Long? = null,
)

/** Создание (id = 0) и редактирование колоды. Язык задаётся только при создании. */
class DeckEditViewModel(
    private val id: Long,
    private val repository: DeckRepository,
) : ViewModel() {
    val isNew = id == 0L

    var state by mutableStateOf(DeckEditUiState(loading = !isNew))
        private set

    init {
        if (!isNew) load()
    }

    fun load() {
        state = state.copy(loading = true, loadError = null)
        viewModelScope.launch {
            state = when (val result = repository.get(id)) {
                is ApiResult.Success -> state.copy(
                    loading = false,
                    original = result.value,
                    name = result.value.name,
                    description = result.value.description,
                    language = result.value.language,
                )
                is ApiResult.Failure -> state.copy(loading = false, loadError = result.error.userMessage())
            }
        }
    }

    fun onName(value: String) {
        state = state.copy(name = value, nameError = null, error = null)
    }

    fun onDescription(value: String) {
        state = state.copy(description = value, descriptionError = null, error = null)
    }

    fun onLanguage(value: Language) {
        if (isNew) state = state.copy(language = value)
    }

    fun save() {
        if (state.saving) return
        val nameError = validateDeckName(state.name)
        val descriptionError = validateDeckDescription(state.description)
        if (nameError != null || descriptionError != null) {
            state = state.copy(nameError = nameError, descriptionError = descriptionError)
            return
        }

        state = state.copy(saving = true, error = null)
        viewModelScope.launch {
            val original = state.original
            val result = if (original == null) {
                repository.create(state.name, state.language, state.description)
            } else {
                repository.update(original, state.name, state.description)
            }
            state = when (result) {
                is ApiResult.Success -> state.copy(saving = false, savedId = result.value.id)
                is ApiResult.Failure -> state.copy(saving = false, error = result.error.userMessage())
            }
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun DeckEditScreen(
    vm: DeckEditViewModel,
    onBack: () -> Unit,
    onSaved: (Long) -> Unit,
) {
    val state = vm.state
    LaunchedEffect(state.savedId) { state.savedId?.let(onSaved) }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text(if (vm.isNew) "Новая колода" else "Изменить колоду") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Назад")
                    }
                },
            )
        },
    ) { padding ->
        when {
            state.loadError != null -> ErrorBox(state.loadError, vm::load, Modifier.padding(padding))
            state.loading -> LoadingBox(Modifier.padding(padding))
            else -> DeckForm(vm, state, Modifier.padding(padding))
        }
    }
}

@Composable
private fun DeckForm(vm: DeckEditViewModel, state: DeckEditUiState, modifier: Modifier = Modifier) {
    Column(
        modifier = modifier
            .fillMaxSize()
            .imePadding()
            .verticalScroll(rememberScrollState())
            .padding(16.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        OutlinedTextField(
            value = state.name,
            onValueChange = vm::onName,
            label = { Text("Название") },
            singleLine = true,
            isError = state.nameError != null,
            supportingText = { Text(state.nameError ?: "${state.name.trim().length} / $DECK_NAME_MAX") },
            modifier = Modifier.fillMaxWidth(),
        )

        Text("Язык", style = MaterialTheme.typography.titleSmall)
        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            Language.entries.forEach { language ->
                FilterChip(
                    selected = state.language == language,
                    onClick = { vm.onLanguage(language) },
                    enabled = vm.isNew,
                    label = { Text(language.title) },
                )
            }
        }
        if (!vm.isNew) {
            Text(
                "Язык колоды после создания не меняется.",
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
        }

        OutlinedTextField(
            value = state.description,
            onValueChange = vm::onDescription,
            label = { Text("Описание (необязательно)") },
            minLines = 3,
            isError = state.descriptionError != null,
            supportingText = {
                Text(state.descriptionError ?: "${state.description.trim().length} / $DECK_DESCRIPTION_MAX")
            },
            modifier = Modifier.fillMaxWidth(),
        )

        state.error?.let { Text(it, color = MaterialTheme.colorScheme.error) }

        Button(onClick = vm::save, enabled = !state.saving, modifier = Modifier.fillMaxWidth()) {
            if (state.saving) {
                CircularProgressIndicator(modifier = Modifier.size(20.dp), strokeWidth = 2.dp)
            } else {
                Text(if (vm.isNew) "Создать" else "Сохранить")
            }
        }
    }
}
