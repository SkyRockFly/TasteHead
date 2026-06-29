package server

import (
	"fmt"
	"net/http"
	imagerowsvc "scraper/internal/service/imageRow"
)

type UpdateBatchNameDTO struct {
	BatchID   int64  `json:"batch_id"`
	BatchName string `json:"batch_name"`
}

type UpdateBatchNameResp struct {
	Accepted bool `json:"accepted"`
}

const updateBatchNameName = "updateBatchNameHandler"

func UpdateBatchNameHandler(service *imagerowsvc.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := getCtxLogger(ctx)

		var dto UpdateBatchNameDTO
		if err := decodeJSON(&dto, r); err != nil {
			handleError(w, fmt.Errorf("decode:%w", err), logger, updateBatchNameName)
			return
		}
		req := imagerowsvc.UpdateBatchNameReq{
			BatchID:   dto.BatchID,
			BatchName: dto.BatchName,
		}

		if err := service.UpdateBatchName(ctx, req); err != nil {
			handleError(w, fmt.Errorf("UpdateBatchName: %w", err), logger, updateBatchNameName)
			return
		}

		writeJSON(w, http.StatusOK, logger, UpdateBatchNameResp{Accepted: true})
	}
}
