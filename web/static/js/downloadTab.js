import { api } from "./api.js";
import { core } from "./core.js";
import { BaseTab, scoreOptions } from "./baseTab.js";
import { modal } from "./modal.js";

export class DownloadTab extends BaseTab {
  constructor() {
    super({
      rootID: "tab-download",
      gridID: "grid-download",
      statusID: "page-status",
      toolbarID: "toolbar-info",
    });
  }

  async init() {
    this.find("#load-btn").addEventListener("click", () =>
      this.loadDownloadPage(true, true, false),
    );

    this.find("#next-btn").addEventListener("click", () =>
      this.nextPage(() => this.loadDownloadPage(false, true, false)),
    );

    this.find("#prev-btn").addEventListener("click", () =>
      this.prevPage(() => this.loadDownloadPage(false, false, false)),
    );

    this.find("#save-all-btn").addEventListener("click", () => {
      this.saveAllScores();
    });

    this.find("#reset-local-btn").addEventListener("click", () => {
      this.resetLocalChanges();
    });

    this.find("#dl-btn").addEventListener("click", () => {
      this.downloadBatch();
    });

    this.find("#refresh-batches-btn").addEventListener("click", () => {
      this.loadBatches();
    });

    this.find("#select-all-btn").addEventListener("click", () => {
      for (const img of this.state.items) {
        this.state.selectedIDs.add(img.id);
      }
      this.renderGrid((img) => this.makeCard(img));
    });

    this.find("#clear-selection-btn").addEventListener("click", () => {
      this.state.selectedIDs.clear();
      this.renderGrid((img) => this.makeCard(img));
    });
    const root = this.getRoot();
    if (!root) {
      console.error("no root");
      return;
    }

    this.find("#load-local-batches").addEventListener("click", () => {
      this.importLocalDirs();
    });

    const tagSelects = root.querySelectorAll(".tag-select");
    if (tagSelects.length !== 0) {
      for (const tagSelect of tagSelects) {
        tagSelect.addEventListener("change", async (e) => {
          const val = e.target.value;
          if (val === "__new__") {
            await this.createNewTag();
          }
        });
      }
    }

    this.find('[data-action="remove-images-btn"]').addEventListener(
      "click",
      () => {
        this.removeImages();
      },
    );

    this.find('[data-action="send-training-btn"]').addEventListener(
      "click",
      () => {
        this.sendToDataset();
      },
    );

    this.find('[data-action="find-unfinished-batches"]').addEventListener(
      "click",
      () => {
        this.findUnfinishedBatches();
      },
    );

    this.find('[data-action="find-batches-fails"]').addEventListener(
      "click",
      () => {
        this.findBatchesWithFails();
      },
    );

    this.find('[data-action="resume-scrape-btn"]').addEventListener(
      "click",
      () => {
        this.resumeScrape();
      },
    );

    this.find('[data-action="solve-batch-fails"]').addEventListener(
      "click",
      () => {
        this.solveFails();
      },
    );

    this.find('[data-action="update-batch-name-btn"]').addEventListener(
      "click",
      () => {
        this.updateBatchName();
      },
    );

    this.find('[data-action="remove-batch-btn"]').addEventListener(
      "click",
      () => {
        this.removeBatch();
      },
    );

    this.find('[data-action="move-images-to-batch-btn"]').addEventListener(
      "click",
      () => {
        this.moveImagesToBatch();
      },
    );

    this.find('[data-action="create-batch-btn"]').addEventListener(
      "click",
      () => {
        this.createBatch();
      },
    );

    this.find('[data-action="delete-tag-btn"]').addEventListener(
      "click",
      () => {
        this.removeTags();
      },
    );

    this.find('[data-action="mass-scoring-btn"]').addEventListener(
      "click",
      () => {
        this.massScore();
      },
    );

    this.find('[data-action="refresh-model-list-btn"]').addEventListener(
      "click",
      () => {
        this.listModels();
      },
    );

    this.find('[data-action="set-model-btn"]').addEventListener("click", () => {
      this.setModel();
    });

    this.loadBatches();
    this.listTags();
    this.listModels();
  }
  async importLocalDirs() {
    try {
      const resp = await api.importLocalDirs();
      if (!resp.ok) {
        console.error("importLocalDirs HTTP", resp.status);
        return;
      }
      console.log("importLocalDirs done");
    } catch (err) {
      console.error("importLocalDirs error", err);
    }
  }

