import csv
import argparse
from pathlib import Path
from typing import List

def parse_args():
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "--main_table_path",
        type=str,
        required=True,
        help="Path to table for appending rows"
    )
    parser.add_argument(
        "--merge_table_path",
        type=str,
        required=True,
        help="Path to table to append rows from"
    )
    return parser.parse_args()

def read_table(path: Path) -> List:
    rows = []
    with path.open("r",encoding="UTF-8") as f:
        len = 0
        reader = csv.DictReader(f)
        for row in reader:
            rows.append(row)
    
    return rows

def main():
    args = parse_args()
    main_table_path = Path(args.main_table_path).resolve()
    merge_table_path = Path(args.merge_table_path).resolve()

    main_table = read_table(main_table_path)
    merge_table = read_table(merge_table_path)

    if len(merge_table) == 0:
        raise RuntimeError("merge table len is 0")
    
    main_keys = main_table[0].keys()
    merge_keys = merge_table[0].keys()
    for main_key in main_keys:
        isPresent: bool = False
        for merge_key in merge_keys:
            if main_key == merge_key:
                isPresent = True
                break
        if not isPresent:
            raise RuntimeError(f"keys of tables are not the same")

    duplicate_ids = []
    for merge_row in merge_table:
        for main_row in main_table:
            if merge_row["hash"] == main_row["hash"]:
                duplicate_ids.append(merge_row["id"])
                print(f"found duplicate id {merge_row['id']}, path: {merge_row["path"]}")
    
    unique_merge_rows = []

    for i,merge_row in enumerate(merge_table):
        isDouble: bool = False
        for id in duplicate_ids:
            if id == merge_row["id"]:
                isDouble = True
        if not isDouble:
            merge_row["id"] = len(main_table)+i
            unique_merge_rows.append(merge_row)
    
    main_table.extend(unique_merge_rows)

    with main_table_path.open("w",encoding="UTF-8") as f:
        writer = csv.DictWriter(f,fieldnames=main_keys)
        writer.writeheader()
        writer.writerows(main_table)

    print("done")
    
if __name__ == "__main__":
    main()