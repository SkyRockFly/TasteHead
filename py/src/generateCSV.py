import os
import csv
import re
from datetime import datetime
from pathlib import Path

ROOT_DIR = Path("raw")

OUTPUT_IMAGES_CSV = ROOT_DIR / "images.csv"
OUTPUT_LABELS_CSV = ROOT_DIR / "labels.csv"

PREFIX_TO_SCORE = {
    "000": 0.0,
    "025": 0.25,
    "050": 0.5,
    "075": 0.75,
    "100": 1.0,
}

VALID_EXTS = {
    ".jpg",
    ".jpeg",
    ".png",
    ".webp"
}

def parse_file(root_dir: Path, full_path: Path, next_id: int):
    rel_path = full_path.relative_to(root_dir)
    name = full_path.name
    stem = full_path.stem.lower()
    ext = full_path.suffix.lower()

    if ext not in VALID_EXTS:
        return None,None,None
    
    parts = stem.split("_")
    prefix = parts[0]

    is_anchor = "anchor" in rel_path.parts

    score = None
    anchor_type = None

    if is_anchor:
        if stem.startswith("yes"):
            score = 1.0
            anchor_type = "hard_yes"
        elif stem.startswith("hardno"):
            score = 0.0
            anchor_type = "hard_no"
        else:
            print(f"[WARN] invalid anchor prefix: {rel_path}")
            return None,None,None
        source = "anchor"
    else:
        m = re.match(r"^(000|025|050|075|100)$",prefix)
        if not m:
            print(f"[SKIP] unknown prefix: {rel_path}")
            return None,None,None
        score = PREFIX_TO_SCORE[m.group(1)]
        source = "pool"

    img_row = {
        "img_id": next_id,
        "path": str(rel_path).replace("\\","/"),
        "source": source,
    }
    
    label_row = {
        "img_id": next_id,
        "score": score,
        "ts": datetime.now().isoformat(timespec="seconds"),
        "session_id": "bootstrap_1",
        "anchor_type": anchor_type or "",
    }

    return img_row,label_row, anchor_type

def main():
    images_rows = []
    labels_rows = []
    anchors = []
    img_id = 1

    for dirpath, _ ,filenames in os.walk(ROOT_DIR):
        dirpath = Path(dirpath)
        for fname in filenames:
            full_path = dirpath / fname

            if full_path.name in {OUTPUT_IMAGES_CSV.name, OUTPUT_LABELS_CSV.name}:
                continue

            img_row, label_row, anchor_type = parse_file(ROOT_DIR,full_path,img_id)
            if img_row is None:
                continue

            images_rows.append(img_row)
            labels_rows.append(label_row)

            if anchor_type:
                anchors.append({"img_id": img_id, "type": anchor_type})

            img_id += 1

            # пишем images.csv
    with OUTPUT_IMAGES_CSV.open("w", newline="", encoding="utf-8") as f:
        writer = csv.DictWriter(f, fieldnames=["img_id", "path", "source"])
        writer.writeheader()
        for row in images_rows:
            writer.writerow(row)

    # пишем labels.csv
    with OUTPUT_LABELS_CSV.open("w", newline="", encoding="utf-8") as f:
        writer = csv.DictWriter(
            f, fieldnames=["img_id", "score", "ts", "session_id", "anchor_type"]
        )
        writer.writeheader()
        for row in labels_rows:
            writer.writerow(row)

    print(f"Готово. Картинок: {len(images_rows)}")
    print(f"images.csv: {OUTPUT_IMAGES_CSV}")
    print(f"labels.csv: {OUTPUT_LABELS_CSV}")
    if anchors:
        print(f"Якорей найдено: {len(anchors)} (hard_yes/hard_no через anchor_type в labels.csv)")


if __name__ == "__main__":
    main()