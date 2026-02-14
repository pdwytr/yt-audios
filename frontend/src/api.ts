import type { PlaylistSummary, Playlist, ProgressData, AudioInfo, Channel } from "./types";

function getToken(): string | null {
  return localStorage.getItem("yt-audios-token");
}

async function authFetch(url: string, init?: RequestInit): Promise<Response> {
  const token = getToken();
  const headers = new Headers(init?.headers);
  if (token) {
    headers.set("Authorization", `Bearer ${token}`);
  }
  const res = await fetch(url, { ...init, headers });
  if (res.status === 401) {
    localStorage.removeItem("yt-audios-token");
    localStorage.removeItem("yt-audios-profile");
    window.location.reload();
    throw new Error("Unauthorized");
  }
  return res;
}

export async function login(
  username: string,
  password: string
): Promise<{ token: string; username: string }> {
  const res = await fetch("/api/login", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ username, password }),
  });
  if (!res.ok) {
    const data = await res.json();
    throw new Error(data.error || "Login failed");
  }
  return res.json();
}

export async function register(
  username: string,
  password: string
): Promise<{ token: string; username: string }> {
  const res = await fetch("/api/register", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ username, password }),
  });
  if (!res.ok) {
    const data = await res.json();
    throw new Error(data.error || "Registration failed");
  }
  return res.json();
}

export async function getRegisterInfo(): Promise<{ remaining: number }> {
  const res = await fetch("/api/register/info");
  return res.json();
}

export async function listChannels(): Promise<Channel[]> {
  const res = await fetch("/api/channels");
  return res.json();
}

export async function listPlaylists(): Promise<PlaylistSummary[]> {
  const res = await authFetch("/api/playlists");
  return res.json();
}

export async function addPlaylist(url: string): Promise<{ id: string }> {
  const res = await authFetch("/api/playlists", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ url }),
  });
  if (!res.ok) {
    const data = await res.json();
    throw new Error(data.error || "Failed to add playlist");
  }
  return res.json();
}

export async function deletePlaylist(id: string): Promise<void> {
  await authFetch("/api/playlists?id=" + encodeURIComponent(id), {
    method: "DELETE",
  });
}

export async function getPlaylistVideos(id: string): Promise<Playlist> {
  const res = await authFetch("/api/playlists/videos?id=" + encodeURIComponent(id));
  return res.json();
}

export async function refreshPlaylist(id: string): Promise<void> {
  await authFetch("/api/playlists/refresh?id=" + encodeURIComponent(id), {
    method: "POST",
  });
}

export async function getProgress(): Promise<ProgressData> {
  const res = await authFetch("/api/progress");
  return res.json();
}

export async function saveProgress(
  playlistId: string,
  videoId: string,
  timestamp: number
): Promise<void> {
  await authFetch("/api/progress", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      playlist_id: playlistId,
      video_id: videoId,
      timestamp,
    }),
  });
}

export function saveProgressBeacon(
  playlistId: string,
  videoId: string,
  timestamp: number
): void {
  const token = getToken();
  // sendBeacon doesn't support custom headers, so we use fetch with keepalive
  fetch("/api/progress", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
    body: JSON.stringify({ playlist_id: playlistId, video_id: videoId, timestamp }),
    keepalive: true,
  }).catch(() => {});
}

export async function loadAudio(url: string): Promise<AudioInfo> {
  const res = await fetch("/audio?url=" + encodeURIComponent(url));
  const data = await res.json();
  if (data.error) throw new Error(data.error);
  return data;
}
