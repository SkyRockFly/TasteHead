package server

import (
	"fmt"
	"net/http"
	trainingsvc "scraper/internal/service/training"
)

type RemoveRowDTO struct {
	ID int64 `json:"id"`
}

type RemoveRowResp struct {
	Accepted bool `json:"accepted"`
}

const removeRowHandler = "createRowsHandler"

func RemoveTrainingRowHandler(service *trainingsvc.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := getCtxLogger(ctx)
		var dto RemoveRowDTO
		if err := decodeJSON(&dto, r); err != nil {
			handleError(w, fmt.Errorf("decode:%w", err), logger, removeRowHandler)
			return
		}

		svcReq := trainingsvc.RemoveRowReq{
			ID: dto.ID,
		}

		if err := service.RemoveRow(ctx, svcReq); err != nil {
			handleError(w, fmt.Errorf("RemoveRow:%w", err), logger, removeRowHandler)
			return
		}

		resp := RemoveRowResp{
			Accepted: true,
		}
		writeJSON(w, http.StatusCreated, logger, resp)
	}
}
