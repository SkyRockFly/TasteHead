import argparse
from pathlib import Path
import os
import csv
import re
import hashlib
import base64
import sys
import shutil
from PIL import Image,UnidentifiedImageError

MAX_SIZE_IMAGE = 320
MAX_ALLOWED_PIXELS = 120_000_000

class SkippedImageError(Exception):
    pass

PREFIX_TO_SCORE = {
    "000": 0.0,
    "025": 0.25,
    "050": 0.5,
    "075": 0.75,
    "100": 1.0,
}

def parse_args():
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "--dir",
        type=str,
        required=True,
        help="Path to images"
    )
    parser.add_argument(
        "--parse_scores",
        action="store_true",
        help="Read score from image file name"
    )
    return parser.parse_args()

def parse_hash(path: Path) -> str:
   h = hashlib.blake2b(digest_size=32)
   bytes = path.read_bytes()
   h.update(bytes)
   raw = h.digest()
   hash = base64.urlsafe_b64encode(raw).decode("ascii").rstrip("=")
   return hash

def parse_score_file(path: Path):
    stem = path.stem.lower()
    parts = stem.split("_")
    prefix = parts[0]

    m = re.match(r"^(000|025|050|075|100)$",prefix)
    if not m:
        print(f"[SKIP] unknown prefix: {path}")
        return None
    score = PREFIX_TO_SCORE[m.group(1)]
    return score

def file_sha256(path: Path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def make_thumb(src: Path, thumb_dir: Path, hash: str) -> Path:
    filename = src.name
    ext = src.suffix.lower()
    new_name = f"{hash}{ext}"
    dst = thumb_dir / new_name
    try:
        with Image.open(src) as img:
            w, h = img.size
            pixels = w * h

            if pixels > MAX_ALLOWED_PIXELS:
                moved_to = move_to_exceptional(src,"too_large")
                raise SkippedImageError(f"TOO LARGE IMAGE: moved {src} ->{moved_to}, {w}*{h}={pixels}")
            
            img = img.convert("RGB")
            img.thumbnail((MAX_SIZE_IMAGE,MAX_SIZE_IMAGE), resample=Image.Resampling.LANCZOS )
            img.save(dst)
            return dst

    except Image.DecompressionBombError as e:
        moved_to = move_to_exceptional(src,"too_large")
        raise SkippedImageError(f"TOO LARGE IMAGE: moved {src} ->{moved_to}: {e}")
    

def move_to_exceptional(imgPath: Path, reason: str) -> Path:
    dstDir = imgPath.parent / "exceptional" / reason 
    dstDir.mkdir(parents=True,exist_ok=True)

    dstPath = dstDir / imgPath.name

    shutil.move(imgPath,dstPath)
    return dstPath


def main():
    args = parse_args()
    dirPath = Path(args.dir).resolve()
    parse_scores = args.parse_scores
    print(f"Image dir: {dirPath}")

    if not dirPath.is_dir():
        print("Current path is not directory:")
        return
    
    rows = []
    THUMB_DIR = dirPath / "thumbs"
    THUMB_DIR.mkdir(parents=True,exist_ok=True)

    counter = 0

    for dir,_,filenames in os.walk(dirPath):
        pathdir = Path(dir)

        if (pathdir.name == "thumbs" or pathdir.name == "exceptional"):
            continue
          
        for fname in sorted(filenames):
            row = {}
            old_path = pathdir / fname

            ext = old_path.suffix.lower()
            if ext not in [".jpg", ".jpeg", ".png", ".webp"]:
                continue

            hash = parse_hash(old_path)
            if hash is None:
                print(f"[SKIPPED] Cannot load hash for filename:{fname}")
                continue
            row["hash"] = hash
            
            try:
                thumb_path = make_thumb(old_path,THUMB_DIR,hash)
            except SkippedImageError as e:
                print(f"[SKIPPED] {e}", file=sys.stderr)
                continue
            except (UnidentifiedImageError, OSError ) as e:
                print(f"[SKIPPED] Cannot make thumb for filename:{fname}, error: {e}")
                continue
            
            new_name = f"{hash}{ext}"
            new_path = pathdir / new_name

            if old_path != new_path:
                try:
                    old_path.rename(new_path)
                except FileExistsError as e:
                    if old_path.stat().st_size != new_path.stat().st_size:
                         raise RuntimeError(
                            "NAME CONFLICT: target exists and size differs\n"
                            f"old_path={old_path}\n"
                            f"new_path={new_path}\n"
                            f"old_size={old_path.stat().st_size}\n"
                            f"new_size={new_path.stat().st_size}"
                        ) from e
                    
                    if file_sha256(old_path) == file_sha256(new_path):
                         print(f"DUPLICATE: remove {old_path}, already exists as {new_path}",file=sys.stderr,)
                         old_path.unlink()
                         continue

                    raise RuntimeError(
                        "NAME COLLISION: target exists but content differs\n"
                        f"old_path={old_path}\n"
                        f"new_path={new_path}\n"
                        f"old_size={old_path.stat().st_size}\n"
                        f"new_size={new_path.stat().st_size}\n"
                        f"old_sha256={file_sha256(old_path)}\n"
                        f"new_sha256={file_sha256(new_path)}"
                    ) from e
        
            row["id"]= counter
            row["dir"] = dirPath.name
            row["path"] = new_path.relative_to(dirPath)
            row["thumb_path"] = thumb_path.relative_to(dirPath)
            rows.append(row)
            counter +=1

    OUTPUT_CSV = dirPath / "output.csv"
    with OUTPUT_CSV.open("w",encoding="utf-8",newline="") as f:
        writer = csv.DictWriter(f,fieldnames=["id","dir","path","thumb_path","hash","model_score","user_score"])
        writer.writeheader()
        for row in rows:
            writer.writerow(row)
    
    print("Done")
        


if __name__ == "__main__":
    main()
