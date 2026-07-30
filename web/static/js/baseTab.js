import { api } from "./api.js";
import { core } from "./core.js";

export const scoreOptions = ["", "0", "0.25", "0.5", "0.75", "1"];

export class BaseTab {
  constructor({ rootID, gridID, statusID = null, toolbarID = null }) {
    this.rootID = rootID;
    this.gridID = gridID;
    this.statusID = statusID;
    this.toolbarID = toolbarID;

    this.state = {
      cursor_next: 0,
      cursor_prev: 0,
      limit: 24,
      hasMore_next: false,
      hasMore_prev: false,
      items: [],
      batches: [],
      tags: [],
      selectedIDs: new Set(),
      pendingScores: new Map(),
    };
  }

  getRoot() {
    return document.getElementById(this.rootID);
  }

  find(selector) {
    return core.findInRoot(this.rootID, selector);
  }

  getBatchByID(id) {
    return this.state.batches.find((b) => b.id === id) || null;
  }

  buildThumbUrl(img) {
    const batch = this.getBatchByID(img.batch_id);
    if (!batch) {
      console.warn("batch not found", img.batch_id);
      return "";
    }

    const rel = core.normalisePath(batch.rel_path + "/thumbs/" + img.rel_path);
    return api.getImageUrl(rel);
  }

  buildFullUrl(img) {
    const batch = this.getBatchByID(img.batch_id);
    if (!batch) {
      console.warn("batch not found", img.batch_id);
      return "";
    }

    const rel = core.normalisePath(batch.rel_path + "/" + img.rel_path);
    return api.getImageUrl(rel);
  }

  updateCounter(selector, label, size) {
    const item = this.find(selector);
    item.textContent = label + ": " + size;
  }

  renderGrid(renderCard) {
    const grid = core.mustGet(this.gridID);
    grid.innerHTML = "";

    for (const item of this.state.items) {
      grid.appendChild(renderCard(item));
    }
    this.updateCounter(
      "#selected-counter",
      "Selected",
      this.state.selectedIDs.size,
    );

    this.updateCounter(
      "#pending-counter",
      "Pending",
      this.state.pendingScores.size,
    );
  }

  updateToolbar() {
    if (!this.toolbarID) return;

    const toolbarInfo = document.getElementById(this.toolbarID);
    if (!toolbarInfo) return;

    toolbarInfo.textContent =
      "page size=" +
      this.state.items.length +
      ", has_more=" +
      this.state.hasMore;
  }

  applyPageResult(data) {
    this.state.items = data.images || data.rows || [];
    this.state.cursor_prev = data.cursor_prev || 0;
    this.state.cursor_next = data.cursor_next || 0;
  }

  canGoNext() {
    if (this.state.hasMore_next) return true;

    core.showToast("has_more=false, no more images.");
    return false;
  }
  canGoPrev() {
    if (this.state.hasMore_prev) return true;

    core.showToast("has_more=false, no more images.");
    return false;
  }

  async loadPage(resetCursor, handler, renderCard, buildPayload) {
    let hasMore = false;
    try {
      const payload = buildPayload(resetCursor);
      if (resetCursor) {
        this.state.hasMore_next = false;
        this.state.hasMore_prev = false;
      }

      core.showToast("Loading...");

      const data = await handler(payload);

      this.applyPageResult(data);

      this.renderGrid(renderCard);
      this.updateToolbar();

      core.showToast("Loaded " + this.state.items.length + " items.");
      hasMore = !!data.has_more;
    } catch (err) {
      core.showToast(String(err), "error");
      console.error(err);
    }
    return hasMore;
  }

