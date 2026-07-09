import csv
import json
from pathlib import Path
import numpy as np
import torch
from PIL import Image
import open_clip
import faiss
import traceback
from typing import List
import argparse
from dataclasses import dataclass
from typing import Optional
import faulthandler
import os
faulthandler.enable()

@dataclass
class ImageRow:
    id: int
    rel_path: Path
    dir_name: Path
    score: Optional[float]

@dataclass
class ImageTable:
    ids: list[int]
    paths: list[Path]
    scores: list[Optional[float]]  # None = no score

MODEL_NAME = "ViT-H-14"
raw_pretrained = os.getenv("OPENCLIP_PRETRAINED", "laion2b_s32b_b79k")
if raw_pretrained == "" or raw_pretrained.lower() in ("none", "null", "random"):
    PRETRAINED = None
else:
    PRETRAINED = raw_pretrained
VectorInfo = List[List[float]]



@dataclass
class TrainingTable:
    ids: list[int]
    paths: list[Path]
    scores: list[float]

@dataclass
class BaseTable:
    ids: list[int]
    paths: list[Path]

def resolve_device() -> str:
    mode = os.getenv("TASTEHEAD_DEVICE", "auto").lower()

    if mode == "cpu":
        return "cpu"

    if mode == "cuda":
        if not torch.cuda.is_available():
            raise RuntimeError("TASTEHEAD_DEVICE=cuda, but CUDA is not available")
        return "cuda"

    if mode == "auto":
        return "cuda" if torch.cuda.is_available() else "cpu"

    raise RuntimeError(f"unknown TASTEHEAD_DEVICE: {mode}")

def env_int(name: str, default: int = 0) -> int:
    raw = os.getenv(name, str(default)).strip()
    try:
        return int(raw)
    except ValueError as e:
        raise RuntimeError(f"invalid int env {name}={raw!r}") from e

DEVICE = resolve_device()
BATCH_SIZE = env_int("TASTEHEAD_CLIP_BATCH_SIZE")
if BATCH_SIZE <= 0:
    raise RuntimeError("TASTEHEAD_EPOCHS must be > 0")

def read_image_table(img_path: Path) -> list[ImageRow]:
    rows: list[ImageRow] = []

    with img_path.open("r", encoding="utf-8") as f:
        reader = csv.DictReader(f)
        for row in reader:
            raw = row.get("user_score", "")
            score = None if raw == "" else float(raw)
            rows.append(
                ImageRow(
                    id=int(row["id"]),
                    rel_path=Path(row["path"]),
                    dir_name=Path(row["dir"]),
                    score= score
                )
            )
    return rows

def getEvalInfo(rows: list[ImageRow], img_dir: Path) -> BaseTable:
    base_dir = img_dir.parent
    eval_ids: list[int] = []
    eval_paths: list[Path] = []

    for row in rows:
        if row.dir_name.name != base_dir.name:
             raise RuntimeError(
                f"batch mismatch: csv dir={row.dir_name.name}, actual dir={base_dir.name}"
            )

        eval_ids.append(row.id)
        eval_paths.append(base_dir / row.rel_path)

    return BaseTable(eval_ids, eval_paths)

def getTrainInfo(rows: list[ImageRow], dataPath: Path) -> TrainingTable:
    train_ids: list[int] = []
    train_paths: list[Path] = []
    train_scores: list[float] = []

    for row in rows:
        if row.score is None:
            raise RuntimeError(
                f"no score in row with id={row.id}, path={row.rel_path}"
            )
        
        train_ids.append(row.id)
        train_paths.append(dataPath / row.dir_name / row.rel_path)
        train_scores.append(row.score)

    train = TrainingTable(train_ids, train_paths, train_scores)
    return train


def parse_args():
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "--csvPath",
        type=str,
        required=True,
        help="Path to table with images' paths"
    )
    parser.add_argument(
        "--dataPath",
        type=str,
        required=False,
        help="Path to dir with images(for training)"
    )
    parser.add_argument(
    "--train_mode",
    action="store_true",
    help="Run in train mode"
)
    parser.add_argument(
    "--faiss",
    action="store_true",
    help="Add faiss index"
)
    
    return parser.parse_args()

