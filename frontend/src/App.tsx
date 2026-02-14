import { useState } from "react";
import type { View } from "./types";
import Login from "./components/Login";
import PlaylistList from "./components/PlaylistList";
import VideoList from "./components/VideoList";
import Player from "./components/Player";
import "./App.css";

const TOKEN_KEY = "yt-audios-token";
const PROFILE_KEY = "yt-audios-profile";

export default function App() {
  const [view, setView] = useState<View>(() => {
    const token = localStorage.getItem(TOKEN_KEY);
    if (token) {
      return { page: "playlists" };
    }
    return { page: "login" };
  });

  switch (view.page) {
    case "login":
      return (
        <Login
          onLogin={() => setView({ page: "playlists" })}
        />
      );

    case "playlists":
      return (
        <PlaylistList
          onSelect={(playlistId) =>
            setView({ page: "videos", playlistId })
          }
          onBack={() => {
            localStorage.removeItem(PROFILE_KEY);
            localStorage.removeItem(TOKEN_KEY);
            setView({ page: "login" });
          }}
        />
      );

    case "videos":
      return (
        <VideoList
          playlistId={view.playlistId}
          onSelect={(videoId) =>
            setView({
              page: "player",
              playlistId: view.playlistId,
              videoId,
            })
          }
          onBack={() => setView({ page: "playlists" })}
        />
      );

    case "player":
      return (
        <Player
          playlistId={view.playlistId}
          videoId={view.videoId}
          onBack={() =>
            setView({
              page: "videos",
              playlistId: view.playlistId,
            })
          }
          onChangeVideo={(videoId) =>
            setView({
              page: "player",
              playlistId: view.playlistId,
              videoId,
            })
          }
        />
      );
  }
}
