import { api } from "./api.js";
import { core } from "./core.js";
import { BaseTab, scoreOptions } from "./baseTab.js";
import { modal } from "./modal.js";

export class DeletedTab extends BaseTab {
  constructor() {
    super({
      rootID: "tab-deleted",
      gridID: "grid-deleted",
    });

    this.mode = "image";
  }

  async init() {
    this.find("#load-btn").addEventListener("click", () =>
      this.loadCurrentPage(true, true),
    );

    this.find("#next-btn").addEventListener("click", () => this.nextPage());

    this.find("#prev-btn").addEventListener("click", () => this.prevPage());

    this.find("[data-action=switch-load-mode-btn]").addEventListener(
      "click",
      () => {
        this.toggleMode();
      },
    );

    this.find("[data-action=hard-delete-btn]").addEventListener("click", () => {
      this.hardDelete();
    });

    this.find("[data-action=restore-btn]").addEventListener("click", () => {
      this.restore();
    });

    this.find("#select-all-btn").addEventListener("click", () => {
      for (const img of this.state.items) {
        this.state.selectedIDs.add(img.id);
      }
      if (this.mode === "image") {
        this.renderGrid((img) => this.makeCardImage(img));
      }
      if (this.mode === "batch") {
        this.renderGrid((batch) => this.makeCardBatch(batch));
      }
    });

    this.find("#clear-selection-btn").addEventListener("click", () => {
      this.state.selectedIDs.clear();
      if (this.mode === "image") {
        this.renderGrid((img) => this.makeCardImage(img));
      }
      if (this.mode === "batch") {
        this.renderGrid((batch) => this.makeCardBatch(batch));
      }
    });
    const root = this.getRoot();
    if (!root) {
      console.error("no root");
      return;
    }

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
    this.loadBatchesIntoState();
  }

  async loadCurrentPage(resetCursor, next) {
    if (this.mode === "image") {
      this.loadImagePage(resetCursor, next);
      return;
    } else if (this.mode === "batch") {
      this.loadBatchPage();
      return;
    }
  }

  async loadBatchPage() {
    try {
      const data = await api.listDeletedBatches();

      core.showToast("Loading...");

      this.state.items = data.batches || [];
      this.renderGrid((batch) => this.makeCardBatch(batch));
    } catch (err) {
      core.showToast(String(err), "error");
      console.error(err);
    }
  }

  async loadImagePage(resetCursor, next) {
    try {
      const hasMore = await super.loadPage(
        resetCursor,
        (payload) => api.listDeletedImages(payload),
        (img) => this.makeCardImage(img),
        (reset) => this.buildLoadPagePayload(reset, next),
      );
      this.updateHasMore(next, hasMore);
    } catch (err) {
      console.info(err);
    }
  }

  hasDeletedCursor(cursor) {
    return (
      cursor != null &&
      cursor.cursor_id > 0 &&
      cursor.cursor_deleted_at != null &&
      cursor.cursor_deleted_at !== ""
    );
  }

  updateHasMore(next, data_has_more) {
    const hasCursor =
      this.hasDeletedCursor(this.state.cursor_next) &&
      this.hasDeletedCursor(this.state.cursor_prev);

    if (next) {
      this.state.hasMore_next = !!data_has_more;
      this.state.hasMore_prev = hasCursor;
    } else {
      this.state.hasMore_prev = !!data_has_more;
      this.state.hasMore_next = hasCursor;
    }
  }

  buildLoadPagePayload(resetCursor, next) {
    const limitInput = this.find("#page-limit");
    this.state.limit = core.readInt(limitInput, 100);

    let cursorID = 0;
    let cursorDeletedAt = null;
    if (!resetCursor) {
      console.info(this.state);
      if (next) {
        cursorID = this.state.cursor_next.cursor_id;
        cursorDeletedAt = this.state.cursor_next.cursor_deleted_at;
      } else {
        cursorID = this.state.cursor_prev.cursor_id;
        cursorDeletedAt = this.state.cursor_prev.cursor_deleted_at;
      }
    }

    return {
      limit: this.state.limit,
      cursor_id: cursorID,
      cursor_deleted_at: cursorDeletedAt,
      next: next,
    };
  }

  nextPage() {
    if (!this.canGoNext()) return;
    this.loadCurrentPage(false, true);
  }

  prevPage() {
    if (!this.canGoPrev()) return;
    this.loadCurrentPage(false, false);
  }

