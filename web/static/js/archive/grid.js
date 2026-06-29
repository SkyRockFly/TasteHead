// grid.js
const gridIDName = "grid";

async function loadPage(resetCursor) {
  try {
    const dirSelect = document.getElementById("batch-select");
    const limitInput = document.getElementById("page-limit");
    const cursorInput = document.getElementById("cursor-input");
    const scoreInput = document.getElementById("score-filter");
    const scoreTypeInput = document.getElementById("score-source");

    const dirID =
      !dirSelect || dirSelect.value === "" ? null : Number(dirSelect.value);

    limit = Number(limitInput.value) || 24;

    if (resetCursor) {
      cursor = 0;
    } else {
      cursor = Number(cursorInput.value) || 0;
    }

    const score =
      scoreInput.value === "" ? null : Number(scoreInput.value.trim());
    const scoreType = scoreTypeInput.value.trim();

    setStatus("page-status", "Загружаю...", false);

    const payload = {
      dir_path: dirID,
      score_type: scoreType,
      limit: limit,
      cursor: cursor,
      score: score,
    };

    const data = await fetchJson("/images/get/info", payload, "POST");

    images = data.images || [];
    cursor = data.cursor || 0;
    hasMore = !!data.has_more;

    findInRoot("tab-download", "#cursor-input").value = cursor;

    const toolbarInfo = document.getElementById("toolbar-info");
    if (toolbarInfo) {
      toolbarInfo.textContent = `cursor=${cursor}, page size=${images.length}, has_more=${hasMore}`;
    }

    renderGrid(images, gridIDName, makeDownloadCard);
    setStatus("page-status", `Загружено ${images.length} штук.`, false);
  } catch (e) {
    setStatus("page-status", String(e), true);
  }
}

function renderGrid(imgs, gridID, renderCard) {
  const grid = mustGet(gridID);
  grid.innerHTML = "";
  console.warn(imgs);

  imgs.forEach((img) => {
    grid.appendChild(renderCard(img));
  });

  updatePendingCounter();
  updateSelectedCounter();
}

function makeDownloadCard(img) {
  const card = document.createElement("div");
  card.className = "card";

  if (pendingScores.has(img.id)) {
    card.classList.add("modified");
  }

  // чекбокс выбора
  const selWrap = document.createElement("div");
  selWrap.className = "card-select";

  const checkBox = document.createElement("input");
  checkBox.type = "checkbox";
  checkBox.className = "card-checkbox";
  checkBox.checked = selectedIDs.has(img.id);

  checkBox.addEventListener("change", () => {
    toggleSelection(img.id);
    updateSelectedCounter();
  });

  selWrap.appendChild(checkBox);

  const thumbWrap = document.createElement("div");
  thumbWrap.className = "thumb-wrapper";

  const imgEl = document.createElement("img");
  imgEl.className = "thumb";
  imgEl.loading = "lazy";
  imgEl.src = buildThumbUrl(img);
  imgEl.alt = img.id;

  thumbWrap.appendChild(imgEl);
  thumbWrap.addEventListener("click", () => openModal(img, images));

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
  modelDiv.textContent = "model: " + fmtScore(img.model_score);

  const userDiv = document.createElement("div");
  userDiv.className = "score-pill user";

  let userVal = img.user_score;
  if (pendingScores.has(img.id)) {
    userVal = pendingScores.get(img.id);
  }

  if (userVal === null || userVal === undefined) {
    userDiv.classList.add("empty");
    userDiv.textContent = "user: —";
  } else {
    userDiv.textContent = "user: " + fmtScore(userVal);
  }

  scoresDiv.appendChild(modelDiv);
  scoresDiv.appendChild(userDiv);

  const select = document.createElement("select");
  select.className = "user-score";

  SCORE_OPTIONS.forEach((val) => {
    const opt = document.createElement("option");
    opt.value = val;
    opt.textContent =
      val === "" ? "— оставить как есть —" : Number(val).toFixed(2);
    select.appendChild(opt);
  });

  let currentScore = pendingScores.has(img.id)
    ? pendingScores.get(img.id)
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
      pendingScores.delete(img.id);
    } else {
      pendingScores.set(img.id, Number(v));
    }
    updatePendingCounter();
    renderGrid(images, gridIDName, makeDownloadCard);
  });

  card.appendChild(thumbWrap);
  card.appendChild(metaTop);
  card.appendChild(scoresDiv);
  card.appendChild(select);
  card.appendChild(selWrap);

  return card;
}

function toggleSelection(id) {
  if (selectedIDs.has(id)) {
    selectedIDs.delete(id);
  } else {
    selectedIDs.add(id);
  }
  renderGrid(images, gridIDName, makeDownloadCard);
}

async function saveAllScores() {
  if (pendingScores.size === 0) {
    setStatus("save-status", "Нет изменённых оценок.", false);
    return;
  }

  const scores = [];
  for (const [id, score] of pendingScores.entries()) {
    scores.push({ id: id, score: score });
  }

  try {
    setStatus("save-status", "Отправляю " + scores.length + " оценок…", false);

    const payload = { scores };
    const data = await fetchJson("/scores/update", payload, "POST");

    if (!data.accepted) {
      throw new Error("backend вернул accepted=false");
    }

    images.forEach((img) => {
      if (pendingScores.has(img.id)) {
        img.user_score = pendingScores.get(img.id);
      }
    });

    pendingScores.clear();
    renderGrid(images, gridIDName, makeDownloadCard);
    setStatus("save-status", "Сохранено.", false);
  } catch (e) {
    setStatus("save-status", String(e), true);
  }
}

function resetLocalChanges() {
  pendingScores.clear();
  renderGrid(images, gridIDName, makeDownloadCard);
  setStatus("save-status", "Локальные изменения сброшены.", false);
}
