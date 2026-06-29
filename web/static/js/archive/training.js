async function TrainModel() {
  const modelNameInput = findInRoot(
    "tab-training",
    '[data-action="model-name-input"]',
  );
  const tagIDInput = findInRoot("tab-training", '[data-action="tag-id-input"');

  const tagID = readInt(tagIDInput);
  const modelName = String(modelNameInput.value).trim;

  payload = {
    model_name: modelName,
    tag_id: tagID,
  };

  const data = await sendJson("/model/train", payload, "POST");
  if (!data.accepted) {
    throw Error("training was interrupted");
  }
}

window.addEventListener("DOMContentLoaded", () => {
  findInRoot("tab-training", 'data-action=["train-model"]').addEventListener(
    "click",
    TrainModel,
  );
});
