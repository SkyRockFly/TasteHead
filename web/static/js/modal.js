import { core } from "./core.js";

let currentArray = [];
let owner = null;
let currentIDX = null;
let modalImage = null;
let modalMode = null;

function openModalByIndex(idx) {
  if (!owner) return;
  if (!currentArray || currentArray.length === 0) return;
  if (idx < 0 || idx >= currentArray.length) return;

  currentIDX = idx;
  const img = currentArray[idx];
  modalImage = img;

  if (modalMode === "normal") {
    renderModal();
  } else if (modalMode === "deleted") {
    console.info(modalMode);
    renderDeletedModal();
  } else {
    console.error("invalid mode:", modalMode);
    return;
  }
}

function renderModal() {
  const backdrop = document.getElementById("modal-backdrop");
  backdrop.classList.add("open");

  document.getElementById("modal-title").textContent = `#${modalImage.id}`;
  document.getElementById("modal-hash").textContent = modalImage.hash;

  const label = owner.getBatchLabel(modalImage);
  document.getElementById("modal-dir").textContent = label;
  document.getElementById("modal-path").textContent = modalImage.rel_path;

  const info = document.getElementById("modal-info");
  const actions = document.getElementById("modal-actions");

  info.replaceChildren();
  actions.replaceChildren();

  info.appendChild(renderScoreInfo(modalImage));
  actions.appendChild(renderScoreControls(modalImage));

  document.getElementById("modal-model-score").textContent = core.fmtScore(
    modalImage.model_score,
  );

  const userVal = owner.getModalScore(modalImage);
  document.getElementById("modal-user-score").textContent =
    core.fmtScore(userVal);

  const fullUrl = owner.buildFullUrl(modalImage);
  const imgEl = document.getElementById("modal-image");
  imgEl.src = fullUrl;

  const select = document.getElementById("modal-score-select");
  if (userVal === null || userVal === undefined || userVal === "") {
    select.value = "";
  } else {
    select.value = String(userVal);
  }
}

function renderDeletedModal() {
  const backdrop = document.getElementById("modal-backdrop");
  backdrop.classList.add("open");

  document.getElementById("modal-title").textContent = `#${modalImage.id}`;
  document.getElementById("modal-hash").textContent = modalImage.hash;

  const label = owner.getBatchLabel(modalImage);
  document.getElementById("modal-dir").textContent = label;
  document.getElementById("modal-path").textContent = modalImage.rel_path;

  const modal = document.getElementById("modal");

  const actions = document.getElementById("modal-actions");
  actions.replaceChildren();

  const deletedSpan = document.createElement("span");
  deletedSpan.className = "deleted-at-info";

  modal.querySelector(".deleted-at-info")?.remove();
  deletedSpan.textContent =
    core.fmtDateTime(modalImage.image_deleted_at) ||
    core.fmtDateTime(modalImage.batch_deleted_at) ||
    "";

  const fullUrl = owner.buildFullUrl(modalImage);
  const imgEl = document.getElementById("modal-image");
  imgEl.src = fullUrl;

  modal.appendChild(deletedSpan);
}

function openModal(nextOwner, img, array, mode) {
  owner = nextOwner;
  currentArray = array;
  const idx = array.findIndex((x) => x.hash === img.hash);
  if (mode === "normal") {
    modalMode = "normal";
  } else if (mode === "deleted") {
    modalMode = "deleted";
  } else {
    console.error("invalid mode argument:", mode);
    return;
  }
  if (idx !== -1) {
    openModalByIndex(idx);
  }
}

function closeModal() {
  const backdrop = document.getElementById("modal-backdrop");
  backdrop.classList.remove("open");
  currentArray = [];
  modalImage = null;
  currentIDX = -1;
  owner = null;
  modalMode = null;
}

function applyModalScore() {
  if (!modalImage || !owner) return;

  const select = document.getElementById("modal-score-select");
  const value = select.value;
  owner.applyModalScore(modalImage, value);
  openModalByIndex(currentIDX);
}

function showPrevInModal() {
  if (currentIDX < 0 || currentArray.length === 0) return;
  let nextIdx = currentIDX - 1;
  if (nextIdx < 0) {
    nextIdx = currentArray.length - 1;
  }
  openModalByIndex(nextIdx);
}

function showNextInModal() {
  if (currentIDX < 0 || currentArray.length === 0) return;
  let nextIdx = currentIDX + 1;
  if (nextIdx >= currentArray.length) {
    nextIdx = 0;
  }
  openModalByIndex(nextIdx);
}

function renderScoreControls(img) {
  const wrap = document.createElement("div");
  wrap.style.marginTop = "6px";

  const label = document.createElement("label");
  label.textContent = "Set score (0..1, step 0.25)";

  const select = document.createElement("select");
  select.id = "modal-score-select";
  select.className = "user-score";

  const options = [
    ["", "— don't change —"],
    ["0", "0.00"],
    ["0.25", "0.25"],
    ["0.5", "0.50"],
    ["0.75", "0.75"],
    ["1", "1.00"],
  ];

  for (const [value, text] of options) {
    const opt = document.createElement("option");
    opt.value = value;
    opt.textContent = text;
    select.appendChild(opt);
  }

  const btn = document.createElement("button");
  btn.id = "modal-apply-btn";
  btn.className = "primary";
  btn.style.marginTop = "4px";
  btn.textContent = "Apply to this image";

  btn.addEventListener("click", async () => {
    const value = select.value;
    if (value === "") return;

    owner.applyModalScore(img, Number(value));

    core.showToast("Score updated.");
  });

  wrap.appendChild(label);
  wrap.appendChild(select);
  wrap.appendChild(btn);

  return wrap;
}

function renderScoreInfo(img) {
  const wrap = document.createElement("div");

  const modelRow = document.createElement("div");
  modelRow.textContent = "Model's score: ";

  const modelScore = document.createElement("span");
  modelScore.id = "modal-model-score";
  modelScore.textContent = core.fmtScore(img.model_score);

  modelRow.appendChild(modelScore);

  const userRow = document.createElement("div");
  userRow.textContent = "User's score: ";

  const userScore = document.createElement("span");
  userScore.id = "modal-user-score";
  userScore.textContent = core.fmtScore(img.user_score);

  userRow.appendChild(userScore);

  wrap.appendChild(modelRow);
  wrap.appendChild(userRow);

  return wrap;
}

function init() {
  document
    .getElementById("modal-close-btn")
    .addEventListener("click", closeModal);

  document.getElementById("modal-backdrop").addEventListener("click", (e) => {
    if (e.target === e.currentTarget) {
      closeModal();
    }
  });

  document
    .getElementById("modal-prev-btn")
    .addEventListener("click", showPrevInModal);
  document
    .getElementById("modal-next-btn")
    .addEventListener("click", showNextInModal);

  document.addEventListener("keydown", (e) => {
    if (!modalImage) return;
    if (e.key === "ArrowLeft") {
      e.preventDefault();
      showPrevInModal();
    } else if (e.key === "ArrowRight") {
      e.preventDefault();
      showNextInModal();
    }
  });
}

export const modal = {
  init,
  openModal,
  closeModal,
  applyModalScore,
};
