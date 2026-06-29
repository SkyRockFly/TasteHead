package server

import (
	"fmt"
	"net/http"
	"scraper/internal/service/download"
)

type HardDeleteImagesDTO struct {
	IDs []int64 `json:"ids"`
}

type HardDeleteImagesResp struct {
	Rejected []Reject `json:"reject"`
}

const HardDeleteImagesName = "HardDeleteImages Handler"

func HardDeleteImagesHandler(service *download.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := getCtxLogger(ctx)
		var dto HardDeleteImagesDTO
		if err := decodeJSON(&dto, r); err != nil {
			handleError(w, fmt.Errorf("decode:%w", err), logger, HardDeleteImagesName)
			return
		}

		svcReq := download.HardDeleteImagesReq{
			IDs: dto.IDs,
		}

		rejects, err := service.HardDeleteImages(ctx, svcReq)
		if err != nil {
			handleError(w, fmt.Errorf("HardDeleteImages:%w", err), logger, HardDeleteImagesName)
			return
		}

		resp := HardDeleteImagesResp{
			Rejected: remapRejectsToDTO(rejects.Rejects),
		}
		writeJSON(w, http.StatusOK, logger, resp)
	}
}
