package server

import (
	"fmt"
	"net/http"
	imagerowsvc "scraper/internal/service/imageRow"
)

type RemoveImgsDTO struct {
	IDs []int64 `json:"ids"`
}

type RemoveImgsResp struct {
	Accepted bool `json:"accepted"`
}

const removeImgsName = "removeImgs handler"

func RemoveImgHandler(service *imagerowsvc.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := getCtxLogger(ctx)

		var dto RemoveImgsDTO
		if err := decodeJSON(&dto, r); err != nil {
			handleError(w, fmt.Errorf("decode:%w", err), logger, removeImgsName)
			return
		}

		req := imagerowsvc.RemoveImgReq{
			IDs: dto.IDs,
		}

		if err := service.RemoveImages(ctx, req); err != nil {
			handleError(w, fmt.Errorf("RemoveImg: %w", err), logger, removeImgsName)
			return
		}

		resp := RemoveImgsResp{
			Accepted: true,
		}

		writeJSON(w, http.StatusOK, logger, resp)
	}
}
