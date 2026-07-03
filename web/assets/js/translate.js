import {
  severity,
  severityColor,
  ASPECT_RATIOS,
  debounce,
} from "./common.js";
import { createMinimap } from "./minimap.js";
import { loadContext } from "./session.js";

const ctx = await loadContext();
const { blocks, translations, cfg } = ctx;

let currentIndex = 0; // block shown in the monitors

// --- header -----------------------------------------------------------------
document.getElementById("file-name").textContent = ctx.title || "";
document.getElementById("metric-note").textContent =
  cfg.metric === "wps"
    ? `max ${cfg.wpsMax} words/s`
    : `max ${cfg.cpsMax} chars/s`;

setupProjectHeader();

// In project mode: language dropdown, Configure link, and query-preserving tabs.
function setupProjectHeader() {
  const textviewLink = document.getElementById("textview-link");
  if (ctx.mode !== "project") {
    textviewLink.href = "/textview.html";
    return;
  }
  const q = `?project=${encodeURIComponent(ctx.slug)}&lang=${encodeURIComponent(ctx.lang)}`;
  textviewLink.href = "/textview.html" + q;
  document.getElementById("config-link").href =
    `/config.html?project=${encodeURIComponent(ctx.slug)}`;

  const sel = document.getElementById("lang-select");
  const existing = new Set(ctx.existingLangs || []);
  const inProgress = ctx.languages.filter((l) => existing.has(l.code));
  const others = ctx.languages.filter((l) => !existing.has(l.code));
  const optgroup = (label, items) => {
    if (!items.length) return "";
    return `<optgroup label="${label}">` +
      items.map((l) => `<option value="${l.code}"${l.code === ctx.lang ? " selected" : ""}>${l.name} (${l.code})</option>`).join("") +
      `</optgroup>`;
  };
  sel.innerHTML = optgroup("In progress", inProgress) + optgroup("Start new", others);
  sel.addEventListener("change", () => ctx.switchLang(sel.value));

  document.getElementById("project-controls").hidden = false;
}

const saveStatus = document.getElementById("save-status");
function showSaved(state) {
  if (ctx.mode !== "project") return;
  saveStatus.textContent = state;
  if (state === "Saved ✓") setTimeout(() => (saveStatus.textContent = ""), 1500);
}

// --- monitors ---------------------------------------------------------------
// A single dropdown drives the aspect ratio of both preview monitors.
const monitorEls = [...document.querySelectorAll(".monitor")];
const ratioSelect = document.getElementById("ratio-select");
const defaultRatio = "16:9 (HD/UHD)";

const ratioGroups = {};
ASPECT_RATIOS.forEach((r) => {
  (ratioGroups[r.group] ||= []).push(r);
});
for (const [group, items] of Object.entries(ratioGroups)) {
  const og = document.createElement("optgroup");
  og.label = group;
  items.forEach((r) => {
    const opt = document.createElement("option");
    opt.value = String(r.value);
    opt.textContent = r.label;
    if (r.label === defaultRatio) opt.selected = true;
    og.appendChild(opt);
  });
  ratioSelect.appendChild(og);
}
ratioSelect.addEventListener("change", () => applyRatio(parseFloat(ratioSelect.value)));
applyRatio(parseFloat(ratioSelect.value));

function applyRatio(ratio) {
  monitorEls.forEach((mon) => {
    mon.querySelector(".screen").style.aspectRatio = String(ratio);
  });
}

// Caption text size — scales the caption font in both monitors.
const monitorsEl = document.getElementById("monitors");
const textSizeSelect = document.getElementById("textsize-select");
function applyTextSize(scale) {
  monitorsEl.style.setProperty("--caption-scale", String(scale));
}
textSizeSelect.addEventListener("change", () => applyTextSize(textSizeSelect.value));
applyTextSize(textSizeSelect.value);

function renderMonitors() {
  // Left monitor = original (reference); right monitor = translation.
  const ref = blocks[currentIndex]?.text || "";
  const tr = translations[currentIndex] || "";
  monitorEls[0].querySelector(".caption").textContent = ref;
  monitorEls[1].querySelector(".caption").textContent = tr;
}

