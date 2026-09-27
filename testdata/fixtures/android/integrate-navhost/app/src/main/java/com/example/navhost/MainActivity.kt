package com.example.navhost

class MainActivity {
    fun onCreate() {
        setContent {
            AppNavHost()
        }
    }
}

@Composable
fun AppNavHost() {
    val navController = rememberNavController()
    NavHost(navController = navController, startDestination = "home") {
        composable("home") {
            HomeScreen()
        }
        composable("order_history") {
            OrderHistoryScreen()
        }
    }
}
