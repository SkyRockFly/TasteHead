package server

import (
	"fmt"
	"net/http"
	"scraper/internal/service/download"
)

const listUnfinishedBatchesName = "listUnfinishedBatchedHandler"

type ListUnfinishedBatchesResp struct {
	Batches []string `json:"batches"`
}

func ListUnfinishedBatchesHandler(service *download.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := getCtxLogger(ctx)

		dirs, err := service.ListUnfinishedBatches()
		if err != nil {
			handleError(w, fmt.Errorf("ListUnfinishedBatches: %w", err), logger, listUnfinishedBatchesName)
			return
		}

		resp := ListUnfinishedBatchesResp{
			Batches: dirs,
		}
		writeJSON(w, http.StatusOK, logger, resp)
	}
}
