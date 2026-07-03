// A VS-Code-style minimap: the WHOLE document is compressed to fit the panel
// height (never scrolls), a translucent blue box marks the currently visible
// region, and clicking/dragging seeks. Rendered to a <canvas> so a block can be
// one pixel or less — in that sub-pixel case each pixel's colour is the
// coverage-weighted blend of every block that falls within it, so the amount of
// content in a pixel drives its colour.
import { severityColor } from "./common.js";

// createMinimap({ canvas, viewport, scrollEl, items, severityOf })
//   canvas     : <canvas> filling the panel
//   viewport   : <div> overlay used as the blue "where am I" box
//   scrollEl   : the scrolling element (an element, or window for page scroll)
//   items      : one DOM element per block, in order (used to measure layout)
//   severityOf : (index) => 0..1 reading-speed severity for that block
export function createMinimap({ canvas, viewport, scrollEl, items, severityOf }) {
  const ctx = canvas.getContext("2d");
  const host = canvas.parentElement; // the .minimap panel
  const isWindow = scrollEl === window;
  const scroller = isWindow ? document.scrollingElement : scrollEl;

  let offsets = []; // { top, h } in scroller-content pixels
  let contentH = 1; // total scrollable content height (px)

  // Position of an item within the scroller's content coordinate space.
  function contentTop(el) {
    const r = el.getBoundingClientRect();
    if (isWindow) return r.top + window.scrollY;
    const sr = scroller.getBoundingClientRect();
    return r.top - sr.top + scroller.scrollTop;
  }

  function measure() {
    offsets = items.map((el) => ({ top: contentTop(el), h: el.getBoundingClientRect().height }));
    contentH = Math.max(1, scroller.scrollHeight);
  }

  function sizeCanvas() {
    const dpr = window.devicePixelRatio || 1;
    const cssW = host.clientWidth;
    const cssH = host.clientHeight;
    canvas.width = Math.max(1, Math.round(cssW * dpr));
    canvas.height = Math.max(1, Math.round(cssH * dpr));
    canvas.style.width = cssW + "px";
    canvas.style.height = cssH + "px";
  }

  function draw() {
    const H = canvas.height; // device pixels
    const W = canvas.width;
    ctx.clearRect(0, 0, W, H);
    const scale = H / contentH; // content px -> device px

    // Accumulate coverage & coverage-weighted severity per pixel row.
    const sevSum = new Float32Array(H);
    const covSum = new Float32Array(H);
    for (let i = 0; i < offsets.length; i++) {
      const o = offsets[i];
      const top = o.top * scale;
      const bot = (o.top + o.h) * scale;
      const sev = severityOf(i) || 0;
      const y0 = Math.max(0, Math.floor(top));
      const y1 = Math.min(H, Math.ceil(bot));
      for (let y = y0; y < y1; y++) {
        const cover = Math.min(bot, y + 1) - Math.max(top, y); // 0..1 of this pixel
        if (cover <= 0) continue;
        covSum[y] += cover;
        sevSum[y] += sev * cover;
      }
    }

    for (let y = 0; y < H; y++) {
      const cov = covSum[y];
      if (cov <= 0) continue; // gap between blocks -> panel background shows
      const clamped = Math.min(1, cov);
      // faint neutral bar so document structure is always visible…
      ctx.fillStyle = `rgba(150,160,175,${0.16 * clamped})`;
      ctx.fillRect(0, y, W, 1);
      // …then the severity colour on top, weighted by how full the pixel is.
      const sev = sevSum[y] / cov;
      if (sev > 0) {
        ctx.fillStyle = severityColor(sev, clamped);
        ctx.fillRect(0, y, W, 1);
      }
    }
  }

  function updateViewport() {
    const cssH = host.clientHeight;
    const view = isWindow ? window.innerHeight : scroller.clientHeight;
    const top = (scroller.scrollTop / contentH) * cssH;
    const h = Math.max(6, (view / contentH) * cssH);
    viewport.style.top = `${top}px`;
    viewport.style.height = `${h}px`;
  }

  function seekTo(clientY) {
    const rect = host.getBoundingClientRect();
    const frac = (clientY - rect.top) / rect.height;
    const view = isWindow ? window.innerHeight : scroller.clientHeight;
    let target = frac * contentH - view / 2;
    target = Math.max(0, Math.min(contentH - view, target));
    if (isWindow) window.scrollTo({ top: target });
    else scroller.scrollTop = target;
  }

  // --- interaction: click/drag to seek --------------------------------------
  host.addEventListener("mousedown", (e) => {
    e.preventDefault();
    seekTo(e.clientY);
    host.classList.add("seeking");
    const move = (ev) => seekTo(ev.clientY);
    const up = () => {
      host.classList.remove("seeking");
      window.removeEventListener("mousemove", move);
      window.removeEventListener("mouseup", up);
    };
    window.addEventListener("mousemove", move);
    window.addEventListener("mouseup", up);
  });

  (isWindow ? window : scroller).addEventListener("scroll", updateViewport, { passive: true });
  window.addEventListener("resize", () => refresh());

  function redraw() {
    draw();
    updateViewport();
  }
  function refresh() {
    measure();
    sizeCanvas();
    draw();
    updateViewport();
  }

  refresh();
  return { redraw, refresh };
}