// --- rows -------------------------------------------------------------------
const rowsEl = document.getElementById("rows");
const scrollEl = document.getElementById("rows-scroll");
const rowEls = [];

blocks.forEach((blk, i) => {
  const row = document.createElement("div");
  row.className = "row";
  row.id = `block-${i}`;

  const time = document.createElement("div");
  time.className = "cell time";
  time.innerHTML = `<span class="tc">${blk.start}</span><span class="tc">${blk.end}</span>` +
    `<span class="dur">${blk.duration.toFixed(1)}s</span>`;

  const ref = document.createElement("div");
  ref.className = "cell ref";
  ref.textContent = blk.text;

  const trCell = document.createElement("div");
  trCell.className = "cell tr";
  const ta = document.createElement("textarea");
  ta.rows = Math.max(2, blk.lines.length);
  ta.value = translations[i] || "";
  ta.placeholder = "translation…";
  ta.dataset.index = i;
  trCell.appendChild(ta);

  row.append(time, ref, trCell);
  rowsEl.appendChild(row);
  rowEls.push({ row, time, ta });

  ta.addEventListener("focus", () => {
    currentIndex = i;
    renderMonitors();
  });
  ta.addEventListener("input", () => {
    translations[i] = ta.value;
    updateSeverity(i);
    if (i === currentIndex) renderMonitors();
    ctx.onEdit(); // quick mode: debounced localStorage; project mode: mark dirty
    scheduleMinimap();
  });
  // Auto-save on loss of focus (writes the translation file in project mode).
  ta.addEventListener("blur", async () => {
    showSaved("Saving…");
    try {
      await ctx.onCommit();
      showSaved("Saved ✓");
    } catch (err) {
      showSaved("Save failed");
      console.error(err);
    }
  });

  updateSeverity(i);
});

function updateSeverity(i) {
  const text = translations[i] || "";
  const sev = severity(text, blocks[i].duration, cfg);
  rowEls[i].time.style.backgroundColor = severityColor(sev);
  rowEls[i].row.classList.toggle("warn", sev > 0);
}

// --- minimap (whole-file overview + viewport box) ---------------------------
const minimap = createMinimap({
  canvas: document.getElementById("minimap-canvas"),
  viewport: document.getElementById("minimap-viewport"),
  scrollEl,
  items: rowEls.map((r) => r.row),
  severityOf: (i) => severity(translations[i] || "", blocks[i].duration, cfg),
});
const scheduleMinimap = debounce(() => minimap.redraw(), 120);

// --- export -----------------------------------------------------------------
document.getElementById("download-btn").addEventListener("click", async () => {
  const format = document.getElementById("export-format").value;
  const res = await fetch("/api/export", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ format, source: ctx.source, translations }),
  });
  if (!res.ok) {
    alert("Export failed: " + (await res.text()));
    return;
  }
  const blob = await res.blob();
  const ext = format === "plain" ? "txt" : format;
  const base =
    ctx.mode === "project"
      ? `${ctx.slug}.${ctx.lang}`
      : (ctx.title || "translation").replace(/\.[^.]+$/, "");
  const a = document.createElement("a");
  a.href = URL.createObjectURL(blob);
  a.download = `${base}.${ext}`;
  a.click();
  URL.revokeObjectURL(a.href);
});

// --- jump target from text view (#block-N) ----------------------------------
function focusFromHash() {
  const m = location.hash.match(/^#block-(\d+)$/);
  if (!m) return;
  const i = parseInt(m[1], 10);
  const entry = rowEls[i];
  if (entry) {
    entry.row.scrollIntoView({ block: "center" });
    entry.ta.focus();
  }
}
window.addEventListener("hashchange", focusFromHash);

renderMonitors();
focusFromHash();
// Re-measure once layout/fonts have settled so block heights are accurate.
requestAnimationFrame(() => minimap.refresh());
