package server

import (
	"fmt"
	"net/http"
	trainingsvc "scraper/internal/service/training"
)

type UpdateTagNameDTO struct {
	ID         int    `json:"id"`
	UpdateName string `json:"update_name"`
}

type UpdateTagNameResp struct {
	Accepted bool `json:"accepted"`
}

const updateTagName = "updateTagName Handler"

func UpdateTagNameHandler(service *trainingsvc.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := getCtxLogger(ctx)
		var dto UpdateTagNameDTO
		if err := decodeJSON(&dto, r); err != nil {
			handleError(w, fmt.Errorf("decode:%w", err), logger, updateTagName)
			return
		}

		svcReq := trainingsvc.UpdateTagNameReq{
			ID:         dto.ID,
			UpdateName: dto.UpdateName,
		}

		if err := service.UpdateTagName(ctx, svcReq); err != nil {
			handleError(w, fmt.Errorf("UpdateTagName:%w", err), logger, updateTagName)
			return
		}

		resp := UpdateTagNameResp{
			Accepted: true,
		}
		writeJSON(w, http.StatusOK, logger, resp)
	}
}