  async listTags() {
    const data = await api.listTags();
    this.state.tags = data.tags || [];
    const root = this.getRoot();
    if (!root) {
      console.error("no root");
      return;
    }
    const selects = root.querySelectorAll(".tag-select");
    if (selects.length != 0) {
      for (const select of selects) {
        select.innerHTML = "";
        const allowNew = select.dataset.allowNew === "true";

        const placeholder = document.createElement("option");
        placeholder.value = "";
        placeholder.textContent = "- choose tag -";
        select.appendChild(placeholder);

        for (const t of this.state.tags) {
          const opt = document.createElement("option");
          opt.value = t.id;
          opt.textContent = t.name;
          select.appendChild(opt);
        }
        if (allowNew) {
          const newOpt = document.createElement("option");
          newOpt.value = "__new__";
          newOpt.textContent = "Add new tag";
          select.appendChild(newOpt);
        }
      }
    }
  }

  async createNewTag() {
    const name = prompt("New tag's name:");
    if (!name || !name.trim()) {
      await this.listTags();
      return;
    }

    const desc = prompt("tag's desc (optional):") || "";

    const payload = {
      name: name.trim(),
      desc: desc.trim(),
    };

    const created = await api.createTag(payload);
    console.log("created tag:", created);

    await this.listTags();
  }

  async removeTags() {
    const selectInput = this.find('[data-action="list-tags"]');
    const tagID = Number(selectInput.value);

    if (Number.isNaN(tagID)) {
      console.error("tag id is not a number:", id);
      return;
    }
    const payload = {
      id: tagID,
    };

    const data = await api.removeTag(payload);

    if (!data.accepted) {
      core.showToast("tag is not deleted", "error");
    }
    core.showToast("tag is deleted");
    await this.listTags();
  }

  resetSelected() {
    this.state.selectedIDs.clear();
    this.updateCounter(
      "#selected-counter",
      "Selected",
      this.state.selectedIDs.size,
    );
  }

  resetGrid() {
    const el = core.mustGet(this.gridID);
    el.innerHTML = "";
  }

  async loadBatchesIntoState() {
    const data = await api.listBatches();
    this.state.batches = data.batches || [];
  }

  async loadBatches() {
    try {
      const data = await api.listBatches();
      const batches = data.batches ?? [];
      this.state.batches = [...batches].sort((a, b) =>
        String(a.name ?? "").localeCompare(String(b.name ?? ""), undefined, {
          sensitivity: "base",
          numeric: true,
        }),
      );
      const tab = document.getElementById(this.rootID);
      const selects = tab.querySelectorAll(".batch-select");
      if (selects.length === 0) {
        return;
      }
      for (const select of selects) {
        select.innerHTML = "";

        const optAll = document.createElement("option");
        optAll.value = "";
        optAll.textContent = "All batches";
        select.appendChild(optAll);

        for (const batch of this.state.batches) {
          const opt = document.createElement("option");
          opt.value = batch.id;
          opt.textContent = batch.name;
          select.appendChild(opt);
        }
      }
    } catch (err) {
      console.error("loadBatches error", err);
    }
  }

  setInputValue(selector, value) {
    const el = this.find(selector);
    el.value = value ?? "";
  }

  async saveAllScores() {
    if (this.state.pendingScores.size === 0) {
      core.showToast("No changed scores");
      return;
    }

    const scores = [];
    for (const [id, score] of this.state.pendingScores.entries()) {
      scores.push({ id: id, score: score });
    }

    try {
      core.showToast("Send " + scores.length + "scores", "info");

      const data = await api.updateUserScores({ scores });

      if (!data.accepted) {
        throw new Error("backend returned accepted=false");
      }

      for (const img of this.state.items) {
        if (this.state.pendingScores.has(img.id)) {
          img.user_score = this.state.pendingScores.get(img.id);
        }
      }

      this.state.pendingScores.clear();
      this.renderGrid((img) => this.makeCard(img));
      core.showToast("Saved");
    } catch (e) {
      core.showToast(String(e), "error", 7000);
    }
  }

  getBatchLabel(item) {
    const batch = this.getBatchByID(item.id);
    return batch ? batch.name : "batch_id=" + item.batch_id;
  }
}
