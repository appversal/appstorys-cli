import React from 'react';
import { View } from 'react-native';
import AppStorys from '@appstorys/appstorys-react-native';

function submit() {
  AppStorys.trackEvent('Login', undefined, { method: 'email' });
}

function clicked() {
  // Reserved event name — should be flagged by lint.
  AppStorys.trackEvent('clicked');
}

export default function HomeScreen() {
  return (
    <View appstorys="tooltip_one">
      <AppStorys.Widgets position="widget_one" />
      <AppStorys.Stories />
    </View>
  );
}
