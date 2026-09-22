class HomeScreen extends State<Home> {
  void initState() {
    super.initState();
    AppStorys.trackScreen('Home Screen', context);
  }

  void submit() {
    AppStorys.trackEvents(event: 'Login', metadata: {'method': 'email'});
  }

  Widget build(BuildContext context) {
    return Stack(
      children: [
        ...AppStorys.overlayElements(),
        AppStorys.widgets(position: 'widget_one'),
      ],
    );
  }
}

const tooltipTag = ValueKey<String>('tooltip_one');
