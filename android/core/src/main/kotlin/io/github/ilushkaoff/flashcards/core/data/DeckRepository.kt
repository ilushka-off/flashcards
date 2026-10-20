package io.github.ilushkaoff.flashcards.core.data

import io.github.ilushkaoff.flashcards.core.api.ApiResult
import io.github.ilushkaoff.flashcards.core.api.FlashcardsApi
import io.github.ilushkaoff.flashcards.core.api.apiCall
import io.github.ilushkaoff.flashcards.core.api.map
import io.github.ilushkaoff.flashcards.core.model.Deck
import io.github.ilushkaoff.flashcards.core.model.DeckCreate
import io.github.ilushkaoff.flashcards.core.model.DeckUpdate
import io.github.ilushkaoff.flashcards.core.model.Language
import kotlinx.serialization.json.Json

class DeckRepository(
    private val api: FlashcardsApi,
    private val json: Json,
) {
    suspend fun list(): ApiResult<List<Deck>> = apiCall(json) { api.listDecks() }.map { it.items }

    suspend fun get(id: Long): ApiResult<Deck> = apiCall(json) { api.getDeck(id) }

    suspend fun create(name: String, language: Language, description: String): ApiResult<Deck> =
        apiCall(json) {
            api.createDeck(DeckCreate(name.trim(), language, description.trim().ifEmpty { null }))
        }

    /** Отправляет только изменившиеся поля. Если ничего не поменялось, запрос не нужен. */
    suspend fun update(old: Deck, name: String, description: String): ApiResult<Deck> {
        val body = DeckUpdate(
            name = name.trim().takeIf { it != old.name },
            description = description.trim().takeIf { it != old.description },
        )
        if (body.name == null && body.description == null) return ApiResult.Success(old)
        return apiCall(json) { api.updateDeck(old.id, body) }
    }

    suspend fun delete(id: Long): ApiResult<Unit> = apiCall(json) { api.deleteDeck(id) }
}
