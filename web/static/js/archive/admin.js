// admin.js

// ==== Дубликаты ====

function buildDubThumbUrl(fileRef) {
  const rel = normalisePath(fileRef.dir + "/thumbs/" + fileRef.rel_path);
  return buildApiUrl("/images/get/pic?path=" + encodeURIComponent(rel));
}

async function loadDuplicates() {
  const resp = await fetch(buildApiUrl("/images/duplicates"));
  if (!resp.ok) {
    console.error("loadDuplicates HTTP", resp.status);
    return;
  }
  const data = await resp.json();
  groups = data.groups || data || [];
  renderDuplicates(groups);
}

function renderDuplicates(groupsArr) {
  const section = document.getElementById("dups-section");
  const container = document.getElementById("dups-container");
  const summary = document.getElementById("dups-summary");

  if (!section || !container || !summary) {
    console.warn("dup-section elements not found");
    return;
  }

  container.innerHTML = "";

  if (!groupsArr.length) {
    summary.textContent = "Дубликатов не найдено.";
    section.style.display = "block";
    return;
  }

  let totalFiles = 0;
  for (const g of groupsArr) {
    totalFiles += 1 + (g.duplicates ? g.duplicates.length : 0);
  }

  summary.textContent = `Групп: ${groupsArr.length}, файлов в группах: ${totalFiles}`;

  for (const group of groupsArr) {
    const groupEl = document.createElement("div");
    groupEl.className = "dup-group";

    const header = document.createElement("div");
    header.className = "dup-header";
    const count = 1 + (group.duplicates ? group.duplicates.length : 0);
    header.innerHTML = `
      <span>hash: <code>${group.hash}</code></span>
      <span>${count} шт.</span>
    `;
    groupEl.appendChild(header);

    const imgsWrap = document.createElement("div");
    imgsWrap.className = "dup-images";

    const allFiles = [group.first_seen, ...(group.duplicates || [])];

    allFiles.forEach((fileRef, idx) => {
      const imgBox = document.createElement("div");
      imgBox.className = "dup-img";

      const imgEl = document.createElement("img");
      imgEl.src = buildDubThumbUrl(fileRef);
      imgEl.alt = `dup ${idx}`;
      imgBox.appendChild(imgEl);

      const cap = document.createElement("div");
      cap.className = "dup-caption";

      const label = idx === 0 ? "[ориг]" : "[dup]";
      const dir = fileRef.dir || fileRef.batch_name || "";
      const path = fileRef.rel_path || "";
      cap.textContent = `${label} ${dir}/${path}`;
      imgBox.appendChild(cap);

      imgsWrap.appendChild(imgBox);
    });

    groupEl.appendChild(imgsWrap);
    container.appendChild(groupEl);
  }

  section.style.display = "block";
}

async function deleteDuplicates() {
  const files = [];

  for (const group of groups) {
    if (!group.duplicates) continue;
    for (const dup of group.duplicates) {
      files.push({
        hash: dup.hash,
        dir: dup.dir,
        rel_path: dup.rel_path,
      });
    }
  }

  if (files.length === 0) {
    console.warn("No files to delete");
    return;
  }

  const payload = { files };
  const data = await fetchJson("/duplicates/delete", payload, "DELETE");

  // предполагаю ответ вида { has_errors: bool, errors: "..." }
  if (data.has_errors) {
    console.error("delete duplicates:", data.errors);
    return;
  }

  groups = [];
  renderDuplicates(groups);
}

// ==== Теги ====

async function loadTags() {
  const resp = await fetch(buildApiUrl("/tag/list"));
  if (!resp.ok) {
    console.error("tags list HTTP", resp.status);
    return;
  }
  const data = await resp.json();
  tags = data.tags || data || [];

  const selects = document.querySelectorAll(".tag-select");
  if (!selects.length) return;
  for (const select of selects) {
    select.innerHTML = "";
    const allowNew = select.dataset.allowNew === "true";

    const placeholder = document.createElement("option");
    placeholder.value = "";
    placeholder.textContent = "- выбери тег -";
    select.appendChild(placeholder);

    for (const t of tags) {
      const opt = document.createElement("option");
      opt.value = t.id;
      opt.textContent = t.name;
      select.appendChild(opt);
    }
    if (allowNew) {
      const newOpt = document.createElement("option");
      newOpt.value = "__new__";
      newOpt.textContent = "Добавить новый тег";
      select.appendChild(newOpt);
    }
  }
}

