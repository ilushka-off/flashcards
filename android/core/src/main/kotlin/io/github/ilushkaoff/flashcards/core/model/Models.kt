package io.github.ilushkaoff.flashcards.core.model

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

// Модели повторяют схемы из api/openapi.yaml, имена полей в snake_case.

@Serializable
data class RegisterRequest(
    val email: String,
    val password: String,
)

@Serializable
data class LoginRequest(
    val email: String,
    val password: String,
)

@Serializable
data class RefreshRequest(
    @SerialName("refresh_token") val refreshToken: String,
)

@Serializable
data class TokenResponse(
    @SerialName("access_token") val accessToken: String,
    @SerialName("refresh_token") val refreshToken: String,
    @SerialName("token_type") val tokenType: String,
    @SerialName("expires_in") val expiresIn: Long,
)

@Serializable
data class User(
    val id: Long,
    val email: String,
    @SerialName("created_at") val createdAt: String,
)

@Serializable
enum class Language(val title: String) {
    @SerialName("en")
    EN("Английский"),

    @SerialName("es")
    ES("Испанский"),
}

@Serializable
data class Deck(
    val id: Long,
    val name: String,
    val language: Language,
    val description: String,
    @SerialName("total_cards") val totalCards: Int,
    @SerialName("due_cards") val dueCards: Int,
    @SerialName("created_at") val createdAt: String,
)

@Serializable
data class DeckList(
    val items: List<Deck>,
)

@Serializable
data class DeckCreate(
    val name: String,
    val language: Language,
    val description: String? = null,
)

// В PATCH уходят только переданные поля: null не сериализуется (explicitNulls = false).
@Serializable
data class DeckUpdate(
    val name: String? = null,
    val description: String? = null,
)

@Serializable
data class ErrorResponse(
    val error: ErrorBody,
)

@Serializable
data class ErrorBody(
    val code: String,
    val message: String,
)
