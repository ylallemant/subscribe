// A fake "video stream" that plays the translated subtitles on their real
// timing so you can judge whether a reader can keep up. Data is handed over
// from the Translate page via localStorage.

const data = JSON.parse(localStorage.getItem("st:preview") || "null");

const screen = document.getElementById("preview-screen");
const caption = document.getElementById("preview-caption");
const toggle = document.getElementById("preview-toggle");
const timeEl = document.getElementById("preview-time");
const seek = document.getElementById("preview-seek");

if (!data || !data.cues || !data.cues.length) {
  caption.textContent = "No preview data — open the preview from the Translate page.";
} else {
  document.title = `Preview · ${data.title || "SubScribe"}`;
  screen.style.setProperty("--caption-scale", String(data.captionScale || 1));

  // Fit the "stream" to the largest rectangle of the target aspect ratio that
  // fits the window — width- or height-constrained, like a real video player.
  const ratio = data.ratio || 16 / 9;
  const stage = document.querySelector(".preview-stage");
  function fitScreen() {
    const availW = stage.clientWidth;
    const availH = stage.clientHeight;
    let w = availW;
    let h = w / ratio;
    if (h > availH) {
      h = availH;
      w = h * ratio;
    }
    screen.style.width = Math.floor(w) + "px";
    screen.style.height = Math.floor(h) + "px";
  }
  fitScreen();
  window.addEventListener("resize", fitScreen);

  const { cues, duration } = data;
  seek.max = String(duration);

  let t = 0;
  let playing = false;
  let lastTs = null;
  let raf = null;

  const activeText = (time) => {
    for (const c of cues) if (time >= c.start && time < c.end) return c.text;
    return "";
  };

  const fmt = (s) => {
    s = Math.max(0, Math.floor(s));
    return `${Math.floor(s / 60)}:${String(s % 60).padStart(2, "0")}`;
  };

  function render() {
    caption.textContent = activeText(t);
    seek.value = String(t);
    timeEl.textContent = `${fmt(t)} / ${fmt(duration)}`;
  }

  function frame(ts) {
    if (!playing) return;
    if (lastTs == null) lastTs = ts;
    t += (ts - lastTs) / 1000;
    lastTs = ts;
    if (t >= duration) {
      t = duration;
      playing = false;
      toggle.textContent = "▶";
    }
    render();
    if (playing) raf = requestAnimationFrame(frame);
  }

  function play() {
    if (t >= duration) t = 0;
    playing = true;
    lastTs = null;
    toggle.textContent = "⏸";
    raf = requestAnimationFrame(frame);
  }
  function pause() {
    playing = false;
    toggle.textContent = "▶";
  }

  toggle.addEventListener("click", () => (playing ? pause() : play()));
  seek.addEventListener("input", () => {
    t = parseFloat(seek.value);
    lastTs = null;
    render();
  });

  window.addEventListener("keydown", (e) => {
    if (e.code === "Space") {
      e.preventDefault();
      playing ? pause() : play();
    } else if (e.code === "ArrowRight") {
      t = Math.min(duration, t + 5);
      lastTs = null;
      render();
    } else if (e.code === "ArrowLeft") {
      t = Math.max(0, t - 5);
      lastTs = null;
      render();
    }
  });

  render();
}
