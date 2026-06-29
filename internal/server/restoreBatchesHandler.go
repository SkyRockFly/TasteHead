package server

import (
	"fmt"
	"net/http"
	imagerowsvc "scraper/internal/service/imageRow"
)

type RestoreBatchesDTO struct {
	IDs []int64 `json:"ids"`
}

type RestoreBatchesResp struct {
	Accepted bool `json:"accepted"`
}

const restoreBatchesName = "restoreBatches Handler"

func RestoreBatchesHandler(service *imagerowsvc.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := getCtxLogger(ctx)
		var dto RestoreBatchesDTO
		if err := decodeJSON(&dto, r); err != nil {
			handleError(w, fmt.Errorf("decode:%w", err), logger, restoreBatchesName)
			return
		}

		svcReq := imagerowsvc.RestoreBatchesReq{
			IDs: dto.IDs,
		}

		if err := service.RestoreBatches(ctx, svcReq); err != nil {
			handleError(w, fmt.Errorf("RestoreBatches:%w", err), logger, restoreBatchesName)
			return
		}

		resp := RestoreBatchesResp{
			Accepted: true,
		}
		writeJSON(w, http.StatusOK, logger, resp)
	}
}
