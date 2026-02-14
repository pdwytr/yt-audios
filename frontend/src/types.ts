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
  videos: Video[];
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

export interface Channel {
  channel_id: string;
  channel_name: string;
}

export type View =
  | { page: "login" }
  | { page: "playlists" }
  | { page: "videos"; playlistId: string }
  | { page: "player"; playlistId: string; videoId: string };
