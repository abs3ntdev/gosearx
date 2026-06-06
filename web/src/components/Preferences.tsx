import { useEffect, useState } from "react";
import {
  getPreferences,
  savePreferences,
  getEngines,
  type Preferences as Prefs,
  type EngineInfo,
} from "../api";

// Preferences page: language, safesearch, autocomplete, new-tab, and per-engine
// enable/disable. Persisted server-side in a cookie (SearXNG-style).
export function Preferences({ onClose }: { onClose: () => void }): React.JSX.Element {
  const [prefs, setPrefs] = useState<Prefs>({});
  const [engines, setEngines] = useState<EngineInfo[]>([]);
  const [saved, setSaved] = useState(false);

  useEffect(() => {
    getPreferences().then(setPrefs);
    getEngines().then(setEngines);
  }, []);

  function update(patch: Partial<Prefs>) {
    setPrefs((p) => ({ ...p, ...patch }));
    setSaved(false);
  }

  function toggleEngine(name: string, enabled: boolean) {
    const disabled = new Set(prefs.disabled_engines ?? []);
    if (enabled) disabled.delete(name);
    else disabled.add(name);
    update({ disabled_engines: [...disabled] });
  }

  async function save() {
    await savePreferences(prefs);
    setSaved(true);
  }

  const disabled = new Set(prefs.disabled_engines ?? []);
  // group engines by first category for display
  const byCat: Record<string, EngineInfo[]> = {};
  for (const e of engines) {
    const c = e.categories?.[0] ?? "other";
    (byCat[c] ||= []).push(e);
  }

  return (
    <div className="prefs-page">
      <div className="prefs-header">
        <h1>Preferences</h1>
        <button className="prefs-close" onClick={onClose}>
          ✕ Close
        </button>
      </div>

      <section className="prefs-section">
        <h2>Search</h2>
        <label className="prefs-row">
          <span>Default language</span>
          <select
            value={prefs.language ?? "all"}
            onChange={(e) => update({ language: e.target.value })}
          >
            <option value="all">Any</option>
            <option value="en">English</option>
            <option value="en-US">English (US)</option>
            <option value="de">Deutsch</option>
            <option value="fr">Français</option>
            <option value="es">Español</option>
            <option value="it">Italiano</option>
            <option value="ja">日本語</option>
            <option value="zh">中文</option>
          </select>
        </label>
        <label className="prefs-row">
          <span>SafeSearch</span>
          <select
            value={prefs.safesearch ?? 0}
            onChange={(e) => update({ safesearch: Number(e.target.value) })}
          >
            <option value={0}>None</option>
            <option value={1}>Moderate</option>
            <option value={2}>Strict</option>
          </select>
        </label>
        <label className="prefs-row">
          <span>Autocomplete</span>
          <input
            type="checkbox"
            checked={prefs.autocomplete !== false}
            onChange={(e) => update({ autocomplete: e.target.checked })}
          />
        </label>
        <label className="prefs-row">
          <span>Open results in new tab</span>
          <input
            type="checkbox"
            checked={!!prefs.results_new_tab}
            onChange={(e) => update({ results_new_tab: e.target.checked })}
          />
        </label>
      </section>

      <section className="prefs-section">
        <h2>Engines</h2>
        {Object.entries(byCat).map(([cat, list]) => (
          <div key={cat} className="prefs-engine-group">
            <h3>{cat}</h3>
            <div className="prefs-engine-grid">
              {list.map((e) => (
                <label key={e.name} className="prefs-engine">
                  <input
                    type="checkbox"
                    checked={!disabled.has(e.name)}
                    onChange={(ev) => toggleEngine(e.name, ev.target.checked)}
                  />
                  <span>{e.name}</span>
                  {e.shortcut && <code className="prefs-shortcut">!{e.shortcut}</code>}
                </label>
              ))}
            </div>
          </div>
        ))}
      </section>

      <div className="prefs-actions">
        <button className="prefs-save" onClick={save}>
          Save preferences
        </button>
        {saved && <span className="prefs-saved">✓ Saved</span>}
      </div>
    </div>
  );
}
