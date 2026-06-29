package server

import (
	"fmt"
	"net/http"
	imagerowsvc "scraper/internal/service/imageRow"
)

type ListBatchesResp struct {
	Batch []Batch `json:"batches"`
}

const listBatchesName = "get download info handler"

func ListBatchesHandler(service *imagerowsvc.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := getCtxLogger(ctx)

		svcResp, err := service.ListBatches(ctx)
		if err != nil {
			handleError(w, fmt.Errorf("svc.ListBatches: %w", err), logger, listBatchesName)
			return
		}

		dtoBatches := make([]Batch, 0, len(svcResp))
		var dtoBatch Batch
		for _, svcBatch := range svcResp {
			dtoBatch = remapBatchToDTO(svcBatch)
			dtoBatches = append(dtoBatches, dtoBatch)
		}
		resp := ListBatchesResp{
			Batch: dtoBatches,
		}

		writeJSON(w, http.StatusOK, logger, resp)
	}
}
