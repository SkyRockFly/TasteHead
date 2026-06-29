package server

import (
	"fmt"
	"net/http"
	"scraper/internal/service/download"
)

type HardDeleteBatchDTO struct {
	IDs []int64 `json:"ids"`
}

type HardDeleteBatchResp struct {
	Accepted bool `json:"accepted"`
}

const HardDeleteBatchName = "hardDeleteBatchesHandler"

func HardDeleteBatchesHandler(service *download.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := getCtxLogger(ctx)
		var dto HardDeleteBatchDTO
		if err := decodeJSON(&dto, r); err != nil {
			handleError(w, fmt.Errorf("decode:%w", err), logger, removeBatchName)
			return
		}

		svcReq := download.HardDeleteBatchReq{
			IDs: dto.IDs,
		}

		if err := service.HardDeleteBatches(ctx, svcReq); err != nil {
			handleError(w, fmt.Errorf("HardDeleteBatch:%w", err), logger, removeBatchName)
			return
		}

		resp := HardDeleteBatchResp{
			Accepted: true,
		}
		writeJSON(w, http.StatusOK, logger, resp)
	}
}
