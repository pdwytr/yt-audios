import { useState, useEffect, useRef } from "react";
import type { PlaylistSummary, Channel } from "../types";
import { listPlaylists, addPlaylist, deletePlaylist, listChannels, getProgress, listSuggestions } from "../api";

interface Props {
  onSelect: (playlistId: string) => void;
  onBack: () => void;
}

interface Suggestion {
  id: string;
  url: string;
  title: string;
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

  // Setup view state
  const [suggestions, setSuggestions] = useState<Suggestion[]>([]);
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [addingSelected, setAddingSelected] = useState(false);
  const [skippedSetup, setSkippedSetup] = useState(false);
  const [showSuggestions, setShowSuggestions] = useState(false);

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
    listSuggestions().then(setSuggestions).catch(() => {});
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

  function toggleSuggestion(id: string) {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else if (next.size < 3) {
        next.add(id);
      }
      return next;
    });
  }

  async function handleAddSelected() {
    setError("");
    setAddingSelected(true);
    try {
      for (const id of selected) {
        const s = suggestions.find((s) => s.id === id);
        if (s) await addPlaylist(s.url);
      }
      setSelected(new Set());
      setShowSuggestions(false);
      await load();
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setAddingSelected(false);
    }
  }

  const showSetup = !loading && suggestions.length > 0 &&
    (showSuggestions || (playlists.length === 0 && !skippedSetup));

  if (showSetup) {
    return (
      <div className="view-container">
        <div className="header-row">
          <h1>{localStorage.getItem("yt-audios-profile") || "Playlists"}</h1>
          <button className="btn-logout" onClick={onBack}>
            Logout
          </button>
        </div>
        <div className="setup-label">Select up to 3 playlists</div>
        {error && <div className="status" style={{ marginBottom: 12 }}>{error}</div>}
        <div className="setup-list">
          {suggestions.map((s) => {
            const isSelected = selected.has(s.id);
            const isDisabled = !isSelected && selected.size >= 3;
            return (
              <div
                key={s.id}
                className={`setup-item${isSelected ? " selected" : ""}${isDisabled ? " disabled" : ""}`}
                onClick={() => !isDisabled && !addingSelected && toggleSuggestion(s.id)}
              >
                <span className="setup-check">{isSelected ? "✓" : ""}</span>
                <span className="list-item-title">{s.title}</span>
              </div>
            );
          })}
        </div>
        <button
          className="btn-add-selected"
          disabled={selected.size === 0 || addingSelected}
          onClick={handleAddSelected}
        >
          {addingSelected ? "Adding..." : `Add ${selected.size} playlist${selected.size !== 1 ? "s" : ""}`}
        </button>
        <button
          className="btn-link"
          style={{ marginTop: 12 }}
          onClick={() => { setSkippedSetup(true); setShowSuggestions(false); }}
        >
          Skip — add by URL instead
        </button>
      </div>
    );
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
      {!loading && playlists.length < 3 && suggestions.length > 0 && (
        <div className="suggestions-section">
          <button className="btn-link" onClick={() => setShowSuggestions(true)}>
            Browse suggested playlists
          </button>
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
