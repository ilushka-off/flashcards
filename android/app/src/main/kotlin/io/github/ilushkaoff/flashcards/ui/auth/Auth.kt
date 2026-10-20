package io.github.ilushkaoff.flashcards.ui.auth

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.unit.dp
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import io.github.ilushkaoff.flashcards.core.api.ApiResult
import io.github.ilushkaoff.flashcards.core.data.AuthRepository
import io.github.ilushkaoff.flashcards.core.validation.validateEmail
import io.github.ilushkaoff.flashcards.core.validation.validatePassword
import io.github.ilushkaoff.flashcards.ui.common.userMessage
import kotlinx.coroutines.launch

enum class AuthMode { LOGIN, REGISTER }

data class AuthUiState(
    val email: String = "",
    val password: String = "",
    val passwordRepeat: String = "",
    val emailError: String? = null,
    val passwordError: String? = null,
    val repeatError: String? = null,
    val error: String? = null,
    val loading: Boolean = false,
)

/** Вход и регистрация. После успеха Session.loggedIn становится true, и NavHost сам уводит на колоды. */
class AuthViewModel(
    val mode: AuthMode,
    private val repository: AuthRepository,
) : ViewModel() {
    var state by mutableStateOf(AuthUiState())
        private set

    fun onEmail(value: String) {
        state = state.copy(email = value, emailError = null, error = null)
    }

    fun onPassword(value: String) {
        state = state.copy(password = value, passwordError = null, error = null)
    }

    fun onPasswordRepeat(value: String) {
        state = state.copy(passwordRepeat = value, repeatError = null, error = null)
    }

    fun submit() {
        if (state.loading) return
        val emailError = validateEmail(state.email)
        val passwordError = validatePassword(state.password)
        val repeatError = if (mode == AuthMode.REGISTER && state.password != state.passwordRepeat) {
            "Пароли не совпадают"
        } else {
            null
        }
        if (emailError != null || passwordError != null || repeatError != null) {
            state = state.copy(emailError = emailError, passwordError = passwordError, repeatError = repeatError)
            return
        }

        state = state.copy(loading = true, error = null)
        viewModelScope.launch {
            val result = when (mode) {
                AuthMode.LOGIN -> repository.login(state.email, state.password)
                AuthMode.REGISTER -> repository.register(state.email, state.password)
            }
            state = when (result) {
                is ApiResult.Success -> state.copy(loading = false)
                is ApiResult.Failure -> state.copy(loading = false, error = result.error.userMessage())
            }
        }
    }
}

@Composable
fun AuthScreen(vm: AuthViewModel, onSwitchMode: () -> Unit) {
    val state = vm.state
    val isLogin = vm.mode == AuthMode.LOGIN

    Scaffold { padding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding)
                .imePadding()
                .verticalScroll(rememberScrollState())
                .padding(24.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp, Alignment.CenterVertically),
            horizontalAlignment = Alignment.CenterHorizontally,
        ) {
            Text("Flashcards", style = MaterialTheme.typography.displaySmall)
            Text(
                if (isLogin) "Вход" else "Регистрация",
                style = MaterialTheme.typography.titleLarge,
            )

            OutlinedTextField(
                value = state.email,
                onValueChange = vm::onEmail,
                label = { Text("Email") },
                singleLine = true,
                isError = state.emailError != null,
                supportingText = { state.emailError?.let { Text(it) } },
                keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Email, imeAction = ImeAction.Next),
                modifier = Modifier.fillMaxWidth(),
            )
            OutlinedTextField(
                value = state.password,
                onValueChange = vm::onPassword,
                label = { Text("Пароль") },
                singleLine = true,
                visualTransformation = PasswordVisualTransformation(),
                isError = state.passwordError != null,
                supportingText = { state.passwordError?.let { Text(it) } },
                keyboardOptions = KeyboardOptions(
                    keyboardType = KeyboardType.Password,
                    imeAction = if (isLogin) ImeAction.Done else ImeAction.Next,
                ),
                keyboardActions = KeyboardActions(onDone = { vm.submit() }),
                modifier = Modifier.fillMaxWidth(),
            )
            if (!isLogin) {
                OutlinedTextField(
                    value = state.passwordRepeat,
                    onValueChange = vm::onPasswordRepeat,
                    label = { Text("Пароль ещё раз") },
                    singleLine = true,
                    visualTransformation = PasswordVisualTransformation(),
                    isError = state.repeatError != null,
                    supportingText = { state.repeatError?.let { Text(it) } },
                    keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Password, imeAction = ImeAction.Done),
                    keyboardActions = KeyboardActions(onDone = { vm.submit() }),
                    modifier = Modifier.fillMaxWidth(),
                )
            }

            state.error?.let {
                Text(it, color = MaterialTheme.colorScheme.error, style = MaterialTheme.typography.bodyMedium)
            }

            Button(
                onClick = vm::submit,
                enabled = !state.loading,
                modifier = Modifier.fillMaxWidth(),
            ) {
                if (state.loading) {
                    CircularProgressIndicator(modifier = Modifier.size(20.dp), strokeWidth = 2.dp)
                } else {
                    Text(if (isLogin) "Войти" else "Зарегистрироваться")
                }
            }
            TextButton(onClick = onSwitchMode, enabled = !state.loading) {
                Text(if (isLogin) "Нет аккаунта? Зарегистрироваться" else "Уже есть аккаунт? Войти")
            }
        }
    }
}
