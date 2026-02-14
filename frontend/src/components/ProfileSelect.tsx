import { useState, useEffect } from "react";
import { listProfiles, createProfile, deleteProfile } from "../api";

interface Props {
  onSelect: (profile: string) => void;
}

export default function ProfileSelect({ onSelect }: Props) {
  const [profiles, setProfiles] = useState<string[]>([]);
  const [newName, setNewName] = useState("");
  const [loading, setLoading] = useState(true);

  async function load() {
    setLoading(true);
    try {
      setProfiles(await listProfiles());
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    load();
  }, []);

  async function handleCreate() {
    const name = newName.trim();
    if (!name) return;
    await createProfile(name);
    setNewName("");
    await load();
  }

  async function handleDelete(name: string) {
    await deleteProfile(name);
    await load();
  }

  return (
    <div className="view-container">
      <h1>Select Profile</h1>
      {loading ? (
        <div className="status">Loading...</div>
      ) : (
        <div className="list">
          {profiles.map((name) => (
            <div key={name} className="list-item" onClick={() => onSelect(name)}>
              <span className="list-item-title">{name}</span>
              <button
                className="btn-delete"
                onClick={(e) => {
                  e.stopPropagation();
                  handleDelete(name);
                }}
              >
                &times;
              </button>
            </div>
          ))}
          {profiles.length === 0 && (
            <div className="empty">No profiles yet. Create one below.</div>
          )}
        </div>
      )}
      <div className="input-row">
        <input
          type="text"
          value={newName}
          onChange={(e) => setNewName(e.target.value)}
          onKeyDown={(e) => e.key === "Enter" && handleCreate()}
          placeholder="New profile name..."
        />
        <button onClick={handleCreate} disabled={!newName.trim()}>
          Create
        </button>
      </div>
    </div>
  );
}
