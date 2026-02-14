import { useState, useEffect, useCallback } from "react";
import type { View } from "./types";
import Login from "./components/Login";
import PlaylistList from "./components/PlaylistList";
import VideoList from "./components/VideoList";
import Player from "./components/Player";
import "./App.css";

const TOKEN_KEY = "yt-audios-token";
const PROFILE_KEY = "yt-audios-profile";

function viewToPath(view: View): string {
  switch (view.page) {
    case "login": return "/login";
    case "playlists": return "/playlists";
    case "videos": return `/playlist/${view.playlistId}`;
    case "player": return `/player/${view.playlistId}/${view.videoId}`;
  }
}

function pathToView(path: string): View | null {
  const parts = path.split("/").filter(Boolean);
  if (parts[0] === "playlist" && parts[1]) {
    return { page: "videos", playlistId: parts[1] };
  }
  if (parts[0] === "player" && parts[1] && parts[2]) {
    return { page: "player", playlistId: parts[1], videoId: parts[2] };
  }
  if (parts[0] === "playlists") {
    return { page: "playlists" };
  }
  return null;
}

function getInitialView(): View {
  const token = localStorage.getItem(TOKEN_KEY);
  if (!token) return { page: "login" };

  const fromUrl = pathToView(window.location.pathname);
  if (fromUrl) return fromUrl;

  return { page: "playlists" };
}

export default function App() {
  const [view, setView] = useState<View>(getInitialView);

  const navigate = useCallback((v: View, replace = false) => {
    const path = viewToPath(v);
    if (replace) {
      window.history.replaceState(v, "", path);
    } else {
      window.history.pushState(v, "", path);
    }
    setView(v);
  }, []);

  // set initial history state
  useEffect(() => {
    window.history.replaceState(view, "", viewToPath(view));
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  // listen for back/forward
  useEffect(() => {
    const onPopState = (e: PopStateEvent) => {
      if (e.state && e.state.page) {
        setView(e.state as View);
      } else {
        // fallback: parse URL
        const fromUrl = pathToView(window.location.pathname);
        const token = localStorage.getItem(TOKEN_KEY);
        setView(fromUrl && token ? fromUrl : token ? { page: "playlists" } : { page: "login" });
      }
    };
    window.addEventListener("popstate", onPopState);
    return () => window.removeEventListener("popstate", onPopState);
  }, []);

  switch (view.page) {
    case "login":
      return (
        <Login
          onLogin={() => navigate({ page: "playlists" }, true)}
        />
      );

    case "playlists":
      return (
        <PlaylistList
          onSelect={(playlistId) =>
            navigate({ page: "videos", playlistId })
          }
          onBack={() => {
            localStorage.removeItem(PROFILE_KEY);
            localStorage.removeItem(TOKEN_KEY);
            navigate({ page: "login" }, true);
          }}
        />
      );

    case "videos":
      return (
        <VideoList
          playlistId={view.playlistId}
          onSelect={(videoId) =>
            navigate({
              page: "player",
              playlistId: view.playlistId,
              videoId,
            })
          }
          onBack={() => window.history.back()}
        />
      );

    case "player":
      return (
        <Player
          playlistId={view.playlistId}
          videoId={view.videoId}
          onBack={() => window.history.back()}
          onChangeVideo={(videoId) =>
            navigate({
              page: "player",
              playlistId: view.playlistId,
              videoId,
            }, true)
          }
        />
      );
  }
}
