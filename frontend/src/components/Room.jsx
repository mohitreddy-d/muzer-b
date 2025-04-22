import React, { useState, useEffect } from 'react';

export default function Room({ token, room }) {
  const [queue, setQueue] = useState([]);

  useEffect(() => {
    loadQueue();
    const protocol = window.location.protocol === 'https:' ? 'wss' : 'ws';
    const ws = new WebSocket(
      `${protocol}://${window.location.host}/api/v1/ws/${room.id}?token=${token}`
    );
    ws.onmessage = () => {
      loadQueue();
    };
    return () => {
      ws.close();
    };
  }, []);

  const loadQueue = async () => {
    const res = await fetch(`/api/v1/rooms/${room.id}/queue`, {
      headers: { Authorization: 'Bearer ' + token },
    });
    if (res.ok) {
      const data = await res.json();
      setQueue(data);
    }
  };

  const addSong = async () => {
    const trackID = prompt('Track ID:');
    const trackName = prompt('Track name:');
    const artist = prompt('Artist:');
    if (!trackID || !trackName || !artist) return;
    await fetch(`/api/v1/rooms/${room.id}/queue`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: 'Bearer ' + token,
      },
      body: JSON.stringify({ track_id: trackID, track_name: trackName, artist }),
    });
    loadQueue();
  };

  const vote = async (trackID, value) => {
    await fetch(`/api/v1/rooms/${room.id}/vote`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: 'Bearer ' + token,
      },
      body: JSON.stringify({ track_id: trackID, vote: value }),
    });
    loadQueue();
  };

  const nextSong = async () => {
    const res = await fetch(`/api/v1/rooms/${room.id}/next`, {
      headers: { Authorization: 'Bearer ' + token },
    });
    if (res.ok) {
      const song = await res.json();
      alert(`Next: ${song.track_name} by ${song.artist}`);
    } else {
      alert('No next song');
    }
  };

  return (
    <div className="p-4 max-w-lg mx-auto space-y-4">
      <div className="flex justify-between items-center">
        <h2 className="text-xl font-bold">
          Room: {room.name} (Code: {room.code})
        </h2>
        <button className="btn" onClick={nextSong}>
          Next Song
        </button>
      </div>
      <ul className="menu bg-base-200 rounded-box w-full">
        {queue.map((item) => (
          <li key={item.id} className="flex justify-between items-center">
            <span>
              {item.track_name} by {item.artist}
            </span>
            <div className="flex items-center space-x-2">
              <button className="btn btn-xs" onClick={() => vote(item.track_id, 1)}>
                +
              </button>
              <span>{item.votes}</span>
              <button className="btn btn-xs" onClick={() => vote(item.track_id, -1)}>
                -
              </button>
            </div>
          </li>
        ))}
      </ul>
      <button className="btn btn-primary" onClick={addSong}>
        Add Song
      </button>
    </div>
  );
}