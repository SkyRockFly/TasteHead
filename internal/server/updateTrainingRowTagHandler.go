package server

import (
	"fmt"
	"net/http"
	trainingsvc "scraper/internal/service/training"
)

type UpdateRowTagDTO struct {
	IDs   []int64 `json:"ids"`
	TagID int     `json:"tag_id"`
}

type UpdateRowTagResp struct {
	Accepted bool `json:"accepted"`
}

const updateTrainingRowTagName = "updateTrainingRowTag handler"

func UpdateTrainingRowTagHandler(service *trainingsvc.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := getCtxLogger(ctx)
		var dto UpdateRowTagDTO
		if err := decodeJSON(&dto, r); err != nil {
			handleError(w, fmt.Errorf("decode:%w", err), logger, updateTrainingRowTagName)
			return
		}

		svcReq := trainingsvc.UpdateRowTagReq{
			IDs:   dto.IDs,
			TagID: dto.TagID,
		}

		if err := service.UpdateRowTag(ctx, svcReq); err != nil {
			handleError(w, fmt.Errorf("UpdateTrainingRowName:%w", err), logger, updateTrainingRowTagName)
			return
		}

		resp := UpdateRowTagResp{
			Accepted: true,
		}
		writeJSON(w, http.StatusOK, logger, resp)
	}
}
