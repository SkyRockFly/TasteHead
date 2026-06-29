package server

import (
	"fmt"
	"net/http"
	"scraper/internal/service/download"
)

const listBatchesWithFails = "listBatchesWithFails"

type ListBatchesWithFailsResp struct {
	Batches []string `json:"batches"`
}

func ListBatchesWithFailsHandler(service *download.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := getCtxLogger(ctx)

		dirs, err := service.ListBatchesWithFails()
		if err != nil {
			handleError(w, fmt.Errorf("ListBatchesWithFails: %w", err), logger, listBatchesWithFails)
			return
		}

		resp := ListBatchesWithFailsResp{
			Batches: dirs,
		}
		writeJSON(w, http.StatusOK, logger, resp)
	}
}
