package server

import (
	"fmt"
	"net/http"
	"scraper/internal/service/download"
)

type ResumeScrapeDTO struct {
	BatchName string `json:"batch_name"`
}

const resumeScrapeName = "resumeScrapeHandler"

func ResumeScrapeHandler(service *download.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := getCtxLogger(ctx)
		var dto ResumeScrapeDTO
		if err := decodeJSON(&dto, r); err != nil {
			handleError(w, fmt.Errorf("decode:%w", err), logger, resumeScrapeName)
			return
		}

		svcReq := download.ResumeScrapeReq{
			BatchName: dto.BatchName,
		}

		svcResp, err := service.ResumeScrape(ctx, svcReq)
		if err != nil {
			handleError(w, fmt.Errorf("ResumeScrape:%w", err), logger, resumeScrapeName)
			return
		}

		fails := remapFailsToDTO(svcResp.ErrDownload)

		resp := ScrapeImagesResp{
			BatchID:     svcResp.BatchID,
			ErrDownload: fails,
		}

		writeJSON(w, http.StatusOK, logger, resp)
	}
}
