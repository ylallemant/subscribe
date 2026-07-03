import { startSession } from "./common.js";
import { api } from "./api.js";

// ===========================================================================
// Projects
// ===========================================================================
const projectList = document.getElementById("project-list");
const projectError = document.getElementById("project-error");
const newProjectForm = document.getElementById("new-project-form");
const projectNameInput = document.getElementById("project-name");

let allowDelete = true; // server capability; the button is hidden when false

function projectErr(msg) {
  projectError.textContent = msg || "";
}

async function refreshProjects() {
  try {
    const [caps, projects] = await Promise.all([
      api.capabilities().catch(() => ({ allowDelete: true })),
      api.listProjects(),
    ]);
    allowDelete = caps.allowDelete !== false;
    renderProjects(projects);
  } catch (err) {
    projectList.innerHTML = "";
    projectErr("Could not load projects: " + err.message);
  }
}

function renderProjects(projects) {
  projectList.innerHTML = "";
  if (!projects.length) {
    projectList.innerHTML = `<li class="muted">No projects yet — create one above.</li>`;
    return;
  }
  for (const p of projects) {
    const li = document.createElement("li");
    li.className = "project-card";
    const href = `/translate.html?project=${encodeURIComponent(p.slug)}`;
    const langs = p.languages && p.languages.length
      ? p.languages.map((l) => `<span class="chip">${l}</span>`).join("")
      : `<span class="muted">no translations yet</span>`;
    const ref = p.hasReference
      ? ""
      : `<span class="chip warnchip">no reference</span>`;
    li.innerHTML = `
      <div class="project-main">
        <a class="project-title" href="${href}">${escapeHtml(p.displayName)}</a>
        <div class="project-meta"><code>${p.slug}</code></div>
        <div class="chips">${ref}${langs}</div>
      </div>
      <div class="project-actions">
        <a class="btn ghost" href="/config.html?project=${encodeURIComponent(p.slug)}">Configure</a>
        ${allowDelete ? `<button class="btn ghost danger" data-del="${p.slug}">Delete</button>` : ""}
      </div>`;
    // The whole tile opens the translation; the action buttons opt out.
    li.addEventListener("click", (e) => {
      if (e.target.closest(".project-actions") || e.target.closest("a")) return;
      window.location.href = href;
    });
    projectList.appendChild(li);
  }
  projectList.querySelectorAll("[data-del]").forEach((b) =>
    b.addEventListener("click", () => deleteProject(b.dataset.del))
  );
}

async function deleteProject(slug) {
  if (!confirm(`Delete project “${slug}” and all its translations? This cannot be undone.`)) return;
  try {
    await api.deleteProject(slug);
    refreshProjects();
  } catch (err) {
    projectErr(err.message);
  }
}

newProjectForm.addEventListener("submit", async (e) => {
  e.preventDefault();
  const name = projectNameInput.value.trim();
  if (!name) return;
  projectErr("");
  try {
    const p = await api.createProject(name);
    // New project has no reference yet → go straight to its config page.
    window.location.href = `/config.html?project=${encodeURIComponent(p.slug)}`;
  } catch (err) {
    projectErr(err.message);
  }
});

function escapeHtml(s) {
  return s.replace(/[&<>"']/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c]));
}

refreshProjects();

// ===========================================================================
// Quick mode (unchanged behaviour)
// ===========================================================================
const form = document.getElementById("upload-form");
const openBtn = document.getElementById("open-btn");
const status = document.getElementById("status");

// Two upload slots: the required reference and an optional existing translation.
const slots = {
  ref: { file: null, source: null },
  tr: { file: null, source: null },
};

function setStatus(msg, isError = false) {
  status.textContent = msg;
  status.classList.toggle("error", isError);
}

function refreshButton() {
  openBtn.disabled = !slots.ref.file;
  openBtn.textContent = slots.tr.file
    ? "Review translation →"
    : "Open workbench →";
}

async function selectFile(slot, file) {
  if (!file) return;
  slots[slot].file = file;
  slots[slot].source = await file.text();
  const zone = document.getElementById(`dropzone-${slot}`);
  zone.classList.add("filled");
  zone.querySelector('[data-role="hint"]').innerHTML =
    `<strong>${file.name}</strong> — click to replace`;
  refreshButton();
  setStatus("");
}

// Parse a raw subtitle string via the server, returning its blocks.
async function parse(source, name) {
  const body = new FormData();
  body.append("file", new Blob([source], { type: "text/plain" }), name);
  const res = await fetch("/api/parse", { method: "POST", body });
  if (!res.ok) throw new Error(`${name}: ${await res.text()}`);
  return (await res.json()).blocks;
}

// Align a translated file's blocks onto the reference blocks. Match by exact
// time box first, then by start timecode, then fall back to position when the
// two files have the same number of blocks.
function alignTranslations(refBlocks, trBlocks) {
  const byKey = new Map();
  const byStart = new Map();
  trBlocks.forEach((b) => {
    byKey.set(`${b.start}|${b.end}`, b.text);
    byStart.set(b.start, b.text);
  });
  const sameCount = refBlocks.length === trBlocks.length;
  return refBlocks.map((b, i) => {
    if (byKey.has(`${b.start}|${b.end}`)) return byKey.get(`${b.start}|${b.end}`);
    if (byStart.has(b.start)) return byStart.get(b.start);
    return sameCount ? trBlocks[i].text : "";
  });
}

async function open() {
  if (!slots.ref.file) return;
  openBtn.disabled = true;
  try {
    setStatus(`Parsing “${slots.ref.file.name}”…`);
    const refBlocks = await parse(slots.ref.source, slots.ref.file.name);

    let translations;
    if (slots.tr.file) {
      setStatus(`Matching “${slots.tr.file.name}” to the reference…`);
      const trBlocks = await parse(slots.tr.source, slots.tr.file.name);
      translations = alignTranslations(refBlocks, trBlocks);
      const matched = translations.filter((t) => t.trim() !== "").length;
      setStatus(`Matched ${matched}/${refBlocks.length} blocks. Opening…`);
    } else {
      setStatus(`Loaded ${refBlocks.length} blocks. Opening…`);
    }

    startSession({
      fileName: slots.ref.file.name,
      source: slots.ref.source,
      blocks: refBlocks,
      translations, // undefined => fresh/empty (resume prior work if any)
    });
    window.location.href = "/translate.html";
  } catch (err) {
    setStatus(err.message, true);
    refreshButton();
  }
}

// --- wire up the two dropzones ---------------------------------------------
for (const slot of ["ref", "tr"]) {
  const zone = document.getElementById(`dropzone-${slot}`);
  const input = zone.querySelector("input");
  zone.addEventListener("click", () => input.click());
  input.addEventListener("change", () => selectFile(slot, input.files[0]));

  ["dragenter", "dragover"].forEach((ev) =>
    zone.addEventListener(ev, (e) => {
      e.preventDefault();
      zone.classList.add("dragover");
    })
  );
  ["dragleave", "drop"].forEach((ev) =>
    zone.addEventListener(ev, (e) => {
      e.preventDefault();
      zone.classList.remove("dragover");
    })
  );
  zone.addEventListener("drop", (e) => selectFile(slot, e.dataTransfer.files[0]));
}

form.addEventListener("submit", (e) => {
  e.preventDefault();
  open();
});
