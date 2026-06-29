package server

import (
	"fmt"
	"net/http"
	"scraper/internal/service/download"
)

type TrainModelReq struct {
	ModelName string `json:"model_name"`
	TagID     int    `json:"tag_id"`
}

type TrainModelResp struct {
	Trained bool `json:"trained"`
}

const trainModelName = "trainModel handler"

func TrainModelHandler(service *download.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := getCtxLogger(ctx)

		var dto TrainModelReq
		if err := decodeJSON(&dto, r); err != nil {
			handleError(w, fmt.Errorf("decode:%w", err), logger, trainModelName)
			return
		}

		req := download.TrainModelReq{
			ModelName: dto.ModelName,
			TagID:     dto.TagID,
		}

		if err := service.TrainModel(ctx, req); err != nil {
			handleError(w, fmt.Errorf("svc.TrainModel: %w", err), logger, trainModelName)
			return
		}

		resp := TrainModelResp{
			Trained: true,
		}

		writeJSON(w, http.StatusCreated, logger, resp)
	}
}
