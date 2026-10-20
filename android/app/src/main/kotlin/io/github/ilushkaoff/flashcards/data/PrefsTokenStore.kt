package io.github.ilushkaoff.flashcards.data

import android.content.Context
import io.github.ilushkaoff.flashcards.core.auth.TokenStore
import io.github.ilushkaoff.flashcards.core.auth.Tokens

/** Токены в приватных SharedPreferences приложения: другие приложения их не видят. */
class PrefsTokenStore(context: Context) : TokenStore {
    private val prefs = context.getSharedPreferences("auth", Context.MODE_PRIVATE)

    override fun read(): Tokens? {
        val access = prefs.getString(KEY_ACCESS, null) ?: return null
        val refresh = prefs.getString(KEY_REFRESH, null) ?: return null
        return Tokens(access, refresh)
    }

    override fun write(tokens: Tokens?) {
        prefs.edit().apply {
            if (tokens == null) {
                remove(KEY_ACCESS)
                remove(KEY_REFRESH)
            } else {
                putString(KEY_ACCESS, tokens.accessToken)
                putString(KEY_REFRESH, tokens.refreshToken)
            }
        }.commit()
    }

    private companion object {
        const val KEY_ACCESS = "access_token"
        const val KEY_REFRESH = "refresh_token"
    }
}
