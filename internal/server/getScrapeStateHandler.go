package server

import (
	"fmt"
	"net/http"
	"scraper/internal/service/download"
)

type GetScrapeStateDTO struct {
	ID int `json:"id"`
}

type GetScrapeStateResp struct {
	NextURL          string `json:"next_url"`
	PostSelector     string `json:"post_selector"`
	ImageAttr        string `json:"image_attr"`
	NextPageSelector string `json:"next_page_selector"`
	NextPageAttr     string `json:"next_page_attr"`
}

const getScrapeStateName = "GetScrapeStateHandler"

func GetScrapeStateHandler(service *download.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := getCtxLogger(ctx)

		var dto GetScrapeStateDTO
		if err := decodeJSON(&dto, r); err != nil {
			handleError(w, fmt.Errorf("decode:%w", err), logger, getScrapeStateName)
			return
		}

		req := download.ReadBatchScrapeStateReq{
			BatchID: dto.ID,
		}

		state, err := service.ReadBatchScrapeState(ctx, req)
		if err != nil {
			handleError(w, fmt.Errorf("ReadBatchScrapeState: %w", err), logger, getScrapeStateName)
			return
		}

		resp := remapBatchScrapeToDTO(state)

		writeJSON(w, http.StatusOK, logger, resp)
	}
}

func remapBatchScrapeToDTO(svc download.ReadBatchScrapeStateResp) GetScrapeStateResp {
	return GetScrapeStateResp{
		NextURL:          svc.NextURL,
		PostSelector:     svc.PostSelector,
		ImageAttr:        svc.ImageAttr,
		NextPageSelector: svc.NextPageSelector,
		NextPageAttr:     svc.NextPageAttr,
	}
}
