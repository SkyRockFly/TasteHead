package server

import (
	"fmt"
	"net/http"
	trainingsvc "scraper/internal/service/training"
)

type CreateRowDTO struct {
	TagID    int     `json:"tag_id"`
	ImageIDs []int64 `json:"image_ids"`
}

type CreateRowResp struct {
	Duplicates []int64 `json:"duplicates"`
}

const createRowsHandlerName = "createRowsHandler"

func CreateTrainingRowsHandler(service *trainingsvc.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := getCtxLogger(ctx)
		var dto CreateRowDTO
		if err := decodeJSON(&dto, r); err != nil {
			handleError(w, fmt.Errorf("decode:%w", err), logger, createRowsHandlerName)
			return
		}

		svcReq := trainingsvc.CreateRowReq{
			ImageIDs: dto.ImageIDs,
			TagID:    dto.TagID,
		}

		svcResp, err := service.CreateRows(ctx, svcReq)
		if err != nil {
			handleError(w, fmt.Errorf("CreateTrainingRows:%w", err), logger, createRowsHandlerName)
			return
		}

		resp := CreateRowResp{
			Duplicates: svcResp.Duplicates,
		}

		writeJSON(w, http.StatusCreated, logger, resp)
	}
}
