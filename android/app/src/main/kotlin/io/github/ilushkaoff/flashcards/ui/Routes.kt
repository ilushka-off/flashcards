package io.github.ilushkaoff.flashcards.ui

import kotlinx.serialization.Serializable

@Serializable
data object LoginRoute

@Serializable
data object RegisterRoute

@Serializable
data object DecksRoute

@Serializable
data class DeckDetailRoute(val id: Long)

/** id = 0 — создание новой колоды, иначе редактирование существующей. */
@Serializable
data class DeckEditRoute(val id: Long = 0)

@Serializable
data object ProfileRoute
