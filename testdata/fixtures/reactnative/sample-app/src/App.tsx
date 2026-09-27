import React, { useEffect } from 'react';
import AppStorys from '@appstorys/appstorys-react-native';
import HomeScreen from './HomeScreen';

function App() {
  useEffect(() => {
    AppStorys.initialize('token123');
  }, []);

  return (
    <AppStorys.Screen name="Home Screen" options={{ positionList: ['widget_one'] }}>
      <HomeScreen />
    </AppStorys.Screen>
  );
}

export default App;
