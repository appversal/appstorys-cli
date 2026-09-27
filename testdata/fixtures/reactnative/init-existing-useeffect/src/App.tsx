import React, { useEffect } from 'react';
import { View } from 'react-native';

export default function App() {
  useEffect(() => {
    console.log('mounted');
  }, []);

  return <View />;
}
