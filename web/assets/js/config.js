import { api } from "./api.js";

const slug = new URLSearchParams(location.search).get("project");
if (!slug) window.location.href = "/";

const els = {
  title: document.getElementById("project-title"),
  open: document.getElementById("open-btn"),
  sourceInput: document.getElementById("source-input"),
  uploadLang: document.getElementById("upload-lang"),
  addFileBtn: document.getElementById("add-file-btn"),
  sourceList: document.getElementById("source-list"),
  refLang: document.getElementById("reference-lang"),
  format: document.getElementById("format"),
  fps: document.getElementById("fps"),
  metric: document.getElementById("metric"),
  cpsMax: document.getElementById("cpsMax"),
  wpsMax: document.getElementById("wpsMax"),
  save: document.getElementById("save-btn"),
  status: document.getElementById("status"),
};

const [project, languages] = await Promise.all([api.getProject(slug), api.languages()]);
const langName = Object.fromEntries(languages.map((l) => [l.code, l.name]));

// The "language of this file" picker offers the full curated list.
els.uploadLang.innerHTML = languages
  .map((l) => `<option value="${l.code}">${l.name} (${l.code})</option>`)
  .join("");

function setStatus(msg, isError = false) {
  els.status.textContent = msg || "";
  els.status.classList.toggle("error", isError);
}

function fillForm(p) {
  const s = p.settings;
  els.title.textContent = s.displayName || p.slug;
  els.format.value = s.format || "srt";
  els.fps.value = s.fps || 25;
  els.metric.value = s.metric || "cps";
  els.cpsMax.value = s.cpsMax || 17;
  els.wpsMax.value = s.wpsMax || 3;
  renderFiles(p);
  updateOpen(p);
}

function renderFiles(p) {
  const files = p.languages || [];
  els.sourceList.innerHTML = files.length
    ? files
        .map(
          (code) =>
            `<li><span class="chip">${code}</span> ${escapeHtml(langName[code] || code)}` +
            `<code>${p.slug}.${code}.${p.settings.format}</code></li>`
        )
        .join("")
    : `<li class="muted">No files yet — add the original above.</li>`;

  // The reference-language picker only offers languages that have a file.
  const prev = p.settings.referenceLang || "";
  els.refLang.innerHTML =
    `<option value="">— select —</option>` +
    files
      .map(
        (code) =>
          `<option value="${code}"${code === prev ? " selected" : ""}>${langName[code] || code} (${code})</option>`
      )
      .join("");
}

function updateOpen(p) {
  const ready = Boolean(p.settings.referenceLang) && (p.languages || []).includes(p.settings.referenceLang);
  els.open.href = `/translate.html?project=${encodeURIComponent(slug)}`;
  els.open.classList.toggle("disabled", !ready);
  els.open.title = ready ? "" : "Add a file and set the original language first";
}

function escapeHtml(s) {
  return String(s).replace(/[&<>"']/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c]));
}

fillForm(project);

// Add / import a file for a language.
els.addFileBtn.addEventListener("click", async () => {
  const file = els.sourceInput.files[0];
  const lang = els.uploadLang.value;
  if (!file) {
    setStatus("Choose a file first.", true);
    return;
  }
  setStatus(`Uploading ${file.name} as ${langName[lang] || lang}…`);
  try {
    const p = await api.importFile(slug, lang, file);
    Object.assign(project, p);
    // Default the reference to the first file added.
    if (!project.settings.referenceLang && p.languages.length === 1) {
      project.settings.referenceLang = p.languages[0];
    }
    renderFiles(project);
    updateOpen(project);
    els.sourceInput.value = "";
    setStatus(`Added ${file.name}.`);
  } catch (err) {
    setStatus(err.message, true);
  }
});

// Save settings.
els.save.addEventListener("click", async () => {
  const settings = {
    displayName: project.settings.displayName,
    format: els.format.value,
    fps: parseFloat(els.fps.value) || 25,
    metric: els.metric.value,
    cpsMax: parseFloat(els.cpsMax.value) || 17,
    wpsMax: parseFloat(els.wpsMax.value) || 3,
    referenceLang: els.refLang.value,
  };
  setStatus("Saving…");
  try {
    const p = await api.saveSettings(slug, settings);
    Object.assign(project, p);
    fillForm(project);
    setStatus("Saved.");
  } catch (err) {
    setStatus(err.message, true);
  }
});
