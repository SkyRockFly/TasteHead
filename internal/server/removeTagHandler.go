package server

import (
	"fmt"
	"net/http"
	trainingsvc "scraper/internal/service/training"
)

type RemoveTagDTO struct {
	ID int `json:"id"`
}

type RemoveTagResp struct {
	Accepted bool `json:"accepted"`
}

const removeTagName = "removeTag Handler"

func RemoveTagHandler(service *trainingsvc.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := getCtxLogger(ctx)
		var dto RemoveTagDTO
		if err := decodeJSON(&dto, r); err != nil {
			handleError(w, fmt.Errorf("decode:%w", err), logger, removeTagName)
			return
		}

		svcReq := trainingsvc.RemoveTagReq{
			ID: dto.ID,
		}

		if err := service.RemoveTag(ctx, svcReq); err != nil {
			handleError(w, fmt.Errorf("RemoveTag:%w", err), logger, removeTagName)
			return
		}

		resp := RemoveTagResp{
			Accepted: true,
		}
		writeJSON(w, http.StatusOK, logger, resp)
	}
}
