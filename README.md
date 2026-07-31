# TasteHead

[![Tests](https://github.com/SkyRockFly/TasteHead/actions/workflows/tests.yaml/badge.svg?branch=develop)](https://github.com/SkyRockFly/TasteHead/actions/workflows/tests.yaml)
![coverage](https://raw.githubusercontent.com/SkyRockFly/TasteHead/badges/.badges/develop/coverage.svg)



TasteHead is a local tool for collecting, organizing, scoring, and training models on image datasets.

It uses OpenCLIP embeddings and a small trainable scoring head to predict which scraped or imported images are likely to match the user's preferences. Its purpose is to reduce manual search time and make large image collections easier to filter.

The project consists of:

- Go backend
- PostgreSQL database
- Python / PyTorch / OpenCLIP pipeline
- Vanilla JavaScript frontend
- Docker Compose runtime

## Status

Functional personal project under development.

## Docker Compose files

There are three Docker Compose files:

- `compose.yaml`
Base CPU-safe configuration. Contains all services and is always used.

- `compose.nvidia.yaml`
NVIDIA CUDA override for GPU inference/training.

- `compose.amd.yaml`
AMD ROCm override for GPU inference/training.

GPU compose files are overrides. They should be used together with compose.yaml.


## Launch

Download repository:

```bash
git clone https://github.com/SkyRockFly/TasteHead
cd TasteHead
```

Copy example.yaml to config.yaml:
```bash
cp ./config/example.yaml ./config/config.yaml
```

config.yaml contains Go/backend configuration:

- container paths
- PostgreSQL connection
- server settings
- logger settings
- Python script paths
- Scrape config (intervals, user-agent)

Copy model.yaml.example to model.yaml:
```bash
cp ./config/model.yaml.example ./config/model.yaml
```

model.yaml contains the currently selected model for evaluation. It can be changed from the UI at runtime and is saved when the program exits.

Copy .env.example to .env:
```bash
cp .env.example .env
```

CPU mode:

Build and run:
```bash
docker compose -f compose.yaml up --build
```
CPU mode is mainly intended for compatibility and small tests. Large OpenCLIP models are very slow and memory-heavy on CPU.

NVIDIA CUDA mode

Build and run:
```bash
docker compose -f compose.yaml -f compose.nvidia.yaml up --build
```
AMD ROCm mode

Build and run:
```bash
docker compose -f compose.yaml -f compose.amd.yaml up --build
```

Open web-page on `http://localhost:8080/ui/scraper/`

On the first ingest_clip run, TasteHead downloads the configured pretrained OpenCLIP weights if they are not already cached. The download may be several gigabytes and requires network access. Subsequent runs use the cached weights.

## Environment

The .env file can be used to configure:
- Data directory path
- OpenCLIP pretrained weights tag or local weights path
- OpenCLIP batch size
- Training epochs
- Learning rate
- Go test environment
- PostgreSQL test database settings

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

### Batch size

```env
TASTEHEAD_CLIP_BATCH_SIZE=16
```

Number of images processed in one batch. Higher values require more GPU VRAM.

### Data directory

```env
TASTEHEAD_DATA_DIR=./data
```

Path to the local data directory.

### Training epochs

```env
TASTEHEAD_EPOCHS=120
```

Number of epochs used to train the scoring model.

### Learning rate

```env
TASTEHEAD_LEARNING_RATE=0.001
```

Learning rate used during model training.

## Usage
### Import local images

It is usually better to start training from images you already have.

To import local images:

Create a folder inside the configured import directory.
Put images into that folder.
Start import from the UI.
If there are no errors, a new batch will be created.
Open the batch in the UI and score the images.

Subdirectories are not processed.

### Supported scores:
- 0
- 0.25
- 0.5
- 0.75
- 1.0

### Score meaning

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


