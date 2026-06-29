package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"scraper/internal/pkg/apperror"
	"scraper/internal/pkg/middlewares"
	"scraper/internal/service/download"
	imagerowsvc "scraper/internal/service/imageRow"
	trainingsvc "scraper/internal/service/training"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
)

const (
	encodeError = `{"error":"encode failed"}`
)

var isShuttingDown atomic.Bool

type errorAPIResponse struct {
	Err string `json:"error"`
}

type healthResponse struct {
	Health bool `json:"health"`
}

type ServerOpts struct {
	TrainSVC  *trainingsvc.Service
	ScrapeSVC *download.Service
	ImageSVC  *imagerowsvc.Service
	Port      int
}

type Batch struct {
	ID        int64      `json:"id"`
	Name      string     `json:"name"`
	RelPath   string     `json:"rel_path"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

type Row struct {
	ID         int64    `json:"id"`
	Hash       string   `json:"hash"`
	BatchID    int64    `json:"batch_id"`
	RelPath    string   `json:"rel_path"`
	ModelScore float32  `json:"model_score"`
	UserScore  *float32 `json:"user_score,omitempty"`
}

type File struct {
	Dir     string `json:"dir"`
	Hash    string `json:"hash"`
	RelPath string `json:"rel_path"`
}

func StartServer(ctx context.Context, opts ServerOpts) error {
	mux := http.NewServeMux()

	mux.Handle("/ui/scraper/",
		http.StripPrefix("/ui/scraper/",
			http.FileServer(http.Dir("web/static")),
		),
	)

	loadImageRowEndpoints(mux, opts.ImageSVC)

	loadScrapeEndpoints(mux, opts.ScrapeSVC)

	loadTrainingEndpoints(mux, opts.TrainSVC)

	server := &http.Server{
		Addr:    ":" + strconv.Itoa(opts.Port),
		Handler: mux,
	}

	errs, eCtx := errgroup.WithContext(ctx)
	errs.Go(func() error {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("listenAndServe: %w", err)
		}
		return nil
	})

	<-eCtx.Done()
	isShuttingDown.Store(true)

	shutdownCtx, done := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer done()

	if err := server.Shutdown(shutdownCtx); err != nil &&
		!errors.Is(err, http.ErrServerClosed) &&
		!errors.Is(err, context.Canceled) {
		if errors.Is(err, context.DeadlineExceeded) {
			_ = server.Close()
		}
		log.Warn().Err(err).Msg("graceful shutdown")
	}

	if err := errs.Wait(); err != nil {
		return fmt.Errorf("server: %w", err)
	}
	return nil
}

func HealthCheckHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := getCtxLogger(r.Context())
		if isShuttingDown.Load() {
			writeJSON(w, http.StatusServiceUnavailable, logger, errorAPIResponse{Err: "shutting down"})
			logger.
				Warn().
				Msg("httpServer.healthcheckHandler: shutting down")
			return
		}
		resp := &healthResponse{Health: true}
		err := json.NewEncoder(w).Encode(resp)
		if err != nil {
			logger.
				Warn().
				Err(err).
				Msg("httpServer.healthcheckHandler: json.Encode failed")
			return
		}
	}
}

func writeJSON(w http.ResponseWriter, statusCode int, logger *zerolog.Logger, resp any) {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(resp); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(encodeError))
		logger.Error().Err(fmt.Errorf("encode: %w", err)).Msg("write JSON")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if _, err := w.Write(buf.Bytes()); err != nil {
		logger.Error().Err(fmt.Errorf("write: %w", err)).Msg("write JSON")
		return
	}
}

func decodeJSON(str any, r *http.Request) error {
	if str == nil {
		return fmt.Errorf("decode: empty struct")
	}
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(str); err != nil {
		return fmt.Errorf("%w, decode: %w", apperror.ErrInvalidJSON, err)
	}

	return nil
}

func getCtxLogger(ctx context.Context) *zerolog.Logger {
	if v := ctx.Value(middlewares.LoggerCtxKey); v != nil {
		if lg, ok := v.(zerolog.Logger); ok {
			return &lg
		}
	}
	logger := zerolog.Nop()
	return &logger
}

func handleError(w http.ResponseWriter, err error, log *zerolog.Logger, hndName string) {
	var code int
	var resp errorAPIResponse
	switch {
	case errors.Is(err, apperror.ErrInvalidJSON):
		log.Info().Err(err).Msg(hndName)
		code = http.StatusBadRequest
		resp.Err = "invalid json"

	case errors.Is(err, apperror.ErrBadRequest):
		log.Info().Err(err).Msg(hndName)
		code = http.StatusBadRequest
		resp.Err = "bad request"

	case errors.Is(err, apperror.ErrNotFound):
		log.Info().Err(err).Msg(hndName)
		code = http.StatusNotFound
		resp.Err = "not found"

	case errors.Is(err, apperror.ErrBackend):
		log.Error().Err(err).Msg(hndName)
		code = http.StatusBadGateway
		resp.Err = "gateway error"

	case errors.Is(err, apperror.ErrAlreadyExists):
		log.Info().Err(err).Msg(hndName)
		code = http.StatusConflict
		resp.Err = "already exists"

	case errors.Is(err, apperror.ErrUnauthorized):
		log.Info().Err(err).Msg(hndName)
		code = http.StatusUnauthorized
		resp.Err = "unauthorized"

	case errors.Is(err, apperror.ErrUniqueEntity):
		log.Info().Err(err).Msg(hndName)
		code = http.StatusConflict
		resp.Err = "trying to create duplicate of unique entity"
	default:
		log.Error().Err(err).Msg(hndName)
		code = http.StatusInternalServerError
		resp.Err = "service error"
	}
	writeJSON(w, code, log, resp)
}

func remapBatchToDTO(svc imagerowsvc.Batch) Batch {
	return Batch{
		ID:        svc.ID,
		Name:      svc.Name,
		RelPath:   svc.RelPath,
		DeletedAt: svc.DeletedAt,
	}
}

func loadImageRowEndpoints(mux *http.ServeMux, imageSVC *imagerowsvc.Service) {
	mux.HandleFunc("PUT /image/update/score",
		middlewares.LogMiddleware(UpdateUserScoreHandler(imageSVC)))

	mux.HandleFunc("POST /image/list/deleted",
		middlewares.LogMiddleware(ListDeletedImagesHandler(imageSVC)))

	mux.HandleFunc("DELETE /image/remove",
		middlewares.LogMiddleware(RemoveImgHandler(imageSVC)))

	mux.HandleFunc("GET /batch/list",
		middlewares.LogMiddleware(ListBatchesHandler(imageSVC)))

	mux.HandleFunc("GET /batch/list/deleted",
		middlewares.LogMiddleware(ListDeletedBatchesHandler(imageSVC)))

	mux.HandleFunc("DELETE /batch/remove",
		middlewares.LogMiddleware(RemoveBatchHandler(imageSVC)))

	mux.HandleFunc("PUT /batch/restore",
		middlewares.LogMiddleware(RestoreBatchesHandler(imageSVC)))

	mux.HandleFunc("PUT /image/restore",
		middlewares.LogMiddleware(RestoreImagesHandler(imageSVC)))

	mux.HandleFunc("PUT /batch/update/name",
		middlewares.LogMiddleware(UpdateBatchNameHandler(imageSVC)))
}

func loadScrapeEndpoints(mux *http.ServeMux, scrapeSVC *download.Service) {
	mux.HandleFunc("POST /scrape",
		middlewares.LogMiddleware(ScrapeImagesHandler(scrapeSVC)))

	mux.HandleFunc("POST /image/list",
		middlewares.LogMiddleware(ListImagesHandler(scrapeSVC)))

	mux.HandleFunc("POST /scrape/get/state",
		middlewares.LogMiddleware(GetScrapeStateHandler(scrapeSVC)))

	mux.HandleFunc("POST /batch/create",
		middlewares.LogMiddleware(CreateBatchHandler(scrapeSVC)))

	mux.HandleFunc("GET /images/get/pic",
		middlewares.LogMiddleware(GetImageHandler(scrapeSVC)))

	mux.HandleFunc("POST /import/batches",
		middlewares.LogMiddleware(ImportLocalDirsHandler(scrapeSVC)))

	mux.HandleFunc("POST /model/train",
		middlewares.LogMiddleware(TrainModelHandler(scrapeSVC)))

	mux.HandleFunc("GET /model/list",
		middlewares.LogMiddleware(ListModelsHandler(scrapeSVC)))

	mux.HandleFunc("GET /batches/list/unfinished",
		middlewares.LogMiddleware(ListUnfinishedBatchesHandler(scrapeSVC)))

	mux.HandleFunc("GET /batches/list/withFails",
		middlewares.LogMiddleware(ListBatchesWithFailsHandler(scrapeSVC)))

	mux.HandleFunc("POST /scrape/resume",
		middlewares.LogMiddleware(ResumeScrapeHandler(scrapeSVC)))

	mux.HandleFunc("POST /scrape/fails",
		middlewares.LogMiddleware(SolveFailsHandler(scrapeSVC)))

	mux.HandleFunc("POST /image/move",
		middlewares.LogMiddleware(MoveImagesHandler(scrapeSVC)))

	mux.HandleFunc("POST /model/set",
		middlewares.LogMiddleware(SetModelHandler(scrapeSVC)))

	mux.HandleFunc("DELETE /batch/hardDelete",
		middlewares.LogMiddleware(HardDeleteBatchesHandler(scrapeSVC)))

	mux.HandleFunc("DELETE /image/hardDelete",
		middlewares.LogMiddleware(HardDeleteImagesHandler(scrapeSVC)))
}

func loadTrainingEndpoints(mux *http.ServeMux, trainSVC *trainingsvc.Service) {
	mux.HandleFunc("POST /training/create",
		middlewares.LogMiddleware(CreateTrainingRowsHandler(trainSVC)))

	mux.HandleFunc("DELETE /training/delete",
		middlewares.LogMiddleware(RemoveTrainingRowHandler(trainSVC)))

	mux.HandleFunc("POST /training/list/req",
		middlewares.LogMiddleware(listTrainingRowsByReqHandler(trainSVC)))

	mux.HandleFunc("PUT /training/update/tag",
		middlewares.LogMiddleware(UpdateTrainingRowTagHandler(trainSVC)))

	mux.HandleFunc("POST /tag/create",
		middlewares.LogMiddleware(CreateTagHandler(trainSVC)))

	mux.HandleFunc("DELETE /tag/delete",
		middlewares.LogMiddleware(RemoveTagHandler(trainSVC)))

	mux.HandleFunc("PUT /tag/update/name",
		middlewares.LogMiddleware(UpdateTagNameHandler(trainSVC)))

	mux.HandleFunc("GET /tag/list",
		middlewares.LogMiddleware(ListTagsHandler(trainSVC)))

	mux.HandleFunc("POST /training/scores/composition",
		middlewares.LogMiddleware(listScoreCompositionsHandler(trainSVC)))
}
