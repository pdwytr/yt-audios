import { useState, useEffect, useRef } from "react";
import type { PlaylistSummary } from "../types";
import { listPlaylists } from "../api";

interface Props {
  profile: string;
  onSelect: (playlistId: string) => void;
  onBack: () => void;
}

export default function PlaylistList({ profile, onSelect, onBack }: Props) {
  const [playlists, setPlaylists] = useState<PlaylistSummary[]>([]);
  const [loading, setLoading] = useState(true);
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
    return () => {
      if (pollRef.current) clearInterval(pollRef.current);
    };
  }, []);

  return (
    <div className="view-container">
      <div className="header-row">
        <button className="btn-back" onClick={onBack} title="Log out">
          &larr;
        </button>
        <h1>Playlists</h1>
      </div>
      {loading ? (
        <div className="status">Loading...</div>
      ) : (
        <div className="list">
          {playlists.map((pl) => (
            <div
              key={pl.id}
              className="list-item"
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
            </div>
          ))}
          {playlists.length === 0 && (
            <div className="empty">No playlists available.</div>
          )}
        </div>
      )}
    </div>
  );
}
