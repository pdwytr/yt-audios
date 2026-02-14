import { useState, useEffect } from "react";
import type { View } from "./types";
import Login from "./components/Login";
import PlaylistList from "./components/PlaylistList";
import VideoList from "./components/VideoList";
import Player from "./components/Player";
import "./App.css";

const PROFILE_KEY = "yt-audios-profile";

export default function App() {
  const [view, setView] = useState<View>(() => {
    const saved = localStorage.getItem(PROFILE_KEY);
    if (saved) {
      return { page: "playlists", profile: saved };
    }
    return { page: "login" };
  });

  useEffect(() => {
    if (view.page !== "login" && "profile" in view) {
      localStorage.setItem(PROFILE_KEY, view.profile);
    }
  }, [view]);

  switch (view.page) {
    case "login":
      return (
        <Login
          onLogin={(profile) => setView({ page: "playlists", profile })}
        />
      );

    case "playlists":
      return (
        <PlaylistList
          profile={view.profile}
          onSelect={(playlistId) =>
            setView({ page: "videos", profile: view.profile, playlistId })
          }
          onBack={() => {
            localStorage.removeItem(PROFILE_KEY);
            localStorage.removeItem("yt-audios-token");
            setView({ page: "login" });
          }}
        />
      );

    case "videos":
      return (
        <VideoList
          profile={view.profile}
          playlistId={view.playlistId}
          onSelect={(videoId) =>
            setView({
              page: "player",
              profile: view.profile,
              playlistId: view.playlistId,
              videoId,
            })
          }
          onBack={() => setView({ page: "playlists", profile: view.profile })}
        />
      );

    case "player":
      return (
        <Player
          profile={view.profile}
          playlistId={view.playlistId}
          videoId={view.videoId}
          onBack={() =>
            setView({
              page: "videos",
              profile: view.profile,
              playlistId: view.playlistId,
            })
          }
          onChangeVideo={(videoId) =>
            setView({
              page: "player",
              profile: view.profile,
              playlistId: view.playlistId,
              videoId,
            })
          }
        />
      );
  }
}
