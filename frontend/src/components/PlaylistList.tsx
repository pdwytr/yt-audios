import { useState, useEffect, useRef } from "react";
import type { PlaylistSummary, Channel } from "../types";
import { listPlaylists, addPlaylist, deletePlaylist, listChannels, getProgress } from "../api";

interface Props {
  onSelect: (playlistId: string) => void;
  onBack: () => void;
}

export default function PlaylistList({ onSelect, onBack }: Props) {
  const [playlists, setPlaylists] = useState<PlaylistSummary[]>([]);
  const [channels, setChannels] = useState<Channel[]>([]);
  const [loading, setLoading] = useState(true);
  const [newUrl, setNewUrl] = useState("");
  const [adding, setAdding] = useState(false);
  const [error, setError] = useState("");
  const [lastPlaylistId, setLastPlaylistId] = useState<string | null>(null);
  const pollRef = useRef<number | null>(null);


  async function load() {
    try {
      const data = await listPlaylists();
      setPlaylists(data);
      if (data.some((p) => p.fetching)) {
        if (!pollRef.current) {
          pollRef.current = window.setInterval(load, 3000);
        }
      } else if (pollRef.current) {
        clearInterval(pollRef.current);
        pollRef.current = null;
      }
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    load();
    listChannels().then(setChannels).catch(() => {});
    getProgress()
      .then((p) => { if (p.playlist_id) setLastPlaylistId(p.playlist_id); })
      .catch(() => {});
    return () => {
      if (pollRef.current) clearInterval(pollRef.current);
    };
  }, []);

  async function handleAdd() {
    const url = newUrl.trim();
    if (!url) return;
    setError("");
    setAdding(true);
    try {
      await addPlaylist(url);
      setNewUrl("");
      await load();
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setAdding(false);
    }
  }

  async function handleDelete(id: string) {
    await deletePlaylist(id);
    await load();
  }

  return (
    <div className="view-container">
      <div className="header-row">
        <h1>{localStorage.getItem("yt-audios-profile") || "Playlists"}</h1>
        <button className="btn-logout" onClick={onBack}>
          Logout
        </button>
      </div>
      {playlists.length < 3 && (
        <div className="input-row" style={{ marginBottom: 16 }}>
          <input
            type="text"
            value={newUrl}
            onChange={(e) => setNewUrl(e.target.value)}
            onKeyDown={(e) => e.key === "Enter" && handleAdd()}
            placeholder="Paste playlist URL..."
            disabled={adding}
          />
          <button onClick={handleAdd} disabled={adding || !newUrl.trim()}>
            {adding ? "Adding..." : "Add"}
          </button>
        </div>
      )}
      {!loading && (
        <div className="playlist-limit">{playlists.length}/3 playlists</div>
      )}
      {error && <div className="status" style={{ marginBottom: 12 }}>{error}</div>}
      {loading ? (
        <div className="status">Loading...</div>
      ) : (
        <div className="list">
          {playlists.map((pl) => (
            <div
              key={pl.id}
              className={`list-item${pl.id === lastPlaylistId ? " active" : ""}`}
              onClick={() => !pl.fetching && onSelect(pl.id)}
            >
              <div className="list-item-content">
                <span className="list-item-title">
                  {pl.title || pl.id}
                </span>
                <span className="list-item-sub">
                  {pl.fetching
                    ? "Fetching..."
                    : `${pl.video_count} videos`}
                </span>
              </div>
              <button
                className="btn-delete"
                onClick={(e) => {
                  e.stopPropagation();
                  handleDelete(pl.id);
                }}
                title="Remove playlist"
              >
                &times;
              </button>
            </div>
          ))}
          {playlists.length === 0 && (
            <div className="empty">No playlists yet. Add one above.</div>
          )}
        </div>
      )}
      {channels.length > 0 && (
        <div className="channels-info">
          <span>Allowed channels: </span>
          {channels.map((ch) => ch.channel_name).join(", ")}
        </div>
      )}
    </div>
  );
}
