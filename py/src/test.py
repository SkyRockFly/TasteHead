import os
from pathlib import Path

def main():
    dirPath = Path(r"E:\AI\Embeddings Default City\ScraperSet\raw\e621")
    for dir,dirs,filenames in os.walk(dirPath):
        print(filenames)


if __name__ == "__main__":
    main()