import React from 'react';
import { View } from 'react-native';
import AppStorys from '@appstorys/appstorys-react-native';

export default function HomeScreen() {
  return (
    <AppStorys.Screen name="Home Screen">
      <View>
        <AppStorys.Widgets position="widget_one" />
      </View>
    </AppStorys.Screen>
  );
}
