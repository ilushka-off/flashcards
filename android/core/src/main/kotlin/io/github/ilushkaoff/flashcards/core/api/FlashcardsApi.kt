package io.github.ilushkaoff.flashcards.core.api

import io.github.ilushkaoff.flashcards.core.model.Deck
import io.github.ilushkaoff.flashcards.core.model.DeckCreate
import io.github.ilushkaoff.flashcards.core.model.DeckList
import io.github.ilushkaoff.flashcards.core.model.DeckUpdate
import io.github.ilushkaoff.flashcards.core.model.LoginRequest
import io.github.ilushkaoff.flashcards.core.model.RefreshRequest
import io.github.ilushkaoff.flashcards.core.model.RegisterRequest
import io.github.ilushkaoff.flashcards.core.model.TokenResponse
import io.github.ilushkaoff.flashcards.core.model.User
import retrofit2.Call
import retrofit2.Response
import retrofit2.http.Body
import retrofit2.http.DELETE
import retrofit2.http.GET
import retrofit2.http.Headers
import retrofit2.http.PATCH
import retrofit2.http.POST
import retrofit2.http.Path

/**
 * Методы Flashcards API v1, которые уже есть на сервере, плюс refresh/logout из контракта.
 * Пути относительные: baseUrl должен заканчиваться на /api/v1/.
 */
interface FlashcardsApi {
    @Headers(NO_AUTH_HEADER)
    @POST("auth/register")
    suspend fun register(@Body body: RegisterRequest): Response<User>

    @Headers(NO_AUTH_HEADER)
    @POST("auth/login")
    suspend fun login(@Body body: LoginRequest): Response<TokenResponse>

    @Headers(NO_AUTH_HEADER)
    @POST("auth/logout")
    suspend fun logout(@Body body: RefreshRequest): Response<Unit>

    @GET("auth/me")
    suspend fun me(): Response<User>

    @GET("decks")
    suspend fun listDecks(): Response<DeckList>

    @GET("decks/{id}")
    suspend fun getDeck(@Path("id") id: Long): Response<Deck>

    @POST("decks")
    suspend fun createDeck(@Body body: DeckCreate): Response<Deck>

    @PATCH("decks/{id}")
    suspend fun updateDeck(@Path("id") id: Long, @Body body: DeckUpdate): Response<Deck>

    @DELETE("decks/{id}")
    suspend fun deleteDeck(@Path("id") id: Long): Response<Unit>

    companion object {
        const val NO_AUTH = "No-Auth"
        const val NO_AUTH_HEADER = "$NO_AUTH: true"
    }
}

/** Отдельный интерфейс для refresh: вызывается синхронно из OkHttp Authenticator. */
interface RefreshApi {
    @POST("auth/refresh")
    fun refresh(@Body body: RefreshRequest): Call<TokenResponse>
}
