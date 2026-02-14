export interface AudioInfo {
  video_id: string;
  title: string;
  duration: number;
}

export interface Video {
  video_id: string;
  title: string;
  duration: number;
}

export interface PlaylistSummary {
  id: string;
  title: string;
  video_count: number;
  fetching: boolean;
}

export interface Playlist {
  id: string;
  title: string;
  url: string;
  videos: Video[];
  fetched_at: string;
  fetching: boolean;
}

export interface PlaylistPosition {
  video_id: string;
  timestamp: number;
}

export interface ProgressData {
  video_id: string;
  timestamp: number;
  positions: Record<string, PlaylistPosition>;
}

export type View =
  | { page: "login" }
  | { page: "playlists"; profile: string }
  | { page: "videos"; profile: string; playlistId: string }
  | { page: "player"; profile: string; playlistId: string; videoId: string };
