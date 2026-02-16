import { useState, useEffect } from "react";
import { login, register, getRegisterInfo, setToken } from "../api";

interface Props {
  onLogin: () => void;
}

export default function Login({ onLogin }: Props) {
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const [isRegister, setIsRegister] = useState(false);
  const [remaining, setRemaining] = useState<number | null>(null);

  useEffect(() => {
    if (isRegister) {
      getRegisterInfo().then((r) => setRemaining(r.remaining)).catch(() => {});
    }
  }, [isRegister]);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!username.trim() || !password) return;
    setError("");
    setLoading(true);
    try {
      const fn = isRegister ? register : login;
      const result = await fn(username.trim(), password);
      setToken(result.token);
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
          <button type="submit" disabled={loading || !username.trim() || !password || (isRegister && remaining === 0)}>
            {loading
              ? (isRegister ? "Registering..." : "Logging in...")
              : (isRegister ? "Register" : "Log In")}
          </button>
        </div>
      </form>
      {error && <div className="status">{error}</div>}
      <div className="auth-toggle">
        {isRegister ? (
          <>Already have an account? <button className="btn-link" onClick={() => { setIsRegister(false); setError(""); }}>Log in</button></>
        ) : (
          <>No account? <button className="btn-link" onClick={() => { setIsRegister(true); setError(""); }}>Register</button></>
        )}
      </div>
      {isRegister && remaining !== null && (
        <div className="auth-note">
          {remaining > 0
            ? `${remaining} registration${remaining === 1 ? "" : "s"} left.`
            : "Registration closed — no spots left."}
        </div>
      )}
    </div>
  );
}
