import torch
from torch import nn
import csv
from pathlib import Path
import numpy as np
from typing import List
import argparse
import faulthandler
import os
faulthandler.enable()


Scores = list[float]
Images = list[list[any]]
Target_logits = List[List[int]]

CLIP_PATH = Path(__file__).resolve().parent.parent / "clip"
criterion = nn.BCEWithLogitsLoss()

BATCH_SIZE = 32
EPOCHS = 140
LEARNING_RATE = 1e-3



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

def parse_args():
    parser = argparse.ArgumentParser()
    parser.add_argument(
    "--learn_mode",
    action="store_true",
    help="Teach model, otherwise eval mode"
)
    parser.add_argument(
        "--score_path",
        type=str,
        required=False,
        help="Score path for teaching (npy format)"
    )
    parser.add_argument(
        "--vect_path",
        type=str,
        required=True,
        help="Vectors info (npy format)"
    )
    parser.add_argument(
        "--model_path",
        type=str,
        required=False,
        help="Model path for evaluating"
    )
    parser.add_argument(
        "--output",
        type=str,
        required=False,
        help="Output for model (required)"
    )

    
    return parser.parse_args()


class TasteHead(nn.Module):
    def __init__(self, in_dim: int = 1024, hidden_dim: int = 128, out_dim: int = 4):
        super().__init__()
        self.net = nn.Sequential(
            nn.Linear(in_dim,hidden_dim),
            nn.ReLU(),
            nn.Linear(hidden_dim,out_dim),
        )

    def forward(self, x: torch.Tensor) -> torch.Tensor:
        # x: [batch_size, in_dim]
        logits = self.net(x)  # [batch_size, out_dim]
        return logits
    
def compute_score(logits: torch.Tensor) -> Scores:
    scores: Scores = []
    for i,logit in enumerate(logits):
        total_score = 0
        probs = torch.sigmoid(logit)
        print(probs)
        for prob in probs:
            if prob > 0.5:
                total_score += 0.25

    print(total_score)
    scores.append(total_score)
    print(scores)

    return scores

  

def compute_score_targets(targets_path: Path) -> torch.Tensor:
    vect_info = np.load(targets_path)
    scores = []
    for vector in vect_info:
        score = float(vector[1])
        scores.append(score)
    
    target_logits : Target_logits = []
    for score in (scores):
        logit_score = int(round(score * 4))
        target_logit = [0,0,0,0]

        for i in range(logit_score):
            target_logit[i] = 1
        target_logits.append(target_logit)    
        
    return torch.tensor(target_logits, dtype=torch.float32)

def train_model(head: TasteHead, tensor_vects : torch.Tensor,targets: torch.Tensor, output:Path):
    output_dir = output.parent
    log_path = output_dir / "train_debug.log"
    with log_path.open("w", encoding="utf-8") as log:
        optimizer = torch.optim.Adam(head.parameters(),LEARNING_RATE)

        if len(targets) != len(tensor_vects):
            print(f"lens targets and vects are not equal:targets - {len(targets)}, vects - {len(tensor_vects)}")
            return
        
        head.train()

        size = len(tensor_vects)

        for epoch in range(0,EPOCHS,1):
            head.train()
            total_loss = 0.0
            steps = 0
            last_loss = 0.0
            last_logits = None
            last_scores = None

            for start in range (0,size,BATCH_SIZE):
                end = min(start+BATCH_SIZE, size)

                batch_vects = tensor_vects[start:end]
                batch_scores = targets[start:end]

                optimizer.zero_grad()

                logits = head(batch_vects)

                if epoch == 0 and start == 0:
                    log.write(f"first batch logits shape: {tuple(logits.shape)}\n")
                    log.write(f"first batch scores shape before: {tuple(batch_scores.shape)}\n")

                batch_scores = batch_scores.view_as(logits)

                if epoch == 0 and start == 0:
                    log.write(f"first batch scores shape after: {tuple(batch_scores.shape)}\n\n")

                loss = criterion(logits,batch_scores)
                loss.backward()

                optimizer.step()

                last_loss = loss.item()
                total_loss += loss.item()
                steps += 1

                last_logits = logits.detach()
                last_scores = batch_scores.detach()

            avg_loss = total_loss / max(steps,1)
            print(f"avg_loss: {avg_loss: .4f}, sum loss {total_loss: .4f}, current loss {last_loss: .4f} on epoch: {epoch+1}")
            with torch.no_grad():
                preds = torch.sigmoid(last_logits)

                log.write(
                    f"epoch {epoch + 1}: "
                    f"avg_loss={avg_loss:.4f}, "
                    f"last_batch_loss={last_loss:.4f}, "
                    f"sum_loss={total_loss:.4f}, "
                    f"steps={steps}\n"
                )

                log.write(
                    f"  logits: "
                    f"min={last_logits.min().item():.4f}, "
                    f"max={last_logits.max().item():.4f}, "
                    f"mean={last_logits.mean().item():.4f}\n"
                )

                log.write(
                    f"  preds:  "
                    f"min={preds.min().item():.4f}, "
                    f"max={preds.max().item():.4f}, "
                    f"mean={preds.mean().item():.4f}\n"
                )

                log.write(
                    f"  target: "
                    f"min={last_scores.min().item():.4f}, "
                    f"max={last_scores.max().item():.4f}, "
                    f"mean={last_scores.mean().item():.4f}\n\n"
                )

                log.flush()
        
        checkpoint = {
        "model_state": head.state_dict(),
        "optimizer_state": optimizer.state_dict(),
        "epoch": epoch,
        }
        chkpointName = output.stem + "_ckpt.pt"
        chkpointDir = output_dir / "checkpoints" 
        chkpointDir.mkdir(parents=True,exist_ok=True)
        torch.save(checkpoint, chkpointDir / chkpointName )

        torch.save(head.state_dict(), output)
        print("Saved head to", output)