  buildLoadPagePayload(resetCursor, next, refresh) {
    const dirSelect = this.find('[data-action="batch-select"]');
    const limitInput = this.find("#page-limit");
    const scoreInput = this.find("#score-filter");
    const scoreTypeInput = this.find("#score-source");

    this.state.limit = core.readInt(limitInput, 100);

    let cursor = 0;
    if (!resetCursor) {
      if (!refresh) {
        if (next) {
          cursor = this.state.cursor_next;
        } else {
          cursor = this.state.cursor_prev;
        }
      } else {
        cursor = this.state.cursor_prev - 1;
      }
    }

    return {
      dir_path: core.readIdOrNull(dirSelect),
      score_type: String(scoreTypeInput.value).trim(),
      limit: this.state.limit,
      cursor: cursor,
      score: core.readScoreOrAll(scoreInput),
      next: next,
    };
  }

  async loadDownloadPage(resetCursor, next, refresh) {
    const hasMore = await this.loadPage(
      resetCursor,
      (payload) => api.listImages(payload),
      (img) => this.makeCard(img),
      (reset) => this.buildLoadPagePayload(reset, next, refresh),
    );
    this.updateHasMore(next, hasMore);
    this.getScrapeState();
  }

  nextPage() {
    if (!this.canGoNext()) return;
    this.loadDownloadPage(false, true, false);
  }

  async massScore() {
    console.info("kek");
    if (this.state.selectedIDs.size === 0) {
      core.showToast("No selected", "error");
      return;
    }

    const ok = confirm("Mass assign score to selected?");
    if (!ok) return;
    const select = this.find("#mass-score-select");
    const score = Number(select.value);

    const scores = Array.from(this.state.selectedIDs).map((id) => ({
      id: id,
      score: score,
    }));

    try {
      const data = await api.updateUserScores({ scores: scores });
      if (!data.accepted) {
        core.showToast("Not accepted", "error");
      }
    } catch (err) {
      core.showToast(err, "error");
    }
    for (const img of this.state.items) {
      if (this.state.selectedIDs.has(img.id)) {
        img.user_score = score;
      }
    }
    this.renderGrid((img) => this.makeCard(img));
    core.showToast("Saved");
  }

  prevPage() {
    if (!this.canGoPrev()) return;
    this.loadDownloadPage(false, false, false);
  }

  updateHasMore(next, data_has_more) {
    const hasCursor = this.state.cursor_next > 0 && this.state.cursor_prev > 0;
    if (next) {
      this.state.hasMore_next = data_has_more;
      this.state.hasMore_prev = hasCursor;
    } else {
      this.state.hasMore_prev = data_has_more;
      this.state.hasMore_next = hasCursor;
    }
  }

