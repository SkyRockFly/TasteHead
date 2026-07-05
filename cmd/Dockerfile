FROM golang:1.26.4-alpine as builder
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY internal ./internal
COPY cmd ./cmd

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /bin/tastehead ./cmd

FROM python:3.13.9-slim AS runtime
WORKDIR /app
COPY requirements.txt ./requirements.txt
COPY web ./web
COPY py ./py

RUN apt-get update \
    && apt-get install -y --no-install-recommends wget ca-certificates \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /bin/tastehead ./tastehead

RUN python -m venv /app/.venv \
    && /app/.venv/bin/pip install --upgrade pip

ARG TORCH_INDEX_URL=https://download.pytorch.org/whl/cpu
RUN /app/.venv/bin/pip install torch torchvision --index-url ${TORCH_INDEX_URL}
RUN /app/.venv/bin/pip install -r requirements.txt

ENV PATH="/app/.venv/bin:$PATH"

CMD [ "./tastehead" ]