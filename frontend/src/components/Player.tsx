import { useState, useRef, useEffect, useCallback } from "react";
import type { Video } from "../types";
import {
  getPlaylistVideos,
  getProgress,
  saveProgress,
  saveProgressBeacon,
} from "../api";

interface Props {
  playlistId: string;
  videoId: string;
  onBack: () => void;
  onChangeVideo: (videoId: string) => void;
}

function fmt(s: number): string {
  if (!isFinite(s)) return "0:00";
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

export default function Player({
  playlistId,
  videoId,
  onBack,
  onChangeVideo,
}: Props) {
  const SPEEDS = [1, 1.25, 1.5, 1.75] as const;
  const audioRef = useRef<HTMLAudioElement>(null);
  const [speed, setSpeed] = useState(() => {
    const saved = localStorage.getItem("yt-audios-speed");
    return saved ? parseFloat(saved) : 1;
  });
  const [title, setTitle] = useState("");
  const [status, setStatus] = useState("");
  const [loading, setLoading] = useState(false);
  const [playing, setPlaying] = useState(false);
  const [currentTime, setCurrentTime] = useState(0);
  const [duration, setDuration] = useState(0);
  const trueDurationRef = useRef(0);
  const [playlist, setPlaylist] = useState<Video[]>([]);
  const saveTimerRef = useRef<number | null>(null);
  const videoIdRef = useRef(videoId);

  const getDuration = useCallback(() => {
    return trueDurationRef.current || audioRef.current?.duration || 0;
  }, []);

  // load playlist videos for next/prev
  useEffect(() => {
    getPlaylistVideos(playlistId).then((pl) => {
      setPlaylist(pl.videos || []);
    });
  }, [playlistId]);

  // load audio when videoId changes
  useEffect(() => {
    videoIdRef.current = videoId;
    loadAudio(videoId);
  }, [videoId]);

  async function loadAudio(vid: string) {
    setStatus("Loading...");
    setLoading(true);
    setTitle("");
    try {
      const resp = await fetch(
        "/audio?url=" + encodeURIComponent("https://www.youtube.com/watch?v=" + vid)
      );
      const data = await resp.json();
      if (data.error) throw new Error(data.error);
      setTitle(data.title);
      trueDurationRef.current = data.duration || 0;
      if (data.duration) setDuration(data.duration);

      const audio = audioRef.current!;
      setStatus("Downloading...");
      audio.src = "/stream?id=" + encodeURIComponent(data.video_id);
      audio.load();

      // restore saved position
      try {
        const progress = await getProgress();
        const pos = progress.positions?.[playlistId];
        if (pos && pos.video_id === vid && pos.timestamp > 0) {
          audio.currentTime = pos.timestamp;
        }
      } catch {
        // no progress saved yet
      }

      audio.playbackRate = speed;
      audio.play().catch(() => {});
      setStatus("");
    } catch (e) {
      setStatus("Error: " + (e as Error).message);
      // auto-skip to next track on error after a short delay
      const idx = playlist.findIndex((v) => v.video_id === vid);
      if (idx >= 0 && idx < playlist.length - 1) {
        setTimeout(() => onChangeVideo(playlist[idx + 1].video_id), 2000);
        setStatus("Skipping unavailable track...");
      }
    } finally {
      setLoading(false);
    }
  }

  // save progress periodically
  useEffect(() => {
    saveTimerRef.current = window.setInterval(() => {
      const audio = audioRef.current;
      if (audio && !audio.paused && audio.currentTime > 0) {
        saveProgress(playlistId, videoIdRef.current, audio.currentTime);
      }
    }, 10000);

    const handleBeforeUnload = () => {
      const audio = audioRef.current;
      if (audio && audio.currentTime > 0) {
        saveProgressBeacon(playlistId, videoIdRef.current, audio.currentTime);
      }
    };
    window.addEventListener("beforeunload", handleBeforeUnload);

    return () => {
      if (saveTimerRef.current) clearInterval(saveTimerRef.current);
      window.removeEventListener("beforeunload", handleBeforeUnload);
      // save on unmount
      const audio = audioRef.current;
      if (audio && audio.currentTime > 0) {
        saveProgress(playlistId, videoIdRef.current, audio.currentTime);
      }
    };
  }, [playlistId]);

  function cycleSpeed() {
    const idx = SPEEDS.indexOf(speed as typeof SPEEDS[number]);
    const next = SPEEDS[(idx + 1) % SPEEDS.length];
    setSpeed(next);
    localStorage.setItem("yt-audios-speed", String(next));
    if (audioRef.current) {
      audioRef.current.playbackRate = next;
    }
  }

  function togglePlay() {
    const audio = audioRef.current;
    if (!audio?.src) return;
    if (audio.paused) {
      audio.play();
    } else {
      audio.pause();
      // save on pause
      saveProgress(playlistId, videoIdRef.current, audio.currentTime);
    }
  }

  function skip(seconds: number) {
    const audio = audioRef.current;
    if (!audio?.src) return;
    const d = getDuration();
    audio.currentTime = Math.max(0, Math.min(d, audio.currentTime + seconds));
  }

  function seek(e: React.MouseEvent<HTMLDivElement>) {
    const d = getDuration();
    if (!d) return;
    const rect = e.currentTarget.getBoundingClientRect();
    audioRef.current!.currentTime =
      ((e.clientX - rect.left) / rect.width) * d;
  }

  function getCurrentIndex(): number {
    return playlist.findIndex((v) => v.video_id === videoId);
  }

  function goNext() {
    const idx = getCurrentIndex();
    if (idx >= 0 && idx < playlist.length - 1) {
      onChangeVideo(playlist[idx + 1].video_id);
    }
  }

  function goPrev() {
    const idx = getCurrentIndex();
    if (idx > 0) {
      onChangeVideo(playlist[idx - 1].video_id);
    }
  }

  useEffect(() => {
    const audio = audioRef.current;
    if (!audio) return;

    const onTimeUpdate = () => {
      setCurrentTime(audio.currentTime);
    };
    const onEnded = () => {
      const idx = playlist.findIndex(
        (v) => v.video_id === videoIdRef.current
      );
      if (idx >= 0 && idx < playlist.length - 1) {
        onChangeVideo(playlist[idx + 1].video_id);
      } else if (playlist.length > 0) {
        onChangeVideo(playlist[0].video_id);
      }
    };
    const onMeta = () => {
      if (!trueDurationRef.current) setDuration(audio.duration);
    };
    const onPlay = () => setPlaying(true);
    const onPause = () => setPlaying(false);
    const onError = () => setStatus("Playback error");

    audio.addEventListener("timeupdate", onTimeUpdate);
    audio.addEventListener("ended", onEnded);
    audio.addEventListener("loadedmetadata", onMeta);
    audio.addEventListener("durationchange", onMeta);
    audio.addEventListener("play", onPlay);
    audio.addEventListener("pause", onPause);
    audio.addEventListener("error", onError);

    return () => {
      audio.removeEventListener("timeupdate", onTimeUpdate);
      audio.removeEventListener("ended", onEnded);
      audio.removeEventListener("loadedmetadata", onMeta);
      audio.removeEventListener("durationchange", onMeta);
      audio.removeEventListener("play", onPlay);
      audio.removeEventListener("pause", onPause);
      audio.removeEventListener("error", onError);
    };
  }, [getDuration, playlist, onChangeVideo]);

  const progress = duration
    ? Math.min((currentTime / duration) * 100, 100)
    : 0;

  const currentIdx = getCurrentIndex();
  const hasPrev = currentIdx > 0;
  const hasNext = currentIdx >= 0 && currentIdx < playlist.length - 1;

  return (
    <div className="view-container">
      <div className="header-row">
        <button className="btn-back" onClick={onBack}>
          &larr;
        </button>
        <h1>Now Playing</h1>
      </div>
      <div className="player">
        <div className="title">
          <span className={`title-text${(title || "").length <= 35 ? " short" : ""}`}>
            {title || (loading ? "" : "Select a video")}
          </span>
        </div>
        <div className="progress-bar" onClick={seek}>
          <div className="progress-fill" style={{ width: `${progress}%` }} />
        </div>
        <div className="time-row">
          <span>{fmt(currentTime)}</span>
          <span>{fmt(duration)}</span>
        </div>
        <div className="controls">
          <button
            className="btn-skip"
            onClick={goPrev}
            disabled={!hasPrev}
            title="Previous"
          >
            &#9198;
          </button>
          <button className="btn-skip" onClick={() => skip(-15)}>
            -15
          </button>
          <button className="btn-play" onClick={togglePlay}>
            {playing ? "\u23F8" : "\u25B6"}
          </button>
          <button className="btn-skip" onClick={() => skip(15)}>
            +15
          </button>
          <button
            className="btn-skip"
            onClick={goNext}
            disabled={!hasNext}
            title="Next"
          >
            &#9197;
          </button>
        </div>
        <div className="speed-row">
          <button className="btn-speed" onClick={cycleSpeed}>
            {speed}x
          </button>
        </div>
        <div className="status">{status}</div>
        {playlist.length > 0 && (
          <div className="track-info">
            Track {currentIdx + 1} of {playlist.length}
          </div>
        )}
        <audio ref={audioRef} preload="auto" />
      </div>
    </div>
  );
}
