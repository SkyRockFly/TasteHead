// modal.js
let currentArray = []

function openModalByIndex(idx, array) {
  if (!array || array.length === 0) return;
  if (idx < 0 || idx >= array.length) return;

  modalIndex = idx;
  const img = array[idx];
  modalImage = img;

  const backdrop = document.getElementById("modal-backdrop");
  backdrop.classList.add("open");

  document.getElementById("modal-title").textContent = `#${img.id}`;
  document.getElementById("modal-hash").textContent = img.hash;

  const batch = getBatchByID(img.batch_id);
  document.getElementById("modal-dir").textContent = batch
    ? batch.name
    : `batch_id=${img.batch_id}`;
  document.getElementById("modal-path").textContent = img.rel_path;

  document.getElementById("modal-model-score").textContent = fmtScore(
    img.model_score,
  );

  let userVal = img.user_score;
  if (pendingScores.has(img.id)) {
    userVal = pendingScores.get(img.id);
  }
  document.getElementById("modal-user-score").textContent =
    fmtScore(userVal);

  const fullUrl = buildFullUrl(img);
  const imgEl = document.getElementById("modal-image");
  imgEl.src = fullUrl;

  const select = document.getElementById("modal-score-select");
  if (userVal === null || userVal === undefined || userVal === "") {
    select.value = "";
  } else {
    select.value = String(userVal);
  }
}

function openModal(img,array) {
  const idx = array.findIndex((x) => x.hash === img.hash);
  currentArray = array
  if (idx !== -1) {
    openModalByIndex(idx,array);
  }
}



function closeModal() {
  const backdrop = document.getElementById("modal-backdrop");
  backdrop.classList.remove("open");
  modalImage = null;
  modalIndex = -1;
}

function applyModalScore() {
  if (!modalImage) return;

  const select = document.getElementById("modal-score-select");
  const v = select.value;

  if (v === "") {
    pendingScores.delete(modalImage.id);
  } else {
    pendingScores.set(modalImage.id, Number(v));
  }
  updatePendingCounter();
  renderGrid(images,gridIDName,makeDownloadCard);
  openModal(modalImage,currentArray); // обновить текст/значения в модалке
}

function showPrevInModal() {
  if (modalIndex < 0 || currentArray.length === 0) return;
  let nextIdx = modalIndex - 1;
  if (nextIdx < 0) {
    nextIdx = currentArray.length - 1;
  }
  openModalByIndex(nextIdx,currentArray);
}

function showNextInModal() {
  if (modalIndex < 0 || currentArray.length === 0) return;
  let nextIdx = modalIndex + 1;
  if (nextIdx >= currentArray.length) {
    nextIdx = 0;
  }
  openModalByIndex(nextIdx,currentArray);
}
