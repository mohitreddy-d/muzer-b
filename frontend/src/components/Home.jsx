import React, { useState } from 'react';

export default function Home({ token, onJoin }) {
  const [name, setName] = useState('');
  const [code, setCode] = useState('');

  const create = async () => {
    const res = await fetch('/api/v1/rooms/', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': 'Bearer ' + token,
      },
      body: JSON.stringify({ name }),
    });
    if (res.ok) {
      const room = await res.json();
      onJoin(room);
    }
  };

  const join = async () => {
    const res = await fetch(`/api/v1/rooms/code/${code}`, {
      headers: { Authorization: 'Bearer ' + token },
    });
    if (res.ok) {
      const room = await res.json();
      onJoin(room);
    }
  };

  return (
    <div className="p-4 space-y-4 max-w-md mx-auto">
      <h1 className="text-2xl font-bold">Join or Create a Room</h1>
      <input
        className="input input-bordered w-full"
        placeholder="Room name"
        value={name}
        onChange={(e) => setName(e.target.value)}
      />
      <button className="btn w-full" onClick={create}>
        Create Room
      </button>
      <input
        className="input input-bordered w-full"
        placeholder="Room code"
        value={code}
        onChange={(e) => setCode(e.target.value)}
      />
      <button className="btn w-full" onClick={join}>
        Join Room
      </button>
    </div>
  );
}