class HomeScreen extends StatefulWidget {
  @override
  State<HomeScreen> createState() => _HomeScreenState();
}

class _HomeScreenState extends State<HomeScreen> {
  @override
  void initState() {
    super.initState();
    AppStorys.trackScreen('Home Screen', context);
  }

  @override
  Widget build(BuildContext context) {
    return Stack(
      children: [
        Text('Home'),
        ...AppStorys.overlayElements(),
      ],
    );
  }
}
