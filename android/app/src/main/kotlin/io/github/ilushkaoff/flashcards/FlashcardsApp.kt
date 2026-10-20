package io.github.ilushkaoff.flashcards

import android.app.Application
import io.github.ilushkaoff.flashcards.data.AppContainer

class FlashcardsApp : Application() {
    lateinit var container: AppContainer
        private set

    override fun onCreate() {
        super.onCreate()
        container = AppContainer(this)
    }
}