  makeCard(img) {
    const card = document.createElement("div");
    card.className = "card";

    if (this.state.pendingScores.has(img.id)) {
      card.classList.add("modified");
    }

    const selWrap = document.createElement("div");
    selWrap.className = "card-select";

    const checkBox = document.createElement("input");
    checkBox.type = "checkbox";
    checkBox.className = "card-checkbox";
    checkBox.checked = this.state.selectedIDs.has(img.id);

    checkBox.addEventListener("change", () => {
      this.toggleSelection(img.id);
    });

    selWrap.appendChild(checkBox);

    const thumbWrap = document.createElement("div");
    thumbWrap.className = "thumb-wrapper";

    const imgEl = document.createElement("img");
    imgEl.className = "thumb";
    imgEl.loading = "lazy";
    imgEl.src = this.buildThumbUrl(img);
    imgEl.alt = img.id;

    thumbWrap.appendChild(imgEl);
    thumbWrap.addEventListener("click", () =>
      modal.openModal(this, img, this.state.items, "normal"),
    );

    const metaTop = document.createElement("div");
    metaTop.className = "meta-top";

    const idSpan = document.createElement("span");
    idSpan.className = "id";
    idSpan.textContent = "#" + img.id;

    const hashSpan = document.createElement("span");
    hashSpan.textContent = img.hash.slice(0, 8) + "…";

    metaTop.appendChild(idSpan);
    metaTop.appendChild(hashSpan);

    const scoresDiv = document.createElement("div");
    scoresDiv.className = "scores";

    const modelDiv = document.createElement("div");
    modelDiv.className = "score-pill model";
    modelDiv.textContent = "model: " + core.fmtScore(img.model_score);

    const userDiv = document.createElement("div");
    userDiv.className = "score-pill user";

    let userVal = img.user_score;
    if (this.state.pendingScores.has(img.id)) {
      userVal = this.state.pendingScores.get(img.id);
    }

    if (userVal === null || userVal === undefined) {
      userDiv.classList.add("empty");
      userDiv.textContent = "user: —";
    } else {
      userDiv.textContent = "user: " + core.fmtScore(userVal);
    }

    scoresDiv.appendChild(modelDiv);
    scoresDiv.appendChild(userDiv);

    const tags = img.tags ?? [];
    if (tags.length > 0) {
      const tagLine = document.createElement("div");
      tagLine.className = "training-tags";
      tagLine.textContent = `Tags: ${tags.join(", ")}`;

      card.appendChild(tagLine);
    }

    const select = document.createElement("select");
    select.className = "user-score";

    scoreOptions.forEach((val) => {
      const opt = document.createElement("option");
      opt.value = val;
      opt.textContent = val === "" ? "— no changes —" : Number(val).toFixed(2);
      select.appendChild(opt);
    });

    let currentScore = this.state.pendingScores.has(img.id)
      ? this.state.pendingScores.get(img.id)
      : img.user_score;

    if (
      currentScore !== null &&
      currentScore !== undefined &&
      currentScore !== ""
    ) {
      select.value = String(currentScore);
    } else {
      select.value = "";
    }

    select.addEventListener("change", () => {
      const v = select.value;
      if (v === "") {
        this.state.pendingScores.delete(img.id);
      } else {
        this.state.pendingScores.set(img.id, Number(v));
      }
      this.renderGrid((img) => this.makeCard(img));
    });

    card.appendChild(thumbWrap);
    card.appendChild(metaTop);
    card.appendChild(scoresDiv);
    card.appendChild(select);
    card.appendChild(selWrap);

    return card;
  }

  toggleSelection(id) {
    if (this.state.selectedIDs.has(id)) {
      this.state.selectedIDs.delete(id);
    } else {
      this.state.selectedIDs.add(id);
    }
    this.renderGrid((img) => this.makeCard(img));
  }

  resetLocalChanges() {
    this.state.pendingScores.clear();
    this.renderGrid((img) => this.makeCard(img));
    core.showToast("Local changes were reset");
  }

  async findUnfinishedBatches() {
    const data = await api.listUnfinishedBatches();
    const batches = data.batches || [];

    const select = this.find('[data-id="unfinished-batch-select"]');
    if (!select) return;

    select.innerHTML = "";

    if (batches.length === 0) {
      core.showToast("No unfinished batches");
      return;
    }

    for (const batch of batches) {
      const opt = document.createElement("option");
      opt.value = batch;
      opt.textContent = batch;
      select.appendChild(opt);
    }
  }

  async findBatchesWithFails() {
    const data = await api.listBatchesWithFails();
    const batches = data.batches || [];

    const select = this.find('[data-id="batches-with-fails"]');
    if (!select) return;

    select.innerHTML = "";

    if (batches.length === 0) {
      core.showToast("No batches with fails");
      return;
    }

    for (const batch of batches) {
      const opt = document.createElement("option");
      opt.value = batch;
      opt.textContent = batch;
      select.appendChild(opt);
    }
  }

