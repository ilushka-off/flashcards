package io.github.ilushkaoff.flashcards.ui.common

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import io.github.ilushkaoff.flashcards.core.api.ApiError
import java.time.OffsetDateTime
import java.time.ZoneId
import java.time.format.DateTimeFormatter
import java.util.Locale

/** Сервер пишет сообщения с маленькой буквы, на экране начинаем с заглавной. */
fun ApiError.userMessage(): String = message.replaceFirstChar { it.titlecase(Locale.getDefault()) }

private val dateFormatter = DateTimeFormatter.ofPattern("d MMMM yyyy", Locale("ru"))

/** RFC 3339 из API в локальную дату: «3 октября 2026». */
fun formatDate(rfc3339: String): String = runCatching {
    OffsetDateTime.parse(rfc3339).atZoneSameInstant(ZoneId.systemDefault()).format(dateFormatter)
}.getOrDefault(rfc3339)

@Composable
fun LoadingBox(modifier: Modifier = Modifier) {
    Box(modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
        CircularProgressIndicator()
    }
}

@Composable
fun ErrorBox(message: String, onRetry: () -> Unit, modifier: Modifier = Modifier) {
    Column(
        modifier = modifier
            .fillMaxSize()
            .padding(24.dp),
        verticalArrangement = Arrangement.spacedBy(16.dp, Alignment.CenterVertically),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Text(message, style = MaterialTheme.typography.bodyLarge, textAlign = TextAlign.Center)
        Button(onClick = onRetry) { Text("Повторить") }
    }
}
