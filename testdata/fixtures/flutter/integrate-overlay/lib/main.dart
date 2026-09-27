import 'home_screen.dart';

void main() {
  WidgetsFlutterBinding.ensureInitialized();
  AppStorys.initialize(token: const String.fromEnvironment('APPSTORYS_API_TOKEN'));
  runApp(MyApp());
}

class MyApp extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      routes: {
        '/': (context) => HomeScreen(),
      },
    );
  }
}
