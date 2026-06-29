import argparse
from pathlib import Path
import os
import csv
import hashlib
import base64
from PIL import Image

def parse_args():
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "--dir",
        type=str,
        required=True,
        help="Path to images"
    )
    return parser.parse_args()

def parse_hash(path: Path) -> str:
   h = hashlib.blake2b(digest_size=32)
   bytes = path.read_bytes()
   h.update(bytes)
   raw = h.digest()
   hash = base64.urlsafe_b64encode(raw).decode("ascii").rstrip("=")
   return hash

def main():
    args = parse_args()
    dirPath = Path(args.dir).resolve()
    print(f"Image dir: {dirPath}")

    if not dirPath.is_dir():
        print("Current path is not directory:")
        return
    

    for dir,_,filenames in os.walk(dirPath):
        pathdir = Path(dir)
        if (pathdir.name == "thumbs" or pathdir.name == "output" or pathdir.name == "hashes" or pathdir.name == "download" ):
            continue

        rows = []
        for _,fname in enumerate(filenames):
            row = {}
            old_path = pathdir / fname

            ext = old_path.suffix.lower()
            if ext not in [".jpg", ".jpeg", ".png", ".webp"]:
                continue

            hash = parse_hash(old_path)
            if hash is None:
                raise RuntimeError("cannot parse hash")
            row["hash"] = hash
            row["dir"] = dirPath.name
            row["path"] = old_path.relative_to(dirPath)
            rows.append(row)
        csvPath = pathdir / "output.csv"
        changedRows = []
        with csvPath.open("r",encoding="utf-8",newline="") as f:
            reader = csv.DictReader(f,fieldnames=["id","dir","path","thumb_path","hash","model_score","user_score"])
            for csvRow in reader:
                for row in rows:
                    if row["dir"] == csvRow["dir"] and str(row["path"]) == str(csvRow["path"]):
                        csvRow["hash"] = row["hash"]
                        break
                changedRows.append(csvRow)
        with csvPath.open("w",encoding="utf-8",newline="") as f:
            writer = csv.DictWriter(f,fieldnames=["id","dir","path","thumb_path","hash","model_score","user_score"])
            writer.writerows(changedRows)

    
    print("Done")

if __name__ == "__main__":
    main()