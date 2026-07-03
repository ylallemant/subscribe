// Dark/light theme. Default follows the OS setting; an explicit user choice is
// remembered in localStorage and applied via data-theme on the root element.
// (Each page also applies the stored value inline in <head> to avoid a flash.)
const KEY = "st:theme";

export function currentTheme() {
  const stored = localStorage.getItem(KEY);
  if (stored === "light" || stored === "dark") return stored;
  return window.matchMedia("(prefers-color-scheme: light)").matches ? "light" : "dark";
}

function apply(theme) {
  document.documentElement.setAttribute("data-theme", theme);
}

export function toggleTheme() {
  const next = currentTheme() === "dark" ? "light" : "dark";
  localStorage.setItem(KEY, next);
  apply(next);
  return next;
}

// Wire a button to toggle the theme; keeps its icon/label in sync.
export function initThemeToggle(btn) {
  const render = () => {
    const t = currentTheme();
    btn.textContent = t === "dark" ? "🌙" : "☀️";
    btn.setAttribute("aria-label", `Switch to ${t === "dark" ? "light" : "dark"} mode`);
    btn.title = btn.getAttribute("aria-label");
  };
  render();
  btn.addEventListener("click", () => {
    toggleTheme();
    render();
  });
}
