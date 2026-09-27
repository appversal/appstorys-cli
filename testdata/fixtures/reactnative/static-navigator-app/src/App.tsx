import React, { useEffect } from 'react';
import AppStorys from '@appstorys/appstorys-react-native';
import { createStaticNavigation } from '@react-navigation/native';
import { createNativeStackNavigator } from '@react-navigation/native-stack';
import HomeScreen from './screens/HomeScreen';
import SettingsScreen from './screens/SettingsScreen';

const RootStack = createNativeStackNavigator({
  screens: {
    Home: HomeScreen,
    Settings: SettingsScreen,
  },
});

const Navigation = createStaticNavigation(RootStack);

export default function App() {
  useEffect(() => {
    AppStorys.initialize('token123');
  }, []);

  return <Navigation />;
}