def main():
    args = parse_args()
    learn_mode = args.learn_mode
    if learn_mode:
        score_path = args.score_path
        if score_path is None:
            raise RuntimeError("no scores for learning")
        score_path = Path(score_path).resolve()
        output_path = Path(args.output).resolve()

    vect_path = Path(args.vect_path).resolve()
    model_path = args.model_path
    
    if model_path is not None:
        model_path = Path(model_path).resolve()
        print("MODEL NAME:",model_path)

    DEVICE = resolve_device()

    vects = np.load(vect_path)
    if learn_mode:
        tensor_targets = compute_score_targets(score_path).to(DEVICE)

    tensor_vects = torch.from_numpy(vects).to(DEVICE)

    taste_head = TasteHead(in_dim=1024,hidden_dim=128,out_dim=4).to(DEVICE)

    if learn_mode:
        train_model(taste_head,tensor_vects,tensor_targets,output_path)
        print("Done training")
        return

    state = torch.load(model_path,map_location=DEVICE)
    taste_head.load_state_dict(state)

    taste_head.eval()
    with torch.no_grad():
       logits = taste_head(tensor_vects)
       sigmoids = torch.sigmoid(logits)

       levels = (sigmoids >= 0.5).sum(dim=1)
       pred_score = (levels.float() / 4.0) 

    imgs = []
    IMG_PATH = vect_path.parent.parent
    OUTPUT_CSV = IMG_PATH / "output.csv"
    with OUTPUT_CSV.open("r",encoding="UTF-8") as f:
        reader = csv.DictReader(f)
        for row in reader:
            imgs.append(row)

    proccesed_ids = np.load(vect_path.parent / "eval_ids_info.npy")
    scores = pred_score.detach().cpu().tolist()
    for idx, s in zip(proccesed_ids,scores):
        row = imgs[int(idx)]
        row["model_score"] = f"{s:.2f}"
        
    with OUTPUT_CSV.open("w",encoding="UTF-8",newline="") as f:
        writer = csv.DictWriter(f,fieldnames=["id","dir","path","thumb_path","hash","model_score","user_score"])
        writer.writeheader()
        for row in imgs:
            writer.writerow(row)
    
    
    #vect_info = np.load(CLIP_PATH / "vect_info.npy")
    #true_scores = vect_info[:, 1]

    #print("All pics")

    #for i in range(0,len(vect_info),1):
    #    print(
    #    f"img {int(vect_info[i,0])}: "
     #   f"true={true_scores[i]:.2f}, pred={pred_score[i].item():.2f}"
    #)

    print("Done evaluating")


if __name__ == "__main__":
    main()
