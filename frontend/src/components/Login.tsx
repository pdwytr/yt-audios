import { useState } from "react";
import { login } from "../api";

interface Props {
  onLogin: () => void;
}

export default function Login({ onLogin }: Props) {
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!username.trim() || !password) return;
    setError("");
    setLoading(true);
    try {
      const result = await login(username.trim(), password);
      localStorage.setItem("yt-audios-token", result.token);
      localStorage.setItem("yt-audios-profile", result.username);
      onLogin();
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="view-container">
      <h1>YT Audio Player</h1>
      <form onSubmit={handleSubmit}>
        <div className="login-fields">
          <input
            type="text"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            placeholder="Username"
            autoFocus
          />
          <input
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            placeholder="Password"
          />
          <button type="submit" disabled={loading || !username.trim() || !password}>
            {loading ? "Logging in..." : "Log In"}
          </button>
        </div>
      </form>
      {error && <div className="status">{error}</div>}
    </div>
  );
}
