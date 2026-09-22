void main() {
  AppStorys.initialize(token: const String.fromEnvironment('APPSTORYS_API_TOKEN'));
  runApp(MyApp());
}
