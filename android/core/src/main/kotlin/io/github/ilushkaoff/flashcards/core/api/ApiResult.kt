package io.github.ilushkaoff.flashcards.core.api

import io.github.ilushkaoff.flashcards.core.model.ErrorResponse
import kotlinx.coroutines.CancellationException
import kotlinx.serialization.json.Json
import retrofit2.Response
import java.io.IOException

sealed interface ApiResult<out T> {
    data class Success<T>(val value: T) : ApiResult<T>
    data class Failure(val error: ApiError) : ApiResult<Nothing>
}

sealed interface ApiError {
    val message: String

    /** Сервер ответил ошибкой в формате {"error": {"code", "message"}}. */
    data class Http(val status: Int, val code: String, override val message: String) : ApiError

    /** Запрос не дошёл до сервера или ответ оборвался. */
    data class Network(override val message: String = "Нет связи с сервером") : ApiError

    /** Ответ не удалось разобрать. */
    data class Unexpected(override val message: String = "Неожиданный ответ сервера") : ApiError
}

inline fun <T, R> ApiResult<T>.map(transform: (T) -> R): ApiResult<R> = when (this) {
    is ApiResult.Success -> ApiResult.Success(transform(value))
    is ApiResult.Failure -> this
}

/** Выполняет запрос и превращает ответ или исключение в ApiResult. */
suspend fun <T> apiCall(json: Json, block: suspend () -> Response<T>): ApiResult<T> {
    val response = try {
        block()
    } catch (e: CancellationException) {
        throw e
    } catch (e: IOException) {
        return ApiResult.Failure(ApiError.Network())
    } catch (e: Exception) {
        return ApiResult.Failure(ApiError.Unexpected())
    }
    return response.toResult(json)
}

@Suppress("UNCHECKED_CAST")
fun <T> Response<T>.toResult(json: Json): ApiResult<T> {
    if (isSuccessful) {
        val body = body()
        return when {
            body != null -> ApiResult.Success(body)
            // 204 No Content: тело пустое, для методов с Response<Unit> это успех
            code() == 204 -> ApiResult.Success(Unit as T)
            else -> ApiResult.Failure(ApiError.Unexpected())
        }
    }
    return ApiResult.Failure(parseError(json, code(), errorBody()?.string()))
}

fun parseError(json: Json, status: Int, raw: String?): ApiError {
    val parsed = raw?.let { runCatching { json.decodeFromString<ErrorResponse>(it) }.getOrNull() }
    return if (parsed != null) {
        ApiError.Http(status, parsed.error.code, parsed.error.message)
    } else {
        ApiError.Http(status, "unknown", "Ошибка сервера ($status)")
    }
}
