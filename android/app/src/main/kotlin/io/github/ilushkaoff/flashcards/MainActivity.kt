package io.github.ilushkaoff.flashcards

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import io.github.ilushkaoff.flashcards.ui.FlashcardsNavHost
import io.github.ilushkaoff.flashcards.ui.theme.FlashcardsTheme

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()
        val container = (application as FlashcardsApp).container
        setContent {
            FlashcardsTheme {
                FlashcardsNavHost(container)
            }
        }
    }
}
