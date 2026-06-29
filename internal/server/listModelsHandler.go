package server

import (
	"fmt"
	"net/http"
	"scraper/internal/service/download"
)

type ListModelsResp struct {
	Models []string `json:"models"`
}

const listModelsName = "listModelsHandler"

func ListModelsHandler(service *download.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := getCtxLogger(ctx)

		models, err := service.ListModels()
		if err != nil {
			handleError(w, fmt.Errorf("svc.ListModels: %w", err), logger, listModelsName)
			return
		}

		resp := ListModelsResp{
			Models: models,
		}

		writeJSON(w, http.StatusOK, logger, resp)
	}
}
