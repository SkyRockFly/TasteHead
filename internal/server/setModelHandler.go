package server

import (
	"fmt"
	"net/http"
	"scraper/internal/service/download"
)

type SetModelDTO struct {
	ModelName string `json:"model_name"`
}

type SetModelResp struct {
	Accepted bool `json:"accepted"`
}

const setModelName = "setModelHandler"

func SetModelHandler(service *download.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := getCtxLogger(ctx)

		var dto SetModelDTO
		if err := decodeJSON(&dto, r); err != nil {
			handleError(w, fmt.Errorf("decode:%w", err), logger, setModelName)
			return
		}

		req := download.SetModelReq{
			ModelName: dto.ModelName,
		}

		if err := service.SetModel(req); err != nil {
			handleError(w, fmt.Errorf("SetModel:%w", err), logger, setModelName)
			return
		}

		resp := SetModelResp{
			Accepted: true,
		}

		writeJSON(w, http.StatusOK, logger, resp)
	}
}
