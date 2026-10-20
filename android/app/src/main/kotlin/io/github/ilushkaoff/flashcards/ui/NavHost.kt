package io.github.ilushkaoff.flashcards.ui

import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewmodel.compose.viewModel
import androidx.lifecycle.viewmodel.initializer
import androidx.lifecycle.viewmodel.viewModelFactory
import androidx.navigation.NavController
import androidx.navigation.NavDestination.Companion.hasRoute
import androidx.navigation.compose.NavHost
import androidx.navigation.compose.composable
import androidx.navigation.compose.rememberNavController
import androidx.navigation.toRoute
import io.github.ilushkaoff.flashcards.data.AppContainer
import io.github.ilushkaoff.flashcards.ui.auth.AuthMode
import io.github.ilushkaoff.flashcards.ui.auth.AuthScreen
import io.github.ilushkaoff.flashcards.ui.auth.AuthViewModel
import io.github.ilushkaoff.flashcards.ui.decks.DeckDetailScreen
import io.github.ilushkaoff.flashcards.ui.decks.DeckDetailViewModel
import io.github.ilushkaoff.flashcards.ui.decks.DeckEditScreen
import io.github.ilushkaoff.flashcards.ui.decks.DeckEditViewModel
import io.github.ilushkaoff.flashcards.ui.decks.DecksScreen
import io.github.ilushkaoff.flashcards.ui.decks.DecksViewModel
import io.github.ilushkaoff.flashcards.ui.profile.ProfileScreen
import io.github.ilushkaoff.flashcards.ui.profile.ProfileViewModel

@Composable
fun FlashcardsNavHost(container: AppContainer) {
    val navController = rememberNavController()
    val loggedIn by container.session.loggedIn.collectAsStateWithLifecycle()
    val start: Any = remember { if (container.session.loggedIn.value) DecksRoute else LoginRoute }

    // Вход и выход переключают весь стек: после входа назад на логин не вернуться,
    // а когда сессия кончилась (logout или refresh не удался), открывается экран входа.
    LaunchedEffect(loggedIn) {
        val onAuthScreen = navController.isOnAuthScreen() ?: return@LaunchedEffect
        if (loggedIn && onAuthScreen) navController.resetTo(DecksRoute)
        if (!loggedIn && !onAuthScreen) navController.resetTo(LoginRoute)
    }

    NavHost(navController = navController, startDestination = start) {
        composable<LoginRoute> {
            val vm: AuthViewModel = viewModel(factory = viewModelFactory {
                initializer { AuthViewModel(AuthMode.LOGIN, container.authRepository) }
            })
            AuthScreen(vm, onSwitchMode = { navController.navigate(RegisterRoute) })
        }
        composable<RegisterRoute> {
            val vm: AuthViewModel = viewModel(factory = viewModelFactory {
                initializer { AuthViewModel(AuthMode.REGISTER, container.authRepository) }
            })
            AuthScreen(vm, onSwitchMode = { navController.popBackStack() })
        }
        composable<DecksRoute> {
            val vm: DecksViewModel = viewModel(factory = viewModelFactory {
                initializer { DecksViewModel(container.deckRepository) }
            })
            DecksScreen(
                vm = vm,
                onOpenDeck = { navController.navigate(DeckDetailRoute(it)) },
                onCreateDeck = { navController.navigate(DeckEditRoute()) },
                onOpenProfile = { navController.navigate(ProfileRoute) },
            )
        }
        composable<DeckDetailRoute> { entry ->
            val id = entry.toRoute<DeckDetailRoute>().id
            val vm: DeckDetailViewModel = viewModel(factory = viewModelFactory {
                initializer { DeckDetailViewModel(id, container.deckRepository) }
            })
            DeckDetailScreen(
                vm = vm,
                onBack = { navController.popBackStack() },
                onEdit = { navController.navigate(DeckEditRoute(id)) },
                onDeleted = { navController.popBackStack(DecksRoute, inclusive = false) },
            )
        }
        composable<DeckEditRoute> { entry ->
            val id = entry.toRoute<DeckEditRoute>().id
            val vm: DeckEditViewModel = viewModel(factory = viewModelFactory {
                initializer { DeckEditViewModel(id, container.deckRepository) }
            })
            DeckEditScreen(
                vm = vm,
                onBack = { navController.popBackStack() },
                onSaved = { deckId ->
                    if (id == 0L) {
                        // Новая колода: вместо формы открываем её карточку
                        navController.navigate(DeckDetailRoute(deckId)) {
                            popUpTo<DecksRoute>()
                        }
                    } else {
                        navController.popBackStack()
                    }
                },
            )
        }
        composable<ProfileRoute> {
            val vm: ProfileViewModel = viewModel(factory = viewModelFactory {
                initializer { ProfileViewModel(container.authRepository) }
            })
            ProfileScreen(vm, onBack = { navController.popBackStack() })
        }
    }
}

/** null, пока граф ещё не показал первый экран. */
private fun NavController.isOnAuthScreen(): Boolean? {
    val destination = currentBackStackEntry?.destination ?: return null
    return destination.hasRoute<LoginRoute>() || destination.hasRoute<RegisterRoute>()
}

private fun NavController.resetTo(route: Any) {
    navigate(route) {
        popUpTo(graph.id) { inclusive = true }
        launchSingleTop = true
    }
}
