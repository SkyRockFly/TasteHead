let toastTimer = null;

function showToast(msg, type = "info", timeout = 3500) {
  const el = document.getElementById("toast");
  if (!el) return;

  if (toastTimer) {
    clearTimeout(toastTimer);
    toastTimer = null;
  }

  el.textContent = msg || "";
  el.className = "toast open";

  if (type === "error") {
    el.classList.add("error");
    el.setAttribute("role", "alert");
    el.setAttribute("aria-live", "assertive");
  } else {
    el.classList.add(type);
    el.setAttribute("role", "status");
    el.setAttribute("aria-live", "polite");
  }

  if (timeout > 0) {
    toastTimer = setTimeout(() => {
      el.classList.remove("open");
    }, timeout);
  }
}

function mustGet(id) {
  const el = document.getElementById(id);
  if (!el) throw new Error(`Missing element #${id}`);
  return el;
}

function readInt(el, def = 0) {
  const n = Number(el.value);
  return Number.isFinite(n) ? Math.trunc(n) : def;
}

function readScoreOrAll(el) {
  const v = String(el.value).trim();
  if (v === "" || v === "all") return null;
  const n = Number(v);
  if (Number.isFinite(n)) return n;
  return null;
}

function readIdOrNull(el) {
  const v = String(el.value).trim();
  if (v === "" || v === "all") return null;
  const n = Number(v);
  return Number.isFinite(n) ? Math.trunc(n) : null;
}

function findInRoot(rootID, selector) {
  const root = document.getElementById(rootID);
  if (!root) {
    throw new Error(`no root ${root}`);
  }
  const item = root.querySelector(selector);
  if (!item) {
    throw new Error(`no selector ${selector}`);
  }

  return item;
}

function normalisePath(p) {
  if (!p) return "";
  return p.replace(/\\\\/g, "/").replace(/\\/g, "/");
}

function fmtScore(v) {
  if (v === null || v === undefined || v === "") return "—";
  const n = Number(v);
  if (Number.isNaN(n)) return String(v);
  return n.toFixed(2);
}

function fmtDateTime(value) {
  if (!value) return "";

  const d = new Date(value);

  if (Number.isNaN(d.getTime())) {
    return String(value);
  }

  return new Intl.DateTimeFormat("ru-RU", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  }).format(d);
}

export const core = {
  mustGet,
  readInt,
  readScoreOrAll,
  readIdOrNull,
  findInRoot,
  normalisePath,
  showToast,
  fmtScore,
  fmtDateTime,
};
