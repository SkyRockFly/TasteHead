# TasteHead

[![Tests](https://github.com/SkyRockFly/TasteHead/actions/workflows/tests.yaml/badge.svg?branch=develop)](https://github.com/SkyRockFly/TasteHead/actions/workflows/tests.yaml)
![coverage](https://raw.githubusercontent.com/SkyRockFly/TasteHead/badges/.badges/develop/coverage.svg)


TasteHead is a local tool for collecting, organizing, scoring, and training on image datasets.
It uses OpenCLIP embeddings and a small trainable model/head to predict which scraped or imported images may be more interesting for the user. The goal is to reduce manual search time and make image filtering more comfortable.


The project consists of:
- Go backend
- PostgreSQL database
- Python / PyTorch / OpenCLIP pipeline
- Vanilla JS frontend
- Docker Compose runtime

## Status

Work-in-progress personal project.

## Configuration

Copy example config files:

```bash
cp config/config.example.yaml config/config.yaml
cp config/model.yaml.example config/model.yaml
```

config.yaml contains Go/backend configuration:

-container paths
-PostgreSQL connection
-server settings
-logger settings
-Python script paths
model.yaml contains the currently selected model for evaluation. It can be changed from the UI at runtime and is saved when the program exits.

## Docker Compose files

There are three Docker Compose files:

-compose.yaml
Base CPU-safe configuration. Contains all services and is always used.
-compose.nvidia.yaml
NVIDIA CUDA override for GPU inference/training.
-compose.amd.yaml
AMD ROCm override for GPU inference/training.

GPU compose files are overrides. They should be used together with compose.yaml.
-compose.yaml. CPU mode and contains all services, always used in launch command, where some parameters will be overrided. Obviously slow/
-compose.nvidia.yaml. Contains parameters for launching training and evaluating on Nvidia GPUs (CUDA)
-compose.nvidia.yaml. Contains parameters for launching training and evaluating on AMD GPUs (Rocm)

## Launch
CPU mode

Build:
```bash
docker compose -f compose.yaml build
```
Run:
```bash
docker compose -f compose.yaml up
```
Or build and run:
```bash
docker compose -f compose.yaml up --build
```
CPU mode is mainly intended for compatibility and small tests. Large OpenCLIP models are very slow and memory-heavy on CPU.

NVIDIA CUDA mode

Build:
```bash
docker compose -f compose.yaml -f compose.nvidia.yaml build
```
Run:
```bash
docker compose -f compose.yaml -f compose.nvidia.yaml up
```
Or build and run:
```bash
docker compose -f compose.yaml -f compose.nvidia.yaml up --build
```
AMD ROCm mode

Build:
```bash
docker compose -f compose.yaml -f compose.amd.yaml build
```
Run:
```bash
docker compose -f compose.yaml -f compose.amd.yaml up
```
Or build and run:
```bash
docker compose -f compose.yaml -f compose.amd.yaml up --build
```
## Environment

The .env file can be used to configure:
data directory path
OpenCLIP pretrained weights name or local weights path
OpenCLIP batch size
Go test environment
PostgreSQL test database settings

Example OpenCLIP pretrained value:

```env
OPENCLIP_PRETRAINED=laion2b_s32b_b79k
```
Example local weights path inside container:
```env
OPENCLIP_PRETRAINED=/appdata/openclip/open_clip_pytorch_model.bin
```
For tests, random OpenCLIP weights can be used:
```env
TEST_OPENCLIP_PRETRAINED=random
```
Random weights are useful for integration tests because they exercise the real OpenCLIP pipeline without downloading large model weights.

Do not use random embeddings for real training.

## Usage
Import local images

It is usually better to start training from images you already have.

To import local images:

Create a folder inside the configured import directory.
Put images into that folder.
Start import from the UI.
If there are no errors, a new batch will be created.
Open the batch in the UI and score the images.

Subdirectories are not processed.

Supported scores:
- 0
- 0.25
- 0.5
- 0.75
- 1.0

Score meaning

Recommended interpretation:

- 0
Does not match your taste/target at all.
- 0.25
Mostly not interesting, but contains some weak or dangerous similar features.
- 0.5
Borderline case. Some parts match, some parts do not.
- 0.75
Mostly matches your taste, but something is slightly wrong or compromised.
- 1.0
Strong positive example. Clearly matches your target.

If an image is unclear, boring, irrelevant, or you do not want to train on it, it is better to skip/ignore it instead of forcing it into 0.5.

## Training

To train a model:

- Create a tag.
- Select scored images.
- Add selected images to the tag.
- Open the training tab.
- Enter or select model name.
- Select the training tag.
- Start training.
- Set the trained model as active for evaluation.

## Scraping

TasteHead can scrape images from public webpages using CSS selectors.

To configure scraping, enter:

- Start page URL
- Image node selector
- Image URL attribute
- Next page node selector
- Next page URL attribute
- Number of pages to scrape

You can inspect page HTML with browser developer tools.

It is recommended to start with a small number of pages to test selectors before running a larger scrape.

## Testing
For Go tests, .env contains:
- Test DB section, to connect to PostgreSQL DB;
- TEST_OPENCLIP_PRETRAINED, weigts for OpenCLIP model, default value is none, so random weights are generated without downloading any weights, to make testing easier.

Go integration tests can use test settings from .env.

Default test mode should not require downloading real OpenCLIP weights.


