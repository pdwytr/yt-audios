import type { PlaylistSummary, Playlist, ProgressData, AudioInfo, Channel } from "./types";

const TOKEN_KEY = "yt-audios-token";
const TAB_ID_KEY = "yt-audios-tab-id";

let currentToken: string | null = null;
let tabId: string | null = null;

// On load: claim active tab or detect we were kicked by another tab
const storedToken = localStorage.getItem(TOKEN_KEY);
const activeTabId = localStorage.getItem(TAB_ID_KEY);
const myTabId = sessionStorage.getItem(TAB_ID_KEY);

if (storedToken) {
  if (myTabId && activeTabId && myTabId !== activeTabId) {
    // Another tab took over while we were away — stay logged out
    currentToken = null;
    sessionStorage.removeItem(TAB_ID_KEY);
  } else {
    // Fresh tab or we're the active tab — claim ownership
    currentToken = storedToken;
    tabId = Math.random().toString(36).slice(2);
    sessionStorage.setItem(TAB_ID_KEY, tabId);
    localStorage.setItem(TAB_ID_KEY, tabId);
  }
}

export function getToken(): string | null {
  return currentToken;
}

export function setToken(token: string | null): void {
  currentToken = token;
  if (token) {
    localStorage.setItem(TOKEN_KEY, token);
    tabId = Math.random().toString(36).slice(2);
    sessionStorage.setItem(TAB_ID_KEY, tabId);
    localStorage.setItem(TAB_ID_KEY, tabId);
  } else {
    localStorage.removeItem(TOKEN_KEY);
    localStorage.removeItem(TAB_ID_KEY);
    sessionStorage.removeItem(TAB_ID_KEY);
    tabId = null;
  }
}

// When another tab claims the session, kick this tab
window.addEventListener("storage", (e) => {
  if (e.key === TAB_ID_KEY && tabId && e.newValue !== tabId) {
    currentToken = null;
    tabId = null;
    window.location.reload();
  }
});

async function authFetch(url: string, init?: RequestInit): Promise<Response> {
  const token = getToken();
  const headers = new Headers(init?.headers);
  if (token) {
    headers.set("Authorization", `Bearer ${token}`);
  }
  const res = await fetch(url, { ...init, headers });
  if (res.status === 401) {
    setToken(null);
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
