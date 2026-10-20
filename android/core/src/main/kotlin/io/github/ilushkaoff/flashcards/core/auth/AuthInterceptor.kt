package io.github.ilushkaoff.flashcards.core.auth

import io.github.ilushkaoff.flashcards.core.api.FlashcardsApi
import okhttp3.Interceptor
import okhttp3.Response

/** Подставляет access-токен во все запросы, кроме помеченных заголовком No-Auth. */
class AuthInterceptor(private val session: Session) : Interceptor {
    override fun intercept(chain: Interceptor.Chain): Response {
        val request = chain.request()
        if (request.header(FlashcardsApi.NO_AUTH) != null) {
            return chain.proceed(request.newBuilder().removeHeader(FlashcardsApi.NO_AUTH).build())
        }
        val access = session.tokens()?.accessToken ?: return chain.proceed(request)
        return chain.proceed(request.newBuilder().header("Authorization", "Bearer $access").build())
    }
}
