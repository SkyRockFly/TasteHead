import { api } from "./api.js";
import { core } from "./core.js";
import { BaseTab, scoreOptions } from "./baseTab.js";
import { modal } from "./modal.js";

export class TrainingTab extends BaseTab {
  constructor() {
    super({
      rootID: "tab-training",
      gridID: "grid-training",
      statusID: "page-status",
      toolbarID: "toolbar-info",
    });

    this.state.deleted_at = null;
  }

  async init() {
    this.find("#load-btn").addEventListener("click", () =>
      this.loadTrainingPage(true, true),
    );

    this.find("#next-btn").addEventListener("click", () =>
      this.nextPage(() => this.loadTrainingPage(false, true)),
    );

    this.find("#prev-btn").addEventListener("click", () =>
      this.prevPage(() => this.loadTrainingPage(false, false)),
    );

    this.find("#save-all-btn").addEventListener("click", () => {
      this.saveAllScores();
    });

    this.find("#reset-local-btn").addEventListener("click", () => {
      this.resetLocalChanges();
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

    this.find('[data-action="train-model"]').addEventListener("click", () => {
      this.trainModel();
    });

    this.find('[data-action="update-images-tag"]').addEventListener(
      "click",
      () => {
        this.updateRowsTag();
      },
    );

    this.find("#score-composition-btn").addEventListener("click", () => {
      this.listScoreComposition();
    });
    this.listTags();
    this.loadBatchesIntoState();
  }

  buildPayload(resetCursor, next) {
    const tagIDInput = this.find('[data-action="list-tags"]');
    const limitInput = this.find("#page-limit");
    const scoreInput = this.find("#score-filter");

    this.state.limit = core.readInt(limitInput, 24);
    let cursor = 0;
    if (!resetCursor) {
      if (next) {
        cursor = this.state.cursor_next;
      } else {
        cursor = this.state.cursor_prev;
      }
    }

    const payload = {
      tag_id: core.readIdOrNull(tagIDInput),
      limit: this.state.limit,
      cursor: cursor,
      user_score: core.readScoreOrAll(scoreInput),
      next: next,
    };
    return payload;
  }

  resetLocalChanges() {
    this.state.pendingScores.clear();
    this.renderGrid((img) => this.makeCard(img));
    core.showToast("Local changes were reset");
  }

  async loadTrainingPage(resetCursor, next) {
    const hasMore = await this.loadPage(
      resetCursor,
      (payload) => api.listTrainingRows(payload),
      (img) => this.makeCard(img),
      (reset) => this.buildPayload(reset, next),
    );
    this.updateHasMore(next, hasMore);
  }

  nextPage() {
    if (!this.canGoNext()) return;
    this.loadTrainingPage(false, true);
  }

  prevPage() {
    if (!this.canGoPrev()) return;
    this.loadTrainingPage(false, false);
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

    const select = document.createElement("select");
    select.className = "user-score";

    scoreOptions.forEach((val) => {
      const opt = document.createElement("option");
      opt.value = val;
      opt.textContent =
        val === "" ? "— оставить как есть —" : Number(val).toFixed(2);
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
      this.renderGrid(images, gridIDName, this.makeCard);
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

  async trainModel() {
    const modelNameInput = this.find('[data-action="model-name-input"]');
    const tagIDInput = this.find('[data-action="train-model-tags"]');

    const tagID = core.readInt(tagIDInput);
    const modelName = String(modelNameInput.value).trim();

    const payload = {
      model_name: modelName,
      tag_id: tagID,
    };
    const data = await api.trainModel(payload);
    if (!data.accepted) {
      throw Error("training was interrupted");
    }
  }

  async updateRowsTag() {
    const tagIDInput = this.find('[data-action="list-tags"]');
    const tagID = core.readInt(tagIDInput);

    const ids = Array.from(this.state.selectedIDs);

    const payload = {
      tag_id: tagID,
      ids: ids,
    };

    const data = await api.updateTrainingRowTag(payload);
    if (!data.accepted) {
      core.showToast("Error updating images tag", "error");
      return;
    }
    core.showToast("Images' tag updated");
  }

  async listScoreComposition() {
    const tagIDInput = this.find('[data-action="list-tags"]');
    const tagID = core.readInt(tagIDInput);
    if (!tagID) {
      core.showToast("no tag selected", "error");
      return;
    }

    const data = await api.listTrainingScoreComposition({
      tag_id: tagID,
    });
    const scores = data || [];

    if (data.length === 0) {
      core.showToast("No scores", "error");
      return;
    }

    const div = this.find('[data-class="score-composition"]');

    const parts = scores.map((row) => {
      return `${row.score} : ${row.count}`;
    });

    div.textContent = parts.join(" | ");
    core.showToast("Images' tag updated");
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
