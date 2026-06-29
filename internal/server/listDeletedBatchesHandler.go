package server

import (
	"fmt"
	"net/http"
	imagerowsvc "scraper/internal/service/imageRow"
)

type ListDeletedBatchesResp struct {
	Batch []Batch `json:"batches"`
}

const ListDeletedBatchesName = "list deletedBatchesName"

func ListDeletedBatchesHandler(service *imagerowsvc.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := getCtxLogger(ctx)

		svcResp, err := service.ListDeletedBatches(ctx)
		if err != nil {
			handleError(w, fmt.Errorf("svc.ListBatches: %w", err), logger, ListDeletedBatchesName)
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
