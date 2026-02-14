import type { PlaylistSummary, Playlist, ProgressData, AudioInfo } from "./types";

export async function login(
  username: string,
  password: string
): Promise<{ token: string; profile: string }> {
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

export async function listProfiles(): Promise<string[]> {
  const res = await fetch("/api/profiles");
  return res.json();
}

export async function createProfile(name: string): Promise<void> {
  await fetch("/api/profiles", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ name }),
  });
}

export async function deleteProfile(name: string): Promise<void> {
  await fetch("/api/profiles?name=" + encodeURIComponent(name), {
    method: "DELETE",
  });
}

export async function listPlaylists(): Promise<PlaylistSummary[]> {
  const res = await fetch("/api/playlists");
  return res.json();
}

export async function addPlaylist(url: string): Promise<{ id: string }> {
  const res = await fetch("/api/playlists", {
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
  await fetch("/api/playlists?id=" + encodeURIComponent(id), {
    method: "DELETE",
  });
}

export async function getPlaylistVideos(id: string): Promise<Playlist> {
  const res = await fetch("/api/playlists/videos?id=" + encodeURIComponent(id));
  return res.json();
}

export async function refreshPlaylist(id: string): Promise<void> {
  await fetch("/api/playlists/refresh?id=" + encodeURIComponent(id), {
    method: "POST",
  });
}

export async function getProgress(profile: string): Promise<ProgressData> {
  const res = await fetch(
    "/api/progress?profile=" + encodeURIComponent(profile)
  );
  return res.json();
}

export async function saveProgress(
  profile: string,
  playlistId: string,
  videoId: string,
  timestamp: number
): Promise<void> {
  await fetch("/api/progress", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      profile,
      playlist_id: playlistId,
      video_id: videoId,
      timestamp,
    }),
  });
}

export async function loadAudio(url: string): Promise<AudioInfo> {
  const res = await fetch("/audio?url=" + encodeURIComponent(url));
  const data = await res.json();
  if (data.error) throw new Error(data.error);
  return data;
}