  makeCardBatch(batch) {
    const card = document.createElement("div");
    card.className = "card";

    const checkBox = document.createElement("input");
    checkBox.type = "checkbox";
    checkBox.className = "card-checkbox";
    checkBox.checked = this.state.selectedIDs.has(batch.id);

    checkBox.addEventListener("change", () => {
      this.toggleSelection(batch.id);
    });

    const selWrap = document.createElement("div");
    selWrap.className = "card-select";

    selWrap.appendChild(checkBox);

    const metaTop = document.createElement("div");
    metaTop.className = "meta-top";

    const idSpan = document.createElement("span");
    idSpan.className = "id";
    idSpan.textContent = "#" + batch.id;

    const nameSpan = document.createElement("span");
    nameSpan.textContent = batch.name;

    metaTop.appendChild(idSpan);
    metaTop.appendChild(nameSpan);
    card.appendChild(metaTop);
    card.appendChild(selWrap);
    return card;
  }

  makeCardImage(img) {
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
      modal.openModal(this, img, this.state.items, "deleted"),
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

    const deletedTop = document.createElement("div");
    deletedTop.className = "meta-top";

    const deletedSpan = document.createElement("span");
    deletedSpan.textContent =
      `image:${core.fmtDateTime(img.image_deleted_at)}` ||
      `batch:${core.fmtDateTime(img.batch_deleted_at)}`;

    deletedTop.appendChild(deletedSpan);

    card.appendChild(thumbWrap);
    card.appendChild(metaTop);
    card.appendChild(deletedTop);
    card.appendChild(selWrap);

    return card;
  }

  toggleSelection(id) {
    if (this.state.selectedIDs.has(id)) {
      this.state.selectedIDs.delete(id);
    } else {
      this.state.selectedIDs.add(id);
    }
    if (this.mode === "image") {
      this.renderGrid((img) => this.makeCardImage(img));
    }
    if (this.mode === "batch") {
      this.renderGrid((batch) => this.makeCardBatch(batch));
    }
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
  }

  toggleMode() {
    if (this.mode === "image") {
      this.mode = "batch";
      const button = this.find('[data-action="switch-load-mode-btn"]');
      button.textContent = "Batch";
    } else if (this.mode === "batch") {
      this.mode = "image";
      const button = this.find('[data-action="switch-load-mode-btn"]');
      button.textContent = "Image";
    }
    this.state.selectedIDs.clear();
  }

  hardDelete() {
    const answer = prompt(
      "WARNING: SELECTED DATA WILL BE DELETED AND IT IS NOT RESTORABLE \n\nType DELETE to continue:",
    );

    if (answer !== "DELETE") {
      core.showToast("Hard delete cancelled.");
      return;
    }
    if (this.mode === "image") {
      this.hardDeleteImages();
    }
    if (this.mode === "batch") {
      this.hardDeleteBatch();
    }
    this.state.selectedIDs.clear();
    this.state.items = [];
    const grid = core.mustGet(this.gridID);
    grid.innerHTML = "";
  }

  async hardDeleteBatch() {
    const ids = Array.from(this.state.selectedIDs);
    const data = await api.hardDeleteBatch({ ids: ids });
    if (!data.accepted) {
      core.showToast("Delete batches not accepted", "error");
      return;
    }
    core.showToast("Deleted", "info");
  }

  async hardDeleteImages() {
    const ids = Array.from(this.state.selectedIDs);
    const data = await api.hardDeleteImages({ ids: ids });
    if (data.rejects?.length > 0) {
      const text = data.rejects
        .map((r) => `id=${r.rejected_id}: ${r.reason}`)
        .join("\n");

      core.showToast(text, "error");
    }
    core.showToast("Deleted", "info");
    this.loadImagePage(false, true);
  }

  restore() {
    if (this.state.selectedIDs.size === 0) {
      core.showToast("No selected", "error", 5000);
      return;
    }
    if (this.mode === "image") {
      this.restoreImages();
    }
    if (this.mode === "batch") {
      this.restoreBatches();
    }
    this.resetGrid();
  }

  async restoreImages() {
    const ids = Array.from(this.state.selectedIDs);
    const data = await api.restoreImages({ ids: ids });
    if (!data.accepted) {
      core.showToast("Not restored", "error");
      return;
    }
    this.loadImagePage(false, true);
  }

  async restoreBatches() {
    const ids = Array.from(this.state.selectedIDs);
    const data = await api.restoreBatches({ ids: ids });
    if (!data.accepted) {
      core.showToast("Not restored", "error");
      return;
    }
    this.loadBatchPage();
  }
}
