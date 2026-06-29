package server

import (
	"fmt"
	"net/http"
	imagerowsvc "scraper/internal/service/imageRow"
)

type RemoveBatchDTO struct {
	ID int `json:"id"`
}

type RemoveBatchResp struct {
	Accepted bool `json:"accepted"`
}

const removeBatchName = "removeTag Handler"

func RemoveBatchHandler(service *imagerowsvc.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := getCtxLogger(ctx)
		var dto RemoveBatchDTO
		if err := decodeJSON(&dto, r); err != nil {
			handleError(w, fmt.Errorf("decode:%w", err), logger, removeBatchName)
			return
		}

		svcReq := imagerowsvc.RemoveBatchReq{
			ID: dto.ID,
		}

		if err := service.RemoveBatch(ctx, svcReq); err != nil {
			handleError(w, fmt.Errorf("RemoveTag:%w", err), logger, removeBatchName)
			return
		}

		resp := RemoveTagResp{
			Accepted: true,
		}
		writeJSON(w, http.StatusOK, logger, resp)
	}
}