async function createNewTag() {
  const name = prompt("New tag's name:");
  if (!name || !name.trim()) {
    await loadTags();
    return;
  }

  const desc = prompt("tag's desc (может быть пустым):") || "";

  const payload = {
    name: name.trim(),
    desc: desc.trim(),
  };

  // тут уже JSON, а не Response
  const created = await fetchJson("/tag/create", payload, "POST");
  console.log("created tag:", created);

  await loadTags();
}

// ==== Батчи / импорт / скачивание ====

async function importLocalDirs() {
  try {
    const resp = await fetch(buildApiUrl("/import/dirs"));
    if (!resp.ok) {
      console.error("importLocalDirs HTTP", resp.status);
      return;
    }
    console.log("importLocalDirs done");
  } catch (err) {
    console.error("importLocalDirs error", err);
  }
}

async function loadBatches() {
  try {
    const resp = await fetch(buildApiUrl("/downloads"));
    if (!resp.ok) {
      console.error("loadBatches HTTP", resp.status);
      return;
    }

    const data = await resp.json();
    const select = document.getElementById("batch-select");
    if (!select) return;

    select.innerHTML = "";

    batches = data.batches || [];

    const optAll = document.createElement("option");
    optAll.value = "";
    optAll.textContent = "Все батчи";
    select.appendChild(optAll);

    for (const batch of batches) {
      const opt = document.createElement("option");
      opt.value = batch.id;
      opt.textContent = batch.name;
      select.appendChild(opt);
    }
  } catch (err) {
    console.error("loadBatches error", err);
  }
}

async function downloadBatch() {
  const url = document.getElementById("dl-url").value.trim();
  const postSel = document.getElementById("dl-post-selector").value.trim();
  const imgAttr = document.getElementById("dl-image-attr").value.trim();
  const nextSel = document.getElementById("dl-next-selector").value.trim();
  const nextAttr = document.getElementById("dl-next-attr").value.trim();
  const pages = Number(document.getElementById("dl-pages").value) || 1;
  const dlLimit = Number(document.getElementById("dl-limit").value) || 24;

  if (!url) {
    setStatus("dl-status", "Нужен URL.", true);
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
    setStatus("dl-status", "Качаю и гоняю через CLIP/голову…", false);

    const data = await fetchJson("/images/download", payload, "POST");

    let msg = `Скачано и оценено: ${(data.images || []).length}`;
    if (data.dir) {
      msg += `\ndir: ${data.dir}`;
    }
    if (data.errs && data.errs.length) {
      msg += `\nОшибки:\n${data.errs}`;
    }

    setStatus("dl-status", msg, false);

    // после скачивания батчей — обновим список
    await loadBatches();
  } catch (e) {
    setStatus("dl-status", String(e), true);
  }
}

async function removeTag() {
  const tagInput = document.getElementById("tag-select");

  if (!select) {
    console.error("tag-select not found in DOM");
    return;
  }

  const id = Number(select.value);

  if (Number.isNaN(id)) {
    console.error("tag id is not a number:", id);
    return;
  }
  payload = {
    id: tagInput,
  };
  try {
    const data = await fetchJson("/tag/delete", payload, "DELETE");

    if (!data.accepted) {
      console.error("removeTag returned error");
      return;
    }
  } catch (err) {
    console.error("removeTag error:", err);
  }
}

async function sendToDataset() {
  const tagInput = findInRoot("tab-download", ".tag-select");
  const tagID = readInt(tagInput);

  if (!tagID) {
    console.error("tag is empty");
    return;
  }

  const imageIDs = Array.from(selectedIDs);

  if (imageIDs.length === 0) {
    console.warn("no images selected");
    return;
  }

  payload = {
    tag_id: tagID,
    image_ids: imageIDs,
  };
  try {
    const data = await fetchJson("/training/create", payload, "POST");

    if (!data.duplicates.length !== 0) {
      console.warn(data.duplicates);
      return;
    }
    selectedIDs;
  } catch (err) {
    console.error("createTrainingRows error:", err);
  }
}

window.addEventListener("DOMContentLoaded", () => {
  document
    .getElementById("send-training-btn")
    .addEventListener("click", sendToDataset);
});
