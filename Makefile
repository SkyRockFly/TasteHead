-include .env
# neural network section

.PHONY: all manage-files model-computing clip-ingest model-training

eval: manage-files clip-ingest model-computing
train: manage-files clip-ingest-train model-training

training-with-go: clip-ingest-train model-training-go


manage-files:
	$(PYTHON) ./py/src/fileManager.py --dir "$(IMG_DIR)"


clip-ingest:
	@PYTORCH_ALLOC_CONF=expandable_segments:True $(PYTHON) ./py/src/ingest_clip.py --csvPath "$(IMG_DIR)\output.csv"

clip-ingest-train:
	@PYTORCH_ALLOC_CONF=expandable_segments:True $(PYTHON) ./py/src/ingest_clip.py --csvPath "$(CSV_PATH_TRAIN)"  --dataPath "$(DATA_PATH_TRAIN)" --train_mode

model-computing:
	@PYTORCH_ALLOC_CONF=expandable_segments:True $(PYTHON)  ./py/src/train_network.py --vect_path "$(IMG_DIR)\output\clip_eval_vecs.npy" --model_path "$(MODEL_PATH)" 

model-training:
	$(PYTHON) ./py/src/train_network.py --vect_path "$(IMG_DIR)\output\clip_training_vecs.npy" --score_path "$(IMG_DIR)\output\vect_training_info.npy" --learn_mode true

model-training-go:
	$(PYTHON) ./py/src/train_network.py --vect_path "$(CSV_DIR_TRAIN)\output\clip_training_vecs.npy" --score_path "$(CSV_DIR_TRAIN)\output\vect_training_info.npy" --learn_mode --output "$(OUTPUT)"


# Test section
TASTE_DB_URL       ?= postgres://$(TASTE_DB_USER):$(TASTE_DB_PASSWORD)@localhost:$(TASTE_DB_PORT_HOST)/$(TASTE_DB_NAME)?sslmode=disable

TASTE_MIGRATIONS_DIR ?= ./migrations
TASTEHEAD_SCRIPTS_DIR ?= $(abspath py/src)
TASTEHEAD_PYTHON_VENV_DIR ?= $(abspath .venv)
GO_TEST_TASTE_FLAGS  ?= ./internal/server -v -count=1 -run TestTrainModelHandler/01_OK

.PHONY: taste-db-up taste-db-wait taste-migrate taste-test taste-db-down

taste-db-up:
	@docker rm -f $(TASTE_DB_CONTAINER) >/dev/null 2>&1 || true
	@docker run -d --name $(TASTE_DB_CONTAINER) \
		-e POSTGRES_PASSWORD=$(TASTE_DB_PASSWORD) \
		-e POSTGRES_DB=$(TASTE_DB_NAME) \
		-p $(TASTE_DB_PORT_HOST):$(TASTE_DB_PORT_CONT) \
		$(TASTE_DB_IMAGE) >/dev/null
	@$(MAKE) taste-db-wait

taste-db-wait:
	@for i in $$(seq 1 $(WAIT_MAX)); do \
	  docker exec $(TASTE_DB_CONTAINER) pg_isready -U $(TASTE_DB_USER) -d $(TASTE_DB_NAME) >/dev/null 2>&1 && exit 0; \
	  sleep 1; \
	done; \
	echo "timeout waiting for tastehead postgres"; \
	docker logs --tail=100 $(TASTE_DB_CONTAINER) || true; \
	exit 1

taste-migrate:
	@$(GOOSE_BIN) -dir $(TASTE_MIGRATIONS_DIR) postgres "$(TASTE_DB_URL)" up

taste-test: taste-db-up taste-migrate
	@DB_URL="$(TASTE_DB_URL)"  \
	TASTEHEAD_SCRIPTS_DIR="$(TASTEHEAD_SCRIPTS_DIR)" \
	TASTEHEAD_PYTHON_VENV_DIR="$(TASTEHEAD_PYTHON_VENV_DIR)" \
	OPENCLIP_PRETRAINED="$(TEST_OPENCLIP_PRETRAINED)" \
	go test $(GO_TEST_TASTE_FLAGS)
	@$(MAKE) taste-db-down

taste-db-down:
	@docker rm -f $(TASTE_DB_CONTAINER) >/dev/null 2>&1 || true