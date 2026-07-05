import { severity, severityColor } from "./common.js";
import { createMinimap } from "./minimap.js";
import { loadContext } from "./session.js";
import { initThemeToggle } from "./theme.js";

const ctx = await loadContext();
const { blocks, translations, cfg } = ctx;

// Preserve project/lang context in the tab links and jump targets.
const q =
  ctx.mode === "project"
    ? `?project=${encodeURIComponent(ctx.slug)}&lang=${encodeURIComponent(ctx.lang)}`
    : "";
document.getElementById("translate-link").href = "/translate.html" + q;
document.getElementById("textview-link").href = "/textview.html" + q;
document.getElementById("file-name").textContent =
  ctx.mode === "project" && ctx.lang ? `${ctx.title} · ${ctx.lang}` : ctx.title || "";

// Group consecutive blocks into paragraphs, breaking after a block whose
// reference text ends with sentence-ending punctuation.
function groupParagraphs() {
  const groups = [];
  let cur = [];
  blocks.forEach((blk, i) => {
    cur.push(i);
    if (/[.!?…]["'”»)\]]?\s*$/.test(blk.text.trim())) {
      groups.push(cur);
      cur = [];
    }
  });
  if (cur.length) groups.push(cur);
  return groups;
}

const paragraphsEl = document.getElementById("paragraphs");
const refSpans = []; // one element per block, used to measure the minimap layout

// build a clickable, colour-coded fragment for a block in a given column
function fragment(i, isTranslation) {
  const text = isTranslation ? translations[i] || "" : blocks[i].text;
  const span = document.createElement("span");
  span.className = "frag" + (isTranslation && !text ? " empty" : "");
  span.textContent = text || (isTranslation ? "…" : "");
  span.id = isTranslation ? `tv-tr-${i}` : `tv-ref-${i}`;
  const sev = severity(text, blocks[i].duration, cfg);
  span.style.backgroundColor = severityColor(sev, 1.1);
  span.title = `${blocks[i].start} → ${blocks[i].end}  ·  click to edit`;
  span.addEventListener("click", () => {
    // Jump to the translation box on the translate page (keeping context).
    window.location.href = `/translate.html${q}#block-${i}`;
  });
  if (!isTranslation) refSpans[i] = span;
  return span;
}

const groups = groupParagraphs();

groups.forEach((group) => {
  const rowRef = document.createElement("p");
  rowRef.className = "para ref";
  const rowTr = document.createElement("p");
  rowTr.className = "para tr";
  group.forEach((i) => {
    rowRef.appendChild(fragment(i, false));
    rowRef.appendChild(document.createTextNode(" "));
    rowTr.appendChild(fragment(i, true));
    rowTr.appendChild(document.createTextNode(" "));
  });
  const wrap = document.createElement("div");
  wrap.className = "para-row";
  wrap.append(rowRef, rowTr);
  paragraphsEl.appendChild(wrap);
});

// Whole-document minimap with a viewport box. The paragraphs scroll inside
// #tv-scroll (not the page), so the minimap uses that as its scroller — same
// robust behaviour as the Translate page.
const minimap = createMinimap({
  canvas: document.getElementById("minimap-canvas"),
  viewport: document.getElementById("minimap-viewport"),
  scrollEl: document.getElementById("tv-scroll"),
  items: refSpans,
  severityOf: (i) => severity(translations[i] || "", blocks[i].duration, cfg),
});
requestAnimationFrame(() => minimap.refresh());

// Theme (dark/light) toggle — defaults to the system setting.
initThemeToggle(document.getElementById("toggle-theme"));

// Colour toggle — hides the reading-speed colour on the text fragments (the
// overview keeps its colours). Defaults to off; preference is remembered.
const COLORS_KEY = "st:tv-colors";
const toggle = document.getElementById("toggle-color");
let colorsOn = localStorage.getItem(COLORS_KEY) === "on";
function applyColors() {
  paragraphsEl.classList.toggle("colors-off", !colorsOn);
  toggle.textContent = colorsOn ? "Colours: on" : "Colours: off";
  toggle.setAttribute("aria-pressed", String(colorsOn));
}
toggle.addEventListener("click", () => {
  colorsOn = !colorsOn;
  localStorage.setItem(COLORS_KEY, colorsOn ? "on" : "off");
  applyColors();
});
applyColors();
