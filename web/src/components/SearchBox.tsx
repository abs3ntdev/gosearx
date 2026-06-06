import { useEffect, useRef, useState } from "react";
import { autocomplete } from "../api";

// SearchBox is the query input with a live autocomplete dropdown (keyboard
// navigable), mirroring SearXNG's autocompleter.
export function SearchBox({
  value,
  onChange,
  onSubmit,
  autocompleteMin,
  autocompleteEnabled,
}: {
  value: string;
  onChange: (v: string) => void;
  onSubmit: () => void;
  autocompleteMin: number;
  autocompleteEnabled: boolean;
}): React.JSX.Element {
  const [suggestions, setSuggestions] = useState<string[]>([]);
  const [open, setOpen] = useState(false);
  const [active, setActive] = useState(-1);
  const acRef = useRef<AbortController | null>(null);
  const boxRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);
  // Tracks whether the user is actively typing (vs. value changing
  // programmatically, e.g. after submit or from a picked suggestion). The
  // dropdown only auto-opens for genuine typing.
  const typingRef = useRef(false);

  useEffect(() => {
    if (!typingRef.current) return; // value changed programmatically -> ignore
    if (!autocompleteEnabled || value.trim().length < autocompleteMin) {
      setSuggestions([]);
      setOpen(false);
      return;
    }
    const t = setTimeout(async () => {
      acRef.current?.abort();
      const ac = new AbortController();
      acRef.current = ac;
      const s = await autocomplete(value, ac.signal);
      // only open if the input is still focused and the user is still typing
      if (typingRef.current && document.activeElement === inputRef.current) {
        setSuggestions(s);
        setOpen(s.length > 0);
        setActive(-1);
      }
    }, 150);
    return () => clearTimeout(t);
  }, [value, autocompleteMin, autocompleteEnabled]);

  // Close dropdown on outside click.
  useEffect(() => {
    function onDoc(e: MouseEvent) {
      if (boxRef.current && !boxRef.current.contains(e.target as Node)) setOpen(false);
    }
    document.addEventListener("mousedown", onDoc);
    return () => document.removeEventListener("mousedown", onDoc);
  }, []);

  // close stops the dropdown and marks subsequent value changes as non-typing.
  function close() {
    typingRef.current = false;
    acRef.current?.abort();
    setOpen(false);
    setSuggestions([]);
    setActive(-1);
  }

  function submit() {
    close();
    inputRef.current?.blur();
    onSubmit();
  }

  function pick(s: string) {
    onChange(s);
    close();
    inputRef.current?.blur();
    // submit on next tick after value propagates
    setTimeout(onSubmit, 0);
  }

  function onKeyDown(e: React.KeyboardEvent) {
    if (!open || suggestions.length === 0) {
      if (e.key === "Enter") submit();
      return;
    }
    if (e.key === "ArrowDown") {
      e.preventDefault();
      setActive((a) => Math.min(a + 1, suggestions.length - 1));
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      setActive((a) => Math.max(a - 1, -1));
    } else if (e.key === "Enter") {
      e.preventDefault();
      if (active >= 0) pick(suggestions[active]);
      else submit();
    } else if (e.key === "Escape") {
      setOpen(false);
    }
  }

  return (
    <div className="searchbox" ref={boxRef}>
      <input
        ref={inputRef}
        className="search-input"
        value={value}
        onChange={(e) => {
          typingRef.current = true; // genuine user typing -> allow dropdown
          onChange(e.target.value);
        }}
        onFocus={() => {
          // reopen only if we already have suggestions for current typing
          if (typingRef.current && suggestions.length > 0) setOpen(true);
        }}
        onKeyDown={onKeyDown}
        placeholder="Search the web…"
        autoFocus
        autoComplete="off"
        spellCheck={false}
      />
      <button className="search-button" onClick={submit} aria-label="Search">
        Search
      </button>
      {open && suggestions.length > 0 && (
        <ul className="autocomplete">
          {suggestions.map((s, i) => (
            <li
              key={s}
              className={i === active ? "ac-item active" : "ac-item"}
              onMouseEnter={() => setActive(i)}
              onMouseDown={(e) => {
                e.preventDefault();
                pick(s);
              }}
            >
              {s}
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
