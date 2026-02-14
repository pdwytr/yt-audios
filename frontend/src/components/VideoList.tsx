import { useState, useEffect } from "react";
import type { Video, ProgressData } from "../types";
import { getPlaylistVideos, getProgress, refreshPlaylist } from "../api";

interface Props {
  profile: string;
  playlistId: string;
  onSelect: (videoId: string) => void;
  onBack: () => void;
}

function fmtDuration(s: number): string {
  if (!s || !isFinite(s)) return "";
  s = Math.floor(s);
  const h = Math.floor(s / 3600);
  const m = Math.floor((s % 3600) / 60);
  const sec = s % 60;
  const ms =
    m.toString().padStart(h ? 2 : 1, "0") +
    ":" +
    sec.toString().padStart(2, "0");
  return h ? h + ":" + ms : ms;
}

export default function VideoList({
  profile,
  playlistId,
  onSelect,
  onBack,
}: Props) {
  const [videos, setVideos] = useState<Video[]>([]);
  const [title, setTitle] = useState("");
  const [loading, setLoading] = useState(true);
  const [currentVideoId, setCurrentVideoId] = useState<string | null>(null);

  useEffect(() => {
    async function load() {
      setLoading(true);
      try {
        const [pl, progress] = await Promise.all([
          getPlaylistVideos(playlistId),
          getProgress(profile).catch((): ProgressData => ({
            video_id: "",
            timestamp: 0,
            positions: {},
          })),
        ]);
        setVideos(pl.videos || []);
        setTitle(pl.title);
        const pos = progress.positions?.[playlistId];
        if (pos) setCurrentVideoId(pos.video_id);
      } finally {
        setLoading(false);
      }
    }
    load();
  }, [playlistId, profile]);

  async function handleRefresh() {
    setLoading(true);
    await refreshPlaylist(playlistId);
    // poll until fetching is done
    const poll = setInterval(async () => {
      const pl = await getPlaylistVideos(playlistId);
      if (!pl.fetching) {
        clearInterval(poll);
        setVideos(pl.videos || []);
        setTitle(pl.title);
        setLoading(false);
      }
    }, 3000);
  }

  return (
    <div className="view-container">
      <div className="header-row">
        <button className="btn-back" onClick={onBack}>
          &larr;
        </button>
        <h1>{title || playlistId}</h1>
        <button className="btn-refresh" onClick={handleRefresh} disabled={loading}>
          Refresh
        </button>
      </div>
      {loading ? (
        <div className="status">Loading...</div>
      ) : (
        <div className="list video-list">
          {videos.map((v, i) => (
            <div
              key={v.video_id}
              className={`list-item${v.video_id === currentVideoId ? " active" : ""}`}
              onClick={() => onSelect(v.video_id)}
            >
              <span className="list-item-index">{i + 1}</span>
              <div className="list-item-content">
                <span className="list-item-title">{v.title}</span>
                <span className="list-item-sub">
                  {fmtDuration(v.duration)}
                </span>
              </div>
            </div>
          ))}
          {videos.length === 0 && (
            <div className="empty">No videos in this playlist.</div>
          )}
        </div>
      )}
    </div>
  );
}
