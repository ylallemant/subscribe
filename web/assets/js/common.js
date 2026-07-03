// Shared helpers: server config, client-side session store (localStorage),
// reading-speed severity + colour, and the aspect-ratio catalogue.

// ---------------------------------------------------------------------------
// Config (mirrors internal/reading.Config; fetched from the server)
// ---------------------------------------------------------------------------

export const DEFAULT_CONFIG = {
  fps: 25,
  metric: "cps",
  cpsMax: 17,
  wpsMax: 3,
  warnFraction: 0.85,
  fullFraction: 1.25,
};

export async function fetchConfig() {
  try {
    const r = await fetch("/api/config");
    if (r.ok) return await r.json();
  } catch (_) {}
  return DEFAULT_CONFIG;
}

// ---------------------------------------------------------------------------
// Severity + colour — MUST match internal/reading.severity()
// ---------------------------------------------------------------------------

export function countChars(text) {
  return text.replace(/\n/g, "").length;
}
export function countWords(text) {
  const t = text.trim();
  return t === "" ? 0 : t.split(/\s+/).length;
}

// severity(text, durationSeconds, config) -> 0 (fine) .. 1 (full red)
export function severity(text, duration, cfg) {
  if (!duration || duration <= 0) return 0;
  const chars = countChars(text);
  const words = countWords(text);
  const actual = cfg.metric === "wps" ? words / duration : chars / duration;
  const max = cfg.metric === "wps" ? cfg.wpsMax : cfg.cpsMax;
  if (max <= 0 || cfg.fullFraction <= cfg.warnFraction) return 0;
  const lo = cfg.warnFraction * max;
  const hi = cfg.fullFraction * max;
  if (actual <= lo) return 0;
  if (actual >= hi) return 1;
  return (actual - lo) / (hi - lo);
}

// severityColor(sev, alphaScale) -> CSS colour, or "transparent" when fine.
// Ramps amber (hue 45) -> red (hue 0) as severity rises.
export function severityColor(sev, alphaScale = 1) {
  if (sev <= 0) return "transparent";
  const hue = 45 * (1 - sev);
  const alpha = (0.14 + 0.5 * sev) * alphaScale;
  return `hsla(${hue}, 90%, 50%, ${alpha})`;
}

// ---------------------------------------------------------------------------
// Aspect ratios for the preview "monitors"
// ---------------------------------------------------------------------------

export const ASPECT_RATIOS = [
  { group: "Monitor / TV", label: "4:3 (Classic TV)", value: 4 / 3 },
  { group: "Monitor / TV", label: "5:4", value: 5 / 4 },
  { group: "Monitor / TV", label: "16:10", value: 16 / 10 },
  { group: "Monitor / TV", label: "16:9 (HD/UHD)", value: 16 / 9 },
  { group: "Monitor / TV", label: "21:9 (Ultrawide)", value: 21 / 9 },
  { group: "Cinema", label: "1.85:1 (Flat)", value: 1.85 },
  { group: "Cinema", label: "1.90:1 (IMAX Digital)", value: 1.9 },
  { group: "Cinema", label: "2.00:1 (Univisium)", value: 2.0 },
  { group: "Cinema", label: "2.35:1 (CinemaScope)", value: 2.35 },
  { group: "Cinema", label: "2.39:1 (Scope)", value: 2.39 },
];

// ---------------------------------------------------------------------------
// Session store (localStorage). One active session keyed by a content hash so
// reopening the same reference file restores the translation in progress.
// ---------------------------------------------------------------------------

const CURRENT_KEY = "st:current";
const trKey = (hash) => `st:tr:${hash}`;
const blocksKey = (hash) => `st:blocks:${hash}`;

// cyrb53 — small, fast, non-crypto content hash.
export function hashString(str) {
  let h1 = 0xdeadbeef,
    h2 = 0x41c6ce57;
  for (let i = 0; i < str.length; i++) {
    const ch = str.charCodeAt(i);
    h1 = Math.imul(h1 ^ ch, 2654435761);
    h2 = Math.imul(h2 ^ ch, 1597334677);
  }
  h1 = Math.imul(h1 ^ (h1 >>> 16), 2246822507) ^ Math.imul(h2 ^ (h2 >>> 13), 3266489909);
  h2 = Math.imul(h2 ^ (h2 >>> 16), 2246822507) ^ Math.imul(h1 ^ (h1 >>> 13), 3266489909);
  return (4294967296 * (2097151 & h2) + (h1 >>> 0)).toString(16);
}

// startSession stores a freshly parsed file and becomes the active session.
// When `translations` is provided (e.g. reviewing an existing translation), it
// seeds the translation column, overwriting any prior work for this file.
// When omitted, prior work for the same file is kept (resume), or a fresh empty
// column is created.
export function startSession({ fileName, source, blocks, translations }) {
  const hash = hashString(source);
  const meta = { hash, fileName, source };
  localStorage.setItem(CURRENT_KEY, JSON.stringify(meta));
  localStorage.setItem(blocksKey(hash), JSON.stringify(blocks));

  if (Array.isArray(translations)) {
    const seed = blocks.map((_, i) => translations[i] || "");
    localStorage.setItem(trKey(hash), JSON.stringify(seed));
  } else if (!localStorage.getItem(trKey(hash))) {
    localStorage.setItem(trKey(hash), JSON.stringify(new Array(blocks.length).fill("")));
  }
  return hash;
}

// loadSession returns { hash, fileName, source, blocks, translations } or null.
export function loadSession() {
  const raw = localStorage.getItem(CURRENT_KEY);
  if (!raw) return null;
  const meta = JSON.parse(raw);
  const blocks = JSON.parse(localStorage.getItem(blocksKey(meta.hash)) || "[]");
  const translations = JSON.parse(
    localStorage.getItem(trKey(meta.hash)) || "[]"
  );
  while (translations.length < blocks.length) translations.push("");
  return { ...meta, blocks, translations };
}

export function saveTranslations(hash, translations) {
  localStorage.setItem(trKey(hash), JSON.stringify(translations));
}

// Small debounce for input handlers.
export function debounce(fn, ms = 250) {
  let t;
  return (...a) => {
    clearTimeout(t);
    t = setTimeout(() => fn(...a), ms);
  };
}
