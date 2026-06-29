const API_BASE = "";

const SCORE_OPTIONS = ["", "0", "0.25", "0.5", "0.75", "1"];

let cursor = 0;
let limit = 24;
let hasMore = false;

let images = []; // текущая страница
let batches = []; // батчи из /downloads
let groups = []; // группы дубликатов
let tags = []; // теги для training

// hash/id -> number (user_score пока не отправлен на бэк)
const pendingScores = new Map();

// выбранные картинки (для будущих операций типа "отправить в датасет")
let selectedIDs = new Set();

let modalImage = null; // объект image в модалке
let modalIndex = -1; // индекс в массиве images

// ==== Утилиты ====

function buildApiUrl(path) {
  if (!API_BASE) return path;
  return API_BASE.replace(/\/$/, "") + path;
}

async function fetchJson(path, body = null, method = "POST") {
  const options = {
    method,
    headers: { "Content-Type": "application/json" },
  };

  if (
    body !== null &&
    body !== undefined &&
    method !== "GET" &&
    method !== "HEAD"
  ) {
    options.body = JSON.stringify(body);
  }

  const res = await fetch(buildApiUrl(path), options);

  if (!res.ok) {
    const text = await res.text();
    throw new Error(`${method} ${path}: HTTP ${res.status}: ${text}`);
  }

  const text = await res.text();
  if (!text) return null;

  return JSON.parse(text);
}

function getBatchByID(id) {
  return batches.find((b) => b.id === id) || null;
}

function normalisePath(p) {
  if (!p) return "";
  return p.replace(/\\\\/g, "/").replace(/\\/g, "/");
}

function buildThumbUrl(img) {
  const batch = getBatchByID(img.batch_id);
  if (!batch) {
    console.warn("batch not found", img.batch_id);
    return "";
  }
  const rel = normalisePath(batch.rel_path + "/thumbs/" + img.rel_path);
  return buildApiUrl("/images/get/pic?path=" + encodeURIComponent(rel));
}

function buildFullUrl(img) {
  const batch = getBatchByID(img.batch_id);
  if (!batch) {
    console.warn("batch not found", img.batch_id);
    return "";
  }
  const rel = normalisePath(batch.rel_path + "/" + img.rel_path);
  return buildApiUrl("/images/get/pic?path=" + encodeURIComponent(rel));
}

function fmtScore(v) {
  if (v === null || v === undefined || v === "") return "—";
  const n = Number(v);
  if (Number.isNaN(n)) return String(v);
  return n.toFixed(2);
}

function updatePendingCounter() {
  const el = document.getElementById("pending-counter");
  if (!el) return;
  el.textContent = "Изменений: " + pendingScores.size;
}

function updateSelectedCounter() {
  const el = document.getElementById("selected-counter");
  if (!el) return;
  el.textContent = "Выбрано: " + selectedIDs.size;
}

function setStatus(id, msg, isError) {
  const el = document.getElementById(id);
  if (!el) return;
  el.textContent = msg || "";
  el.style.color = isError ? "#fca5a5" : "#e5e7eb";
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

// читает либо число (0/0.25/...), либо спец-строку типа "all"
function readScoreOrAll(el) {
  const v = String(el.value).trim();
  if (v === "" || v === "all") return null; // null = "не фильтровать"
  const n = Number(v);
  if (Number.isFinite(n)) return n;
  return null; // если ввели мусор — лучше считать как "all", а не ломаться
}

function readIdOrNull(el) {
  const v = String(el.value).trim();
  if (v === "" || v === "all") return null;
  const n = Number(v);
  return Number.isFinite(n) ? Math.trunc(n) : null;
}

function findInRoot(root, selector) {
  const el = document.getElementById(root);
  if (!root) {
    throw new Error(`no root ${root}`);
  }
  const item = el.querySelector(selector);
  if (!item) {
    throw new Error(`no selector ${selector}`);
  }

  return item;
}
