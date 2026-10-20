package io.github.ilushkaoff.flashcards.ui.profile

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import io.github.ilushkaoff.flashcards.core.api.ApiResult
import io.github.ilushkaoff.flashcards.core.data.AuthRepository
import io.github.ilushkaoff.flashcards.core.model.User
import io.github.ilushkaoff.flashcards.ui.common.ErrorBox
import io.github.ilushkaoff.flashcards.ui.common.LoadingBox
import io.github.ilushkaoff.flashcards.ui.common.formatDate
import io.github.ilushkaoff.flashcards.ui.common.userMessage
import kotlinx.coroutines.launch

data class ProfileUiState(
    val user: User? = null,
    val error: String? = null,
    val loggingOut: Boolean = false,
)

class ProfileViewModel(private val repository: AuthRepository) : ViewModel() {
    var state by mutableStateOf(ProfileUiState())
        private set

    init {
        load()
    }

    fun load() {
        state = state.copy(error = null)
        viewModelScope.launch {
            state = when (val result = repository.me()) {
                is ApiResult.Success -> state.copy(user = result.value)
                is ApiResult.Failure -> state.copy(error = result.error.userMessage())
            }
        }
    }

    /** После выхода Session.loggedIn станет false, и NavHost откроет экран входа. */
    fun logout() {
        if (state.loggingOut) return
        state = state.copy(loggingOut = true)
        viewModelScope.launch { repository.logout() }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ProfileScreen(vm: ProfileViewModel, onBack: () -> Unit) {
    val state = vm.state

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Профиль") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Назад")
                    }
                },
            )
        },
    ) { padding ->
        val user = state.user
        when {
            user == null && state.error != null -> ErrorBox(state.error, vm::load, Modifier.padding(padding))
            user == null -> LoadingBox(Modifier.padding(padding))
            else -> Column(
                modifier = Modifier
                    .fillMaxSize()
                    .padding(padding)
                    .padding(16.dp),
                verticalArrangement = Arrangement.spacedBy(12.dp),
            ) {
                Text(user.email, style = MaterialTheme.typography.titleLarge)
                Text(
                    "С нами с ${formatDate(user.createdAt)}",
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
                OutlinedButton(
                    onClick = vm::logout,
                    enabled = !state.loggingOut,
                    modifier = Modifier.fillMaxWidth(),
                ) { Text("Выйти") }
            }
        }
    }
}
