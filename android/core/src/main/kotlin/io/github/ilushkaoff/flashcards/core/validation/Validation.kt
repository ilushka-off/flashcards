package io.github.ilushkaoff.flashcards.core.validation

// Те же ограничения, что в api/openapi.yaml. Проверяем на клиенте, чтобы не гонять заведомо
// плохой запрос, но сервер всё равно остаётся источником правды.

const val PASSWORD_MIN_BYTES = 8
const val PASSWORD_MAX_BYTES = 72
const val EMAIL_MAX = 255
const val DECK_NAME_MAX = 100
const val DECK_DESCRIPTION_MAX = 500

private val emailRegex = Regex("^[^@\\s]+@[^@\\s]+\\.[^@\\s]+$")

fun validateEmail(email: String): String? {
    val value = email.trim()
    return when {
        value.isEmpty() -> "Введите email"
        value.length > EMAIL_MAX -> "Email длиннее $EMAIL_MAX символов"
        !emailRegex.matches(value) -> "Некорректный email"
        else -> null
    }
}

/** Длина пароля в байтах UTF-8: bcrypt на сервере обрезает всё длиннее 72 байт. */
fun validatePassword(password: String): String? {
    val bytes = password.toByteArray(Charsets.UTF_8).size
    return when {
        bytes < PASSWORD_MIN_BYTES -> "Пароль короче $PASSWORD_MIN_BYTES символов"
        bytes > PASSWORD_MAX_BYTES -> "Пароль длиннее $PASSWORD_MAX_BYTES байт"
        else -> null
    }
}

fun validateDeckName(name: String): String? {
    val value = name.trim()
    return when {
        value.isEmpty() -> "Введите название"
        value.codePointCount(0, value.length) > DECK_NAME_MAX -> "Название длиннее $DECK_NAME_MAX символов"
        else -> null
    }
}

fun validateDeckDescription(description: String): String? {
    val value = description.trim()
    return if (value.codePointCount(0, value.length) > DECK_DESCRIPTION_MAX) {
        "Описание длиннее $DECK_DESCRIPTION_MAX символов"
    } else {
        null
    }
}
