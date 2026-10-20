package io.github.ilushkaoff.flashcards.core

import io.github.ilushkaoff.flashcards.core.validation.validateDeckDescription
import io.github.ilushkaoff.flashcards.core.validation.validateDeckName
import io.github.ilushkaoff.flashcards.core.validation.validateEmail
import io.github.ilushkaoff.flashcards.core.validation.validatePassword
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Test

class ValidationTest {
    @Test
    fun email() {
        assertNull(validateEmail(" ilya@example.com "))
        assertNotNull(validateEmail(""))
        assertNotNull(validateEmail("ilya@"))
        assertNotNull(validateEmail("ilya example.com"))
    }

    @Test
    fun `password length is counted in bytes`() {
        assertNull(validatePassword("12345678"))
        assertNotNull(validatePassword("1234567"))
        assertNull(validatePassword("я".repeat(36)))  // 72 байта
        assertNotNull(validatePassword("я".repeat(37))) // 74 байта, хотя символов 37
    }

    @Test
    fun `deck name is trimmed and limited`() {
        assertNotNull(validateDeckName("   "))
        assertNull(validateDeckName("a".repeat(100)))
        assertNotNull(validateDeckName("a".repeat(101)))
    }

    @Test
    fun `deck description is optional`() {
        assertNull(validateDeckDescription(""))
        assertNotNull(validateDeckDescription("a".repeat(501)))
    }
}
