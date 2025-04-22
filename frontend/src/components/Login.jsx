import React from 'react';

export default function Login() {
  const handleLogin = async () => {
    const res = await fetch('/api/v1/auth/login');
    const data = await res.json();
    window.location.href = data.url;
  };

  return (
    <div className="flex items-center justify-center h-screen">
      <button className="btn btn-primary" onClick={handleLogin}>
        Login with Spotify
      </button>
    </div>
  );
}