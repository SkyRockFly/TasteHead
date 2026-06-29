// init.js

window.addEventListener("DOMContentLoaded", () => {
  // навигация по страницам
  findInRoot("tab-download", "#load-btn").addEventListener("click", () =>
    loadPage(true),
  );

  findInRoot("tab-download", "#prev-btn").addEventListener("click", () => {
    const newCursor = Math.max(0, cursor - 2 * limit);
    findInRoot("tab-download", "#cursor-input").value = newCursor;
    loadPage(false);
  });

  findInRoot("tab-training", "#prev-btn").addEventListener("click", () => {
    const newCursor = Math.max(0, cursor - 2 * limit);
    findInRoot("tab-training", "#cursor-input").value = newCursor;
    loadTrainingRows(false);
  });

  findInRoot("tab-download", "#next-btn").addEventListener("click", () => {
    if (!hasMore) {
      setStatus("page-status", "has_more=false, дальше пусто.", false);
      return;
    }
    loadPage(false);
  });

  findInRoot("tab-training", "#next-btn").addEventListener("click", () => {
    if (!hasMore) {
      setStatus("page-status", "has_more=false, дальше пусто.", false);
      return;
    }
    loadTrainingRows(false);
  });

  // сохранение / сброс оценок
  findInRoot("tab-training", "#save-all-btn").addEventListener(
    "click",
    saveAllScores,
  );
  findInRoot("tab-download", "#save-all-btn").addEventListener(
    "click",
    saveAllScores,
  );

  findInRoot("tab-training", "#reset-local-btn").addEventListener(
    "click",
    resetLocalChanges,
  );
  findInRoot("tab-download", "#reset-local-btn").addEventListener(
    "click",
    saveAllScores,
  );

  // скачивание батча
  document.getElementById("dl-btn").addEventListener("click", downloadBatch);

  // модалка
  document
    .getElementById("modal-close-btn")
    .addEventListener("click", closeModal);
  document
    .getElementById("modal-apply-btn")
    .addEventListener("click", applyModalScore);

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

  // батчи
  document
    .getElementById("refresh-batches-btn")
    .addEventListener("click", loadBatches);
  document
    .getElementById("load-local-batches")
    .addEventListener("click", importLocalDirs);

  // дубликаты
  const dupsBtn = document.getElementById("load-dups-btn");
  if (dupsBtn) {
    dupsBtn.addEventListener("click", () => {
      loadDuplicates().catch((err) => {
        console.error("loadDuplicates:", err);
        alert("Не смог загрузить дубликаты, смотри консоль.");
      });
    });
  }
  document
    .getElementById("delete-dups-btn")
    .addEventListener("click", deleteDuplicates);

  // выбор на странице
  document.getElementById("select-all-btn").addEventListener("click", () => {
    for (const img of images) {
      selectedIDs.add(img.id);
    }
    renderGrid(images, gridIDName, makeDownloadCard);
  });

  findInRoot("tab-download", "#clear-selection-btn").addEventListener(
    "click",
    () => {
      selectedIDs.clear();
      renderGrid(images, gridIDName, makeDownloadCard);
    },
  );

  // теги

  const tagSelectDownload = findInRoot("tab-download", ".tag-select");
  const tagSelectTraining = findInRoot("tab-training", ".tag-select");
  if (tagSelectDownload) {
    tagSelectDownload.addEventListener("change", async (e) => {
      const val = e.target.value;
      if (val === "__new__") {
        await createNewTag();
      }
    });
  }

  if (tagSelectTraining) {
    tagSelectTraining.addEventListener("change", async (e) => {
      const val = e.target.value;
      if (val === "__new__") {
        await createNewTag();
      }
    });
  }

  // стрелки в модалке
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

  // старт
  updatePendingCounter();
  loadBatches();
  loadTags().catch(() => {});
});