  async createBatch() {
    const name = prompt("New batch name:");
    if (!name || !name.trim()) {
      await this.loadBatches();
      return;
    }

    const payload = {
      name: name.trim(),
    };

    try {
      const id = await api.createBatch(payload);
    } catch (err) {
      core.showToast(`Error:${err}`, "error");
    }

    await this.loadBatches();
  }

  async removeImages() {
    const ok = confirm("Selected images will be deleted");

    if (!ok) return;
    const ids = Array.from(this.state.selectedIDs);

    const data = await api.removeImages({ ids: ids });
    if (!data.accepted) {
      core.showToast("Error deleting images", "error");
      return;
    }
    this.state.selectedIDs.clear();
    this.loadDownloadPage(false, true, false);
    core.showToast("Accepted");
  }

  async listModels() {
    const data = await api.listModels();
    const models = data.models || [];
    if (models.length === 0) {
      core.showToast("No models found", "error");
      return;
    }
    const select = this.find('[data-action="list-models"]');
    select.innerHTML = "";

    for (const model of models) {
      const option = document.createElement("option");
      option.textContent = model;
      option.value = model;

      select.appendChild(option);
    }
    core.showToast("Models loaded", "info");
  }

  async setModel() {
    const select = this.find('[data-action="list-models"]');
    const name = String(select.value).trim();

    if (name === "") {
      core.showToast("No model selected", "error");
      return;
    }

    const data = await api.setModel({ model_name: name });

    if (!data.accepted) {
      core.showToast("Model not accepted", "error");
      return;
    }
    core.showToast("Model accepted", "info");
  }

  async getScrapeState() {
    const select = this.find('[data-action="batch-select"]');
    const id = Number(select.value) || 0;
    if (!Number.isInteger(id) || id <= 0) {
      core.showToast("Batch not selected", "error");
      return;
    }
    try {
      const data = await api.getScrapeState({ id });

      this.setInputValue("#dl-url", data.next_url);
      this.setInputValue("#dl-post-selector", data.post_selector);
      this.setInputValue("#dl-image-attr", data.image_attr);
      this.setInputValue("#dl-next-selector", data.next_page_selector);
      this.setInputValue("#dl-next-attr", data.next_page_attr);

      core.showToast("Scrape state loaded", "success");
    } catch (err) {
      console.error(err);
      core.showToast("Failed to load scrape state", "error");
    }
  }

  async removeBatch() {
    const ok = confirm("Selected batch will be deleted");

    if (!ok) return;
    const batchSelect = this.find('[data-action="batch-select"]');
    const batchID = core.readInt(batchSelect);

    const data = await api.removeBatch({ id: batchID });
    if (!data.accepted) {
      core.showToast("Error deleting images", "error");
      return;
    }
    this.loadBatches();
    const grid = core.mustGet(this.gridID);
    grid.innerHTML = "";
    core.showToast("Accepted");
  }

  async resumeScrape() {
    const select = this.find('[data-id="unfinished-batch-select"]');
    if (!select) return;

    const batchName = select.value;

    const payload = {
      batch_name: batchName,
    };

    try {
      const data = await api.resumeScrape(payload);
      select.innerHTML = "";
    } catch (err) {
      core.showToast(`resumeScrape:${err}`, "error");
    }
  }

  async solveFails() {
    const select = this.find('[data-id="batches-with-fails"]');
    if (!select) return;

    const batchName = select.value;

    const payload = {
      batch_name: batchName,
    };

    try {
      const data = await api.solveDownloadFails(payload);
      select.innerHTML = "";
    } catch (err) {
      core.showToast(`solveFails:${err}`, "error");
    }
  }

