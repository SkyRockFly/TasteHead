package server

import (
	"fmt"
	"net/http"
	imagerowsvc "scraper/internal/service/imageRow"
)

type RestoreImagesDTO struct {
	IDs []int64 `json:"ids"`
}

type RestoreImagesResp struct {
	Accepted bool `json:"accepted"`
}

const restoreImagesName = "restoreImages Handler"

func RestoreImagesHandler(service *imagerowsvc.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := getCtxLogger(ctx)
		var dto RestoreBatchesDTO
		if err := decodeJSON(&dto, r); err != nil {
			handleError(w, fmt.Errorf("decode:%w", err), logger, restoreImagesName)
			return
		}

		svcReq := imagerowsvc.RestoreImagesReq{
			IDs: dto.IDs,
		}

		if err := service.RestoreImages(ctx, svcReq); err != nil {
			handleError(w, fmt.Errorf("RestoreImages:%w", err), logger, restoreImagesName)
			return
		}

		resp := RestoreBatchesResp{
			Accepted: true,
		}
		writeJSON(w, http.StatusOK, logger, resp)
	}
}
