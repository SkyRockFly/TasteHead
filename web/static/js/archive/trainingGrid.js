let trainingRows = [];

var gridTrainingName = "grid-training";

async function loadTrainingRows(resetCursor) {
  const tagIDInput = findInRoot("tab-training", ".tag-select");
  const userScoreInput = findInRoot("tab-training", "#score-filter");
  const limitInput = findInRoot("tab-training", "#page-limit");
  const cursorInput = findInRoot("tab-training", "#cursor-input");

  const limit = readInt(limitInput);
  const userScore = readScoreOrAll(userScoreInput);
  const tagID = readIdOrNull(tagIDInput);

  if (resetCursor) {
    cursor = 0;
  } else {
    cursor = readInt(cursorInput);
  }
  const payload = {
    tag_id: tagID,
    user_score: userScore,
    limit: limit,
    cursor: cursor,
  };

  const data = await sendJson("/training/list/req", payload, "POST");
  trainingRows = data.rows || [];
  cursor = data.cursor || 0;
  hasMore = data.has_more || false;

  findInRoot("tab-training", "#cursor-input").value = cursor;

  renderGrid(trainingRows, gridTrainingName, makeTrainingCard);
}

async function moveToDataset() {
  const tagInput = findInRoot("tab-download", ".tag-select");
  const tagID = readInt(tagInput);
  if (!tagID) {
    throw new Error("tagID is not selected");
  }

  const imageIDs = Array.from(selectedIDs);

  payload = {
    tag_id: tagID,
    image_ids: imageIDs,
  };

  const data = await sendJson("/training/create", payload, "POST");

  if (!data.accepted) {
    throw new Error("rows to dataset not accepted");
  }
}

function makeTrainingCard(row) {
  const card = document.createElement("div");
  card.className = "card";

  if (pendingScores.has(row.id)) {
    card.classList.add("modified");
  }

  // чекбокс выбора
  const selWrap = document.createElement("div");
  selWrap.className = "card-select";

  const checkBox = document.createElement("input");
  checkBox.type = "checkbox";
  checkBox.className = "card-checkbox";
  checkBox.checked = selectedIDs.has(row.id);

  checkBox.addEventListener("change", () => {
    toggleSelection(row.id);
    updateSelectedCounter();
  });

  selWrap.appendChild(checkBox);

  const thumbWrap = document.createElement("div");
  thumbWrap.className = "thumb-wrapper";

  const imgEl = document.createElement("img");
  imgEl.className = "thumb";
  imgEl.loading = "lazy";
  imgEl.src = buildThumbUrl(row);
  imgEl.alt = row.id;

  thumbWrap.appendChild(imgEl);
  thumbWrap.addEventListener("click", () => openModal(row, trainingRows));

  const metaTop = document.createElement("div");
  metaTop.className = "meta-top";

  const idSpan = document.createElement("span");
  idSpan.className = "id";
  idSpan.textContent = "#" + row.id;

  const hashSpan = document.createElement("span");
  hashSpan.textContent = row.hash.slice(0, 8) + "…";

  metaTop.appendChild(idSpan);
  metaTop.appendChild(hashSpan);

  const scoresDiv = document.createElement("div");
  scoresDiv.className = "scores";

  const modelDiv = document.createElement("div");
  modelDiv.className = "score-pill model";
  modelDiv.textContent = "model: " + fmtScore(row.model_score);

  const userDiv = document.createElement("div");
  userDiv.className = "score-pill user";

  const tagDiv = document.createElement("div");
  tagDiv.innerHTML = row.tag_name;

  let userVal = row.user_score;
  if (pendingScores.has(row.id)) {
    userVal = pendingScores.get(row.id);
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

  let currentScore = pendingScores.has(row.id)
    ? pendingScores.get(row.id)
    : row.user_score;

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
      pendingScores.delete(row.id);
    } else {
      pendingScores.set(row.id, Number(v));
    }
    updatePendingCounter();
    renderGrid(trainingRows, gridIDName);
  });

  card.appendChild(thumbWrap);
  card.appendChild(metaTop);
  card.appendChild(scoresDiv);
  card.appendChild(select);
  card.appendChild(selWrap);
  card.appendChild(tagDiv);

  return card;
}

window.addEventListener("DOMContentLoaded", () => {
  const loadButton = findInRoot("tab-training", "#load-btn");
  loadButton.addEventListener("click", loadTrainingRows);
});