def make_embeddings(
    model: torch.nn.Module,
    preprocess,
    table: BaseTable,
) -> tuple[np.ndarray,List[int]]:
    all_vects: list[np.ndarray] = []
    out_ids: list[int] = []

    n = len(table.paths)
    if n == 0:
        print("No images,skip")
        return None, None
    print(f"Found {n} images")

    with torch.inference_mode():
        for start in range(0,n,BATCH_SIZE):
            end = min(start + BATCH_SIZE,n)

            batch_paths = table.paths[start:end]
            batch_ids   = table.ids[start:end]

            batch_tensors = []
            batch_kept_ids = []

            for path, img_id in zip(batch_paths, batch_ids):
                img_path = Path(path)
                
                try:
                    img = Image.open(img_path).convert("RGB")
                except Exception as e:
                    print(f"[ERR] Can't open {img_path}: {e}")
                    continue

                try:
                    tensor = preprocess(img)
                except Exception as e:
                    print(f"[ERR] preprocess failed: {type(e).__name__}: {e}", flush=True)
                    raise
                batch_tensors.append(tensor)
                batch_kept_ids.append(img_id)
                

            if not batch_tensors:
                print(f"[{start}:{end}] no valid images in this batch, skip")
                continue
            
            batch_tensor = torch.stack(batch_tensors)
            batch_tensor = batch_tensor.to(DEVICE)
            if DEVICE == "cuda":
                with torch.autocast("cuda"):
                    emb = model.encode_image(batch_tensor)
            elif DEVICE == "cpu":
                emb = model.encode_image(batch_tensor)
            else:
                raise RuntimeError(f"NO DEVICE:{DEVICE}")

            emb = emb / emb.norm(dim=-1,keepdim=True)
            emb_np = emb.cpu().numpy().astype("float32")
            print(f"Batch {start}:{end} -> tensor {batch_tensor.shape}, emb {emb_np.shape}")
            
            for j,idx in enumerate(batch_kept_ids):
                all_vects.append(emb_np[j])
                out_ids.append(idx)
        
    if not all_vects:
        raise RuntimeError("no valid embeddings")
    
    matrix = np.stack(all_vects, axis=0)
    return matrix, out_ids

def main():
    args = parse_args()
    csvImagePath = Path(args.csvPath).resolve()
    isTrainMode =  args.train_mode
    if (isTrainMode):
        dataPath = Path(args.dataPath).resolve()
    isFaiss = args.faiss

    if (isTrainMode):
        print(f"Datapath  {dataPath}")

    if not csvImagePath.is_file():
        print("Current path is not a file:")
        return
    
    print("DEVICE:",DEVICE)
    print("Load Model:",MODEL_NAME,PRETRAINED)

    imageList = read_image_table(csvImagePath)
    
    if (isTrainMode):
        trainTable = getTrainInfo(imageList,dataPath)
    else:
        evalTable = getEvalInfo(imageList,csvImagePath)
   
    try:
        model, _, preprocess = open_clip.create_model_and_transforms(
        MODEL_NAME,
        pretrained=PRETRAINED,
        device=DEVICE
    )
    except Exception as e:
        print(f"Load model: {e}")
        return
    
    model.to(DEVICE)
    model.eval()

    if (isTrainMode):
        training_matrix, training_ids = make_embeddings(
        model,
        preprocess,
        trainTable,
    )
    else:
        eval_matrix, eval_ids = make_embeddings(
        model,
        preprocess,
        evalTable
    )

    if (isTrainMode):
        idToScore: dict[int, float] = {}
        for row in imageList:
            if row.score is None:
                raise RuntimeError(f"missing score for image id {row.id}")
            idToScore[row.id] = row.score
    
    output_dir = csvImagePath.parent / "output"
    output_dir.mkdir(exist_ok=True)
    embed_dim = None
    overall = None
    
    if (isTrainMode):
        np.save(output_dir / "clip_training_vecs.npy", training_matrix)
        print("Training embeddings saved in", output_dir / "clip_training_vecs.npy")
        vector_info = []
        for id in training_ids:
            score = idToScore[id]
            vector_info.append([float(id), float(score)])
        vector_info_np = np.array(vector_info, dtype="float32")
        print("Training embeddings scores saved in", output_dir / "vect_training_info.npy")
        np.save(output_dir / "vect_training_info.npy",vector_info_np)
        embed_dim = training_matrix.shape[1]
        overall = training_matrix.shape
    else:
        np.save(output_dir / "clip_eval_vecs.npy", eval_matrix)
        print("Training embeddings saved in", output_dir / "clip_eval_vecs.npy")

        vector_info = []
        for id in eval_ids:
            vector_info.append(id)
        vector_info_np = np.array(vector_info, dtype="float32")
        print("Eval embeddings ids saved in", output_dir / "eval_ids_info.npy")
        np.save(output_dir / "eval_ids_info.npy",vector_info_np) 
        embed_dim = eval_matrix.shape[1]
        overall = eval_matrix.shape

    if isFaiss:
        d = training_matrix.shape[1]
        index = faiss.IndexFlatIP(d)
        index.add(training_matrix)
        faiss.write_index(index, str(output_dir/ "clip.index.faiss"))
        print("faiss-index saved at", output_dir / "clip.index.faiss")

    # конфиг модели
    config = {
        "model_name": MODEL_NAME,
        "pretrained": PRETRAINED,
        "embed_dim": embed_dim,
        "device": DEVICE,
        "batch_size": BATCH_SIZE,
    }

    with (output_dir/ "config.json").open("w", encoding="utf-8") as f:
        json.dump(config, f, ensure_ascii=False, indent=2)

    print("Done. Lines overall:", overall )  



if __name__ == "__main__":
    try:
        main()
    except Exception:
        print("\n=== UNCAUGHT ERROR ===")
        traceback.print_exc()
