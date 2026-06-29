package server

import (
	"fmt"
	"net/http"
	"scraper/internal/service/download"
)

type SolveFaildDTO struct {
	BatchName string `json:"batch_name"`
}

type SolveFailsResp struct {
	Fails []Fails `json:"fails"`
}

const solveFailsName = "solveFailsHandler"

func SolveFailsHandler(service *download.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := getCtxLogger(ctx)
		var dto SolveFaildDTO
		if err := decodeJSON(&dto, r); err != nil {
			handleError(w, fmt.Errorf("decode:%w", err), logger, solveFailsName)
			return
		}

		svcReq := download.SolveFailsReq{
			DirName: dto.BatchName,
		}

		svcfails, err := service.SolveFails(ctx, svcReq)
		if err != nil {
			handleError(w, fmt.Errorf("SolveFails:%w", err), logger, solveFailsName)
			return
		}

		fails := remapFailsToDTO(svcfails)

		resp := SolveFailsResp{
			Fails: fails,
		}

		writeJSON(w, http.StatusOK, logger, resp)
	}
}
