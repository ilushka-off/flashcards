package io.github.ilushkaoff.flashcards.core.auth

import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow

data class Tokens(
    val accessToken: String,
    val refreshToken: String,
)

/** Где хранятся токены между запусками. В приложении это SharedPreferences, в тестах память. */
interface TokenStore {
    fun read(): Tokens?
    fun write(tokens: Tokens?)
}

class InMemoryTokenStore(private var tokens: Tokens? = null) : TokenStore {
    override fun read(): Tokens? = tokens
    override fun write(tokens: Tokens?) {
        this.tokens = tokens
    }
}

/**
 * Текущая сессия пользователя. Читается из OkHttp-потоков (interceptor, authenticator)
 * и из UI, поэтому методы синхронизированы.
 */
class Session(private val store: TokenStore) {
    private val lock = Any()
    private val _loggedIn = MutableStateFlow(store.read() != null)

    /** true, пока есть сохранённые токены. UI переходит на экран входа, когда станет false. */
    val loggedIn: StateFlow<Boolean> = _loggedIn.asStateFlow()

    fun tokens(): Tokens? = synchronized(lock) { store.read() }

    fun save(tokens: Tokens) = synchronized(lock) {
        store.write(tokens)
        _loggedIn.value = true
    }

    fun clear() = synchronized(lock) {
        store.write(null)
        _loggedIn.value = false
    }

    internal fun <T> locked(block: () -> T): T = synchronized(lock, block)
}
