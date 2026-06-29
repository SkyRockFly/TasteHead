const API_BASE = "";

function buildApiUrl(path) {
  if (!API_BASE) return path;
  return API_BASE.replace(/\/$/, "") + path;
}

async function requestJson(path, body = null, method = "POST") {
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

export const api = {
  updateUserScores(payload) {
    return requestJson("/image/update/score", payload, "PUT");
  },

  listImages(payload) {
    return requestJson("/image/list", payload, "POST");
  },

  listDeletedImages(payload) {
    return requestJson("/image/list/deleted", payload, "POST");
  },

  removeImages(payload) {
    return requestJson("/image/remove", payload, "DELETE");
  },

  createBatch(payload) {
    return requestJson("/batch/create", payload, "POST");
  },

  listBatches() {
    return requestJson("/batch/list", null, "GET");
  },

  listDeletedBatches() {
    return requestJson("/batch/list/deleted", null, "GET");
  },

  removeBatch(payload) {
    return requestJson("/batch/remove", payload, "DELETE");
  },

  restoreBatches(payload) {
    return requestJson("/batch/restore", payload, "PUT");
  },

  restoreImages(payload) {
    return requestJson("/image/restore", payload, "PUT");
  },

  updateBatchName(payload) {
    return requestJson("/batch/update/name", payload, "PUT");
  },

  // ==== scrape / import / download ====

  scrapeImages(payload) {
    return requestJson("/scrape", payload, "POST");
  },

  getScrapeState(payload) {
    return requestJson("/scrape/get/state", payload, "POST");
  },

  getImageUrl(path) {
    return buildApiUrl("/images/get/pic?path=" + encodeURIComponent(path));
  },

  importLocalDirs(payload = {}) {
    return requestJson("/import/batches", payload, "POST");
  },

  trainModel(payload) {
    return requestJson("/model/train", payload, "POST");
  },

  getScrapeState(payload) {
    return requestJson("/scrape/get/state", payload, "POST");
  },

  listModels() {
    return requestJson("/model/list", null, "GET");
  },

  setModel(payload) {
    return requestJson("/model/set", payload, "POST");
  },

  listUnfinishedBatches() {
    return requestJson("/batches/list/unfinished", null, "GET");
  },

  listBatchesWithFails() {
    return requestJson("/batches/list/withFails", null, "GET");
  },

  resumeScrape(payload) {
    return requestJson("/scrape/resume", payload, "POST");
  },

  solveDownloadFails(payload) {
    return requestJson("/scrape/fails", payload, "POST");
  },

  moveImages(payload) {
    return requestJson("/image/move", payload, "POST");
  },

  hardDeleteBatch(payload) {
    return requestJson("/batch/hardDelete", payload, "DELETE");
  },

  hardDeleteImages(payload) {
    return requestJson("/image/hardDelete", payload, "DELETE");
  },

  // ==== training ====

  createTrainingRows(payload) {
    return requestJson("/training/create", payload, "POST");
  },

  removeTrainingRow(payload) {
    return requestJson("/training/delete", payload, "DELETE");
  },

  listTrainingRows(payload) {
    return requestJson("/training/list/req", payload, "POST");
  },

  updateTrainingRowTag(payload) {
    return requestJson("/training/update/tag", payload, "PUT");
  },

  createTag(payload) {
    return requestJson("/tag/create", payload, "POST");
  },

  removeTag(payload) {
    return requestJson("/tag/delete", payload, "DELETE");
  },

  updateTagName(payload) {
    return requestJson("/tag/update/name", payload, "PUT");
  },

  listTags() {
    return requestJson("/tag/list", null, "GET");
  },

  listTrainingScoreComposition(payload) {
    return requestJson("/training/scores/composition", payload, "POST");
  },
};
