package io.github.ilushkaoff.flashcards.core

import io.github.ilushkaoff.flashcards.core.api.ApiClient
import io.github.ilushkaoff.flashcards.core.api.ApiError
import io.github.ilushkaoff.flashcards.core.api.ApiResult
import io.github.ilushkaoff.flashcards.core.auth.InMemoryTokenStore
import io.github.ilushkaoff.flashcards.core.auth.Session
import io.github.ilushkaoff.flashcards.core.auth.Tokens
import io.github.ilushkaoff.flashcards.core.data.AuthRepository
import io.github.ilushkaoff.flashcards.core.data.DeckRepository
import io.github.ilushkaoff.flashcards.core.model.Language
import kotlinx.coroutines.test.runTest
import okhttp3.mockwebserver.MockResponse
import okhttp3.mockwebserver.MockWebServer
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test

class ApiClientTest {
    private lateinit var server: MockWebServer
    private lateinit var session: Session
    private lateinit var auth: AuthRepository
    private lateinit var decks: DeckRepository

    @Before
    fun setUp() {
        server = MockWebServer()
        server.start()
        session = Session(InMemoryTokenStore())
        val client = ApiClient(server.url("/api/v1/").toString(), session)
        auth = AuthRepository(client.api, session, client.json)
        decks = DeckRepository(client.api, client.json)
    }

    @After
    fun tearDown() {
        server.shutdown()
    }

    @Test
    fun `login saves tokens and sends no Authorization header`() = runTest {
        server.enqueue(json(200, TOKENS_1))

        val result = auth.login(" ilya@example.com ", "password123")

        assertTrue(result is ApiResult.Success)
        assertEquals(Tokens("access-1", "refresh-1"), session.tokens())
        assertTrue(session.loggedIn.value)
        val request = server.takeRequest()
        assertEquals("/api/v1/auth/login", request.path)
        assertNull(request.getHeader("Authorization"))
        assertNull(request.getHeader("No-Auth"))
        assertEquals("""{"email":"ilya@example.com","password":"password123"}""", request.body.readUtf8())
    }

    @Test
    fun `login error is parsed from API error format`() = runTest {
        server.enqueue(json(401, """{"error":{"code":"unauthorized","message":"неверный email или пароль"}}"""))

        val result = auth.login("ilya@example.com", "password123")

        assertEquals(
            ApiResult.Failure(ApiError.Http(401, "unauthorized", "неверный email или пароль")),
            result,
        )
        assertFalse(session.loggedIn.value)
    }

    @Test
    fun `authorized request carries bearer token`() = runTest {
        session.save(Tokens("access-1", "refresh-1"))
        server.enqueue(json(200, """{"items":[$DECK]}"""))

        val result = decks.list()

        assertEquals(1, (result as ApiResult.Success).value.size)
        assertEquals(Language.EN, result.value.first().language)
        assertEquals("Bearer access-1", server.takeRequest().getHeader("Authorization"))
    }

    @Test
    fun `401 refreshes tokens and retries the request`() = runTest {
        session.save(Tokens("access-1", "refresh-1"))
        server.enqueue(json(401, UNAUTHORIZED))
        server.enqueue(json(200, TOKENS_2))
        server.enqueue(json(200, """{"items":[]}"""))

        val result = decks.list()

        assertTrue(result is ApiResult.Success)
        assertEquals(Tokens("access-2", "refresh-2"), session.tokens())
        server.takeRequest()
        val refresh = server.takeRequest()
        assertEquals("/api/v1/auth/refresh", refresh.path)
        assertEquals("""{"refresh_token":"refresh-1"}""", refresh.body.readUtf8())
        assertEquals("Bearer access-2", server.takeRequest().getHeader("Authorization"))
    }

    @Test
    fun `failed refresh clears the session`() = runTest {
        session.save(Tokens("access-1", "refresh-1"))
        server.enqueue(json(401, UNAUTHORIZED))
        server.enqueue(json(404, """{"error":{"code":"not_found","message":"не найдено"}}"""))

        val result = decks.list()

        assertEquals(401, ((result as ApiResult.Failure).error as ApiError.Http).status)
        assertNull(session.tokens())
        assertFalse(session.loggedIn.value)
    }

    @Test
    fun `update sends only changed fields`() = runTest {
        session.save(Tokens("access-1", "refresh-1"))
        val old = (run { server.enqueue(json(200, DECK)); decks.get(1) } as ApiResult.Success).value
        server.takeRequest()
        server.enqueue(json(200, DECK))

        decks.update(old, "English для работы", "Новое описание")

        val request = server.takeRequest()
        assertEquals("PATCH", request.method)
        assertEquals("""{"description":"Новое описание"}""", request.body.readUtf8())
    }

    @Test
    fun `update without changes makes no request`() = runTest {
        session.save(Tokens("access-1", "refresh-1"))
        server.enqueue(json(200, DECK))
        val old = (decks.get(1) as ApiResult.Success).value
        server.takeRequest()

        val result = decks.update(old, " English для работы ", "Слова из созвонов")

        assertEquals(ApiResult.Success(old), result)
        assertEquals(1, server.requestCount)
    }

    @Test
    fun `create omits empty description and delete accepts 204`() = runTest {
        session.save(Tokens("access-1", "refresh-1"))
        server.enqueue(json(201, DECK))
        server.enqueue(MockResponse().setResponseCode(204))

        decks.create(" English для работы ", Language.EN, "  ")
        val deleted = decks.delete(1)

        assertEquals("""{"name":"English для работы","language":"en"}""", server.takeRequest().body.readUtf8())
        assertEquals(ApiResult.Success(Unit), deleted)
        assertEquals("DELETE", server.takeRequest().method)
    }

    @Test
    fun `network failure becomes Network error`() = runTest {
        server.shutdown()

        val result = decks.list()

        assertTrue((result as ApiResult.Failure).error is ApiError.Network)
    }

    @Test
    fun `logout revokes refresh token and clears session`() = runTest {
        session.save(Tokens("access-1", "refresh-1"))
        server.enqueue(MockResponse().setResponseCode(204))

        auth.logout()

        val request = server.takeRequest()
        assertEquals("/api/v1/auth/logout", request.path)
        assertNull(request.getHeader("Authorization"))
        assertNull(session.tokens())
    }

    private fun json(code: Int, body: String) = MockResponse()
        .setResponseCode(code)
        .setHeader("Content-Type", "application/json")
        .setBody(body)

    private companion object {
        const val TOKENS_1 = """{"access_token":"access-1","refresh_token":"refresh-1","token_type":"Bearer","expires_in":900}"""
        const val TOKENS_2 = """{"access_token":"access-2","refresh_token":"refresh-2","token_type":"Bearer","expires_in":900}"""
        const val UNAUTHORIZED = """{"error":{"code":"unauthorized","message":"требуется авторизация"}}"""
        const val DECK = """{"id":1,"name":"English для работы","language":"en","description":"Слова из созвонов","total_cards":0,"due_cards":0,"created_at":"2026-10-03T07:30:00Z"}"""
    }
}
