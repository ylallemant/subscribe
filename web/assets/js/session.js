// loadContext() returns a uniform working context for the translate/text-view
// pages, hiding whether we're in stateless "quick mode" (browser localStorage)
// or "project mode" (server-backed, one file per language).
//
// Shape:
//   { mode, blocks, translations, cfg, title, source,
//     slug, lang, languages, existingLangs,
//     onEdit(), onCommit(), switchLang(lang) }
import { fetchConfig, loadSession, saveTranslations, debounce } from "./common.js";
import { api, lastLang, setLastLang } from "./api.js";

export async function loadContext() {
  const params = new URLSearchParams(location.search);
  const slug = params.get("project");
  return slug ? loadProject(slug, params) : loadQuick();
}

async function loadQuick() {
  const s = loadSession();
  if (!s) {
    window.location.href = "/";
    return new Promise(() => {}); // never resolves; navigation takes over
  }
  const cfg = await fetchConfig();
  const save = debounce(() => saveTranslations(s.hash, s.translations), 250);
  return {
    mode: "quick",
    slug: null,
    lang: null,
    languages: null,
    existingLangs: [],
    blocks: s.blocks,
    translations: s.translations,
    cfg,
    title: s.fileName || "",
    source: s.source,
    onEdit: () => save(),
    onCommit: () => saveTranslations(s.hash, s.translations),
    switchLang: null,
  };
}

async function loadProject(slug, params) {
  const [proj, languages, original] = await Promise.all([
    api.getProject(slug),
    api.languages(),
    api.original(slug),
  ]);

  const existing = proj.languages || [];
  const lang =
    params.get("lang") || lastLang(slug) || proj.settings.referenceLang || existing[0] || "eng";
  const { translations } = await api.loadTranslation(slug, lang);
  setLastLang(slug, lang);

  const blocks = original.blocks;
  while (translations.length < blocks.length) translations.push("");

  let dirty = false;
  const commit = async () => {
    if (!dirty) return;
    dirty = false;
    await api.saveTranslation(slug, lang, translations);
  };

  return {
    mode: "project",
    slug,
    lang,
    languages,
    existingLangs: existing,
    blocks,
    translations,
    cfg: original.config,
    title: proj.settings.displayName || slug,
    source: original.source,
    onEdit: () => {
      dirty = true;
    },
    onCommit: commit,
    switchLang: async (newLang) => {
      await api.saveTranslation(slug, lang, translations); // flush current work
      setLastLang(slug, newLang);
      const url = new URL(location.href);
      url.searchParams.set("lang", newLang);
      window.location.href = url.toString();
    },
  };
}
