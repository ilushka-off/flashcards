package io.github.ilushkaoff.flashcards.data

import android.content.Context
import io.github.ilushkaoff.flashcards.BuildConfig
import io.github.ilushkaoff.flashcards.core.api.ApiClient
import io.github.ilushkaoff.flashcards.core.auth.Session
import io.github.ilushkaoff.flashcards.core.data.AuthRepository
import io.github.ilushkaoff.flashcards.core.data.DeckRepository

/** Ручной DI: все зависимости приложения создаются один раз здесь. */
class AppContainer(context: Context) {
    val session = Session(PrefsTokenStore(context))

    private val client = ApiClient(
        baseUrl = BuildConfig.API_BASE_URL,
        session = session,
        logHttp = BuildConfig.DEBUG,
    )

    val authRepository = AuthRepository(client.api, session, client.json)
    val deckRepository = DeckRepository(client.api, client.json)
}
