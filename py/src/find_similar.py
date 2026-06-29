from pathlib import Path
import csv
import numpy as np
import faiss

CLIP_PATH = Path(__file__).resolve().parent.parent / "clip"

def main():
    img_ids = np.load(CLIP_PATH / "img_ids.npy")
    img_vects = np.load(CLIP_PATH / "clip_vecs.npy")

    index = faiss.read_index(str(CLIP_PATH / "clip.index.faiss"))
    query_vect = img_vects[64:65]
    D,I = index.search(query_vect, k=10)
    print(D)
    print(I)

    for i,cos in enumerate(D[0]):
        if cos <= 0.97:
            stopIndex = i
            break

    print(D[0][0:stopIndex])
    print(I[0][0:stopIndex])

    

if __name__ == "__main__":
   main()