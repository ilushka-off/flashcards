package io.github.ilushkaoff.flashcards.core.api

import io.github.ilushkaoff.flashcards.core.auth.AuthInterceptor
import io.github.ilushkaoff.flashcards.core.auth.Session
import io.github.ilushkaoff.flashcards.core.auth.TokenAuthenticator
import kotlinx.serialization.json.Json
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.logging.HttpLoggingInterceptor
import retrofit2.Retrofit
import retrofit2.converter.kotlinx.serialization.asConverterFactory
import java.util.concurrent.TimeUnit

/** Собирает OkHttp и Retrofit с авторизацией. baseUrl вида http://10.0.2.2:8080/api/v1/ */
class ApiClient(
    baseUrl: String,
    session: Session,
    logHttp: Boolean = false,
) {
    val json = Json {
        ignoreUnknownKeys = true
        explicitNulls = false
    }

    private val converter = json.asConverterFactory("application/json".toMediaType())

    private val baseHttp = OkHttpClient.Builder()
        .connectTimeout(10, TimeUnit.SECONDS)
        .readTimeout(15, TimeUnit.SECONDS)
        .apply {
            if (logHttp) {
                addInterceptor(HttpLoggingInterceptor().setLevel(HttpLoggingInterceptor.Level.BODY))
            }
        }
        .build()

    // refresh идёт отдельным клиентом без authenticator, чтобы 401 на refresh не зациклился
    private val refreshApi: RefreshApi = Retrofit.Builder()
        .baseUrl(baseUrl)
        .client(baseHttp)
        .addConverterFactory(converter)
        .build()
        .create(RefreshApi::class.java)

    val api: FlashcardsApi = Retrofit.Builder()
        .baseUrl(baseUrl)
        .client(
            baseHttp.newBuilder()
                .addInterceptor(AuthInterceptor(session))
                .authenticator(TokenAuthenticator(session, refreshApi))
                .build(),
        )
        .addConverterFactory(converter)
        .build()
        .create(FlashcardsApi::class.java)
}
