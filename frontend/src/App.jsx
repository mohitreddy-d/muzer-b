import React, { useState, useEffect } from 'react';
import Login from './components/Login';
import Home from './components/Home';
import Room from './components/Room';

function App() {
  const [token, setToken] = useState(null);
  const [room, setRoom] = useState(null);

  useEffect(() => {
    const hash = window.location.hash;
    if (hash.startsWith('#token=')) {
      const t = hash.replace('#token=', '');
      localStorage.setItem('token', t);
      setToken(t);
      window.history.replaceState(null, null, window.location.pathname);
    } else {
      const t = localStorage.getItem('token');
      if (t) {
        setToken(t);
      }
    }
  }, []);

  if (!token) {
    return <Login />;
  }

  if (!room) {
    return <Home token={token} onJoin={setRoom} />;
  }

  return <Room token={token} room={room} />;
}

export default App;