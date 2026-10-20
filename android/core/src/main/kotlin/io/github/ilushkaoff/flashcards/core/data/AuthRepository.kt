package io.github.ilushkaoff.flashcards.core.data

import io.github.ilushkaoff.flashcards.core.api.ApiResult
import io.github.ilushkaoff.flashcards.core.api.FlashcardsApi
import io.github.ilushkaoff.flashcards.core.api.apiCall
import io.github.ilushkaoff.flashcards.core.api.map
import io.github.ilushkaoff.flashcards.core.auth.Session
import io.github.ilushkaoff.flashcards.core.auth.Tokens
import io.github.ilushkaoff.flashcards.core.model.LoginRequest
import io.github.ilushkaoff.flashcards.core.model.RefreshRequest
import io.github.ilushkaoff.flashcards.core.model.RegisterRequest
import io.github.ilushkaoff.flashcards.core.model.User
import kotlinx.serialization.json.Json

class AuthRepository(
    private val api: FlashcardsApi,
    private val session: Session,
    private val json: Json,
) {
    suspend fun login(email: String, password: String): ApiResult<Unit> =
        apiCall(json) { api.login(LoginRequest(email.trim(), password)) }
            .map { session.save(Tokens(it.accessToken, it.refreshToken)) }

    /** Регистрирует пользователя и сразу входит, чтобы не вводить пароль второй раз. */
    suspend fun register(email: String, password: String): ApiResult<Unit> =
        when (val created = apiCall(json) { api.register(RegisterRequest(email.trim(), password)) }) {
            is ApiResult.Failure -> created
            is ApiResult.Success -> login(email, password)
        }

    suspend fun me(): ApiResult<User> = apiCall(json) { api.me() }

    /** Отзывает refresh-токен на сервере, если получится, и в любом случае забывает токены. */
    suspend fun logout() {
        session.tokens()?.let { tokens ->
            apiCall(json) { api.logout(RefreshRequest(tokens.refreshToken)) }
        }
        session.clear()
    }
}
