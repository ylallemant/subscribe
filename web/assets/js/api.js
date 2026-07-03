// Thin wrapper around the project + quick-mode HTTP API.

async function req(method, url, body) {
  const opts = { method };
  if (body !== undefined) {
    opts.headers = { "Content-Type": "application/json" };
    opts.body = JSON.stringify(body);
  }
  const res = await fetch(url, opts);
  if (!res.ok) throw new Error((await res.text()) || res.statusText);
  if (res.status === 204) return null;
  const ct = res.headers.get("Content-Type") || "";
  return ct.includes("application/json") ? res.json() : res.text();
}

export const api = {
  capabilities: () => req("GET", "/api/capabilities"),
  languages: () => req("GET", "/api/languages"),

  listProjects: () => req("GET", "/api/projects"),
  createProject: (displayName) => req("POST", "/api/projects", { displayName }),
  getProject: (slug) => req("GET", `/api/projects/${slug}`),
  deleteProject: (slug) => req("DELETE", `/api/projects/${slug}`),
  saveSettings: (slug, settings) => req("PUT", `/api/projects/${slug}/settings`, settings),

  importFile: async (slug, lang, file) => {
    const body = new FormData();
    body.append("file", file, file.name);
    body.append("lang", lang);
    const res = await fetch(`/api/projects/${slug}/files`, { method: "POST", body });
    if (!res.ok) throw new Error(await res.text());
    return res.json();
  },

  original: (slug) => req("GET", `/api/projects/${slug}/original`),
  loadTranslation: (slug, lang) => req("GET", `/api/projects/${slug}/translations/${lang}`),
  saveTranslation: (slug, lang, translations) =>
    req("PUT", `/api/projects/${slug}/translations/${lang}`, { translations }),
};

// Per-project "last language I worked on", remembered in the browser.
export function lastLang(slug) {
  return localStorage.getItem(`st:lastlang:${slug}`) || "";
}
export function setLastLang(slug, lang) {
  localStorage.setItem(`st:lastlang:${slug}`, lang);
}