  async downloadBatch() {
    const url = this.find("#dl-url").value.trim();
    const postSel = this.find("#dl-post-selector").value.trim();
    const imgAttr = this.find("#dl-image-attr").value.trim();
    const nextSel = this.find("#dl-next-selector").value.trim();
    const nextAttr = this.find("#dl-next-attr").value.trim();
    const pages = Number(this.find("#dl-pages").value) || 1;
    const dlLimit = Number(this.find("#dl-limit").value) || 24;

    if (!url) {
      core.showToast("Need URL");
      return;
    }

    const payload = {
      url: url,
      post_selector: postSel,
      image_attr: imgAttr,
      next_page_selector: nextSel,
      next_page_attr: nextAttr,
      pages: pages,
      limit: dlLimit,
    };

    try {
      core.showToast("Downloading and sending to CLIP/head…", "info", 20000);

      const data = await api.scrapeImages(payload);

      let msg = `Downloaded and parsed: ${(data.images || []).length}`;
      if (data.dir) {
        msg += `\ndir: ${data.dir}`;
      }
      if (data.errs && data.errs.length) {
        msg += `\nErrors:\n${data.errs}`;
      }
      core.showToast(msg, "info", 7000);

      await this.loadBatches();
    } catch (e) {
      core.showToast(String(e), "error", 7000);
    }
  }

  async moveImagesToBatch() {
    const batchInput = this.find('[data-action="move-to-batch"]');
    const batchID = core.readInt(batchInput);

    const ids = Array.from(this.state.selectedIDs);

    const payload = {
      ids: ids,
      toBatchID: batchID,
    };

    const data = await api.moveImages(payload);
    console.info(data);
    if (data.rejects?.length > 0) {
      const rejectedText = data.rejects
        .map((item) => `${r.id}: ${r.reason}`)
        .join("\n");
      core.showToast(rejectedText, "error", 10000);
    }
    core.showToast("Accepted", "info", 7000);
    this.resetSelected();
    this.resetGrid();
    this.loadDownloadPage(false, true, true);
  }

  async sendToDataset() {
    const selectInput = this.find('[data-action="list-tags"]');
    const tagID = Number(selectInput.value);

    if (!tagID) {
      core.showToast("tag is empty", "error");
      return;
    }

    const imageIDs = Array.from(this.state.selectedIDs);

    if (imageIDs.length === 0) {
      core.showToast("no images selected", "error");
      return;
    }

    const payload = {
      tag_id: tagID,
      image_ids: imageIDs,
    };
    try {
      const data = await api.createTrainingRows(payload);
      console.info(data);

      if (data.duplicates?.length !== 0) {
        const ids = data.duplicates.map((item) => {
          if (typeof item === "object" && item !== null) {
            return item;
          }
        });

        const visible = ids
          .slice(0, 10)
          .map((id) => "#" + id)
          .join(", ");

        const tail = ids.length > 10 ? `... +${ids.length - 10}` : "";

        core.showToast(
          `Already in dataset (${ids.length}) : ${visible}${tail}`,
          "error",
          7000,
        );
        return;
      }
    } catch (err) {
      console.error("createTrainingRows error:", err);
    }
    this.loadDownloadPage(false, true, true);
    core.showToast("Accepted", "info", 7000);
  }

  async updateBatchName() {
    const batchInput = this.find('[data-action="batch-select"]');
    const batchID = core.readInt(batchInput);

    const name = prompt("Enter batch name:");
    if (name === null) return;

    const trimmedName = name.trim();
    if (trimmedName === "") {
      core.showToast("Tag's name can't be empty:", "error");
      return;
    }

    const payload = {
      batch_id: batchID,
      batch_name: trimmedName,
    };

    const data = await api.updateBatchName(payload);

    if (!data.accepted) {
      core.showToast("Not accepted", "error");
      return;
    }
  }

  applyModalScore(img, value) {
    if (value === "") {
      this.state.pendingScores.delete(img.id);
    } else {
      this.state.pendingScores.set(img.id, Number(value));
    }
    this.renderDownloadGrid();
  }

  renderDownloadGrid() {
    this.renderGrid((item) => this.makeCard(item));
  }

  getModalScore(img) {
    if (this.state.pendingScores.has(img.id)) {
      return this.state.pendingScores.get(img.id);
    }

    return img.user_score;
  }
}
