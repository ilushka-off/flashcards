package io.github.ilushkaoff.flashcards.core.auth

import io.github.ilushkaoff.flashcards.core.api.RefreshApi
import io.github.ilushkaoff.flashcards.core.model.RefreshRequest
import okhttp3.Authenticator
import okhttp3.Request
import okhttp3.Response
import okhttp3.Route
import java.io.IOException

/**
 * На 401 обновляет пару токенов через POST /auth/refresh и повторяет запрос.
 * Если refresh не удался, сессия очищается и UI уходит на экран входа.
 * Несколько параллельных 401 делают один refresh: остальные увидят уже новый токен.
 */
class TokenAuthenticator(
    private val session: Session,
    private val refreshApi: RefreshApi,
) : Authenticator {
    override fun authenticate(route: Route?, response: Response): Request? {
        val sentAccess = response.request.header("Authorization")?.removePrefix("Bearer ")
            ?: return null // запрос без токена (логин, регистрация): обновлять нечего
        if (responseCount(response) > 1) {
            session.clear()
            return null
        }

        val fresh = session.locked {
            val current = session.tokens() ?: return@locked null
            if (current.accessToken != sentAccess) {
                // Пока ждали блокировку, другой поток уже обновил токены
                return@locked current
            }
            refresh(current.refreshToken)?.also(session::save)
        }

        if (fresh == null) {
            session.clear()
            return null
        }
        return response.request.newBuilder()
            .header("Authorization", "Bearer ${fresh.accessToken}")
            .build()
    }

    private fun refresh(refreshToken: String): Tokens? = try {
        val response = refreshApi.refresh(RefreshRequest(refreshToken)).execute()
        response.body()?.takeIf { response.isSuccessful }?.let { Tokens(it.accessToken, it.refreshToken) }
    } catch (e: IOException) {
        null
    }

    private fun responseCount(response: Response): Int {
        var count = 1
        var prior = response.priorResponse
        while (prior != null) {
            count++
            prior = prior.priorResponse
        }
        return count
    }
}
