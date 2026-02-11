import { useState, useRef, useEffect, useCallback } from "react";
import "./App.css";

interface AudioInfo {
  video_id: string;
  title: string;
  duration: number;
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

export default function App() {
  const audioRef = useRef<HTMLAudioElement>(null);
  const [url, setUrl] = useState("");
  const [title, setTitle] = useState("");
  const [status, setStatus] = useState("");
  const [loading, setLoading] = useState(false);
  const [playing, setPlaying] = useState(false);
  const [currentTime, setCurrentTime] = useState(0);
  const [duration, setDuration] = useState(0);
  const trueDurationRef = useRef(0);

  const getDuration = useCallback(() => {
    return trueDurationRef.current || audioRef.current?.duration || 0;
  }, []);

  async function loadAudio() {
    const ytUrl = url.trim();
    if (!ytUrl) return;
    setStatus("Loading...");
    setLoading(true);
    setTitle("");
    try {
      const resp = await fetch("/audio?url=" + encodeURIComponent(ytUrl));
      const data: AudioInfo = await resp.json();
      if ("error" in data) throw new Error((data as Record<string, string>).error);
      setTitle(data.title);
      trueDurationRef.current = data.duration || 0;
      if (data.duration) setDuration(data.duration);
      const audio = audioRef.current!;
      setStatus("Downloading...");
      audio.src = "/stream?id=" + encodeURIComponent(data.video_id);
      audio.load();
      audio.play().catch(() => {});
      setStatus("");
    } catch (e) {
      setStatus("Error: " + (e as Error).message);
    } finally {
      setLoading(false);
    }
  }

  function togglePlay() {
    const audio = audioRef.current;
    if (!audio?.src) return;
    if (audio.paused) audio.play();
    else audio.pause();
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

  useEffect(() => {
    const audio = audioRef.current;
    if (!audio) return;

    const onTimeUpdate = () => {
      const d = getDuration();
      if (d && audio.currentTime >= d) {
        audio.currentTime = 0;
        audio.play().catch(() => {});
        return;
      }
      setCurrentTime(audio.currentTime);
    };
    const onMeta = () => {
      if (!trueDurationRef.current) setDuration(audio.duration);
    };
    const onPlay = () => setPlaying(true);
    const onPause = () => setPlaying(false);
    const onError = () => setStatus("Playback error");

    audio.addEventListener("timeupdate", onTimeUpdate);
    audio.addEventListener("loadedmetadata", onMeta);
    audio.addEventListener("durationchange", onMeta);
    audio.addEventListener("play", onPlay);
    audio.addEventListener("pause", onPause);
    audio.addEventListener("error", onError);

    return () => {
      audio.removeEventListener("timeupdate", onTimeUpdate);
      audio.removeEventListener("loadedmetadata", onMeta);
      audio.removeEventListener("durationchange", onMeta);
      audio.removeEventListener("play", onPlay);
      audio.removeEventListener("pause", onPause);
      audio.removeEventListener("error", onError);
    };
  }, [getDuration]);

  const progress = duration
    ? Math.min((currentTime / duration) * 100, 100)
    : 0;

  return (
    <div className="player">
      <h1>YT Audio Player</h1>
      <div className="input-row">
        <input
          type="text"
          value={url}
          onChange={(e) => setUrl(e.target.value)}
          onKeyDown={(e) => e.key === "Enter" && loadAudio()}
          placeholder="Paste YouTube URL..."
        />
        <button onClick={loadAudio} disabled={loading}>
          Load
        </button>
      </div>
      <div className="title">{title}</div>
      <div className="progress-bar" onClick={seek}>
        <div className="progress-fill" style={{ width: `${progress}%` }} />
      </div>
      <div className="time-row">
        <span>{fmt(currentTime)}</span>
        <span>{fmt(duration)}</span>
      </div>
      <div className="controls">
        <button className="btn-skip" onClick={() => skip(-15)}>
          -15
        </button>
        <button className="btn-play" onClick={togglePlay}>
          {playing ? "\u23F8" : "\u25B6"}
        </button>
        <button className="btn-skip" onClick={() => skip(15)}>
          +15
        </button>
      </div>
      <div className="status">{status}</div>
      <audio ref={audioRef} preload="auto" />
    </div>
  );
}
