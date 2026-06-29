package server

import (
	"fmt"
	"net/http"
	"scraper/internal/service/download"
)

type CreateBatchDTO struct {
	BatchName string `json:"name"`
}

type CreateBatchResp struct {
	ID int64 `json:"batch_id"`
}

const createBatchName = "createBatchName"

func CreateBatchHandler(service *download.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := getCtxLogger(ctx)
		var dto CreateBatchDTO
		if err := decodeJSON(&dto, r); err != nil {
			handleError(w, fmt.Errorf("decode:%w", err), logger, createBatchName)
			return
		}

		svcReq := download.CreateBatchReq{
			Name: dto.BatchName,
		}

		batchID, err := service.CreateBatch(ctx, svcReq)
		if err != nil {
			handleError(w, fmt.Errorf("CreateBatch:%w", err), logger, createBatchName)
			return
		}

		resp := CreateBatchResp{
			ID: batchID,
		}
		writeJSON(w, http.StatusCreated, logger, resp)
	}
}
