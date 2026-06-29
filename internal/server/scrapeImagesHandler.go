package server

import (
	"fmt"
	"net/http"
	"scraper/internal/pkg/apperror"
	"scraper/internal/service/download"
)

type ScrapeImagesDTO struct {
	URL              string `json:"url"`
	PostSelector     string `json:"post_selector"`
	ImageAttr        string `json:"image_attr"`
	NextPageSelector string `json:"next_page_selector"`
	NextPageAttr     string `json:"next_page_attr"`
	Pages            int    `json:"pages"`
	Limit            int    `json:"limit"`
}

type ScrapeImagesResp struct {
	BatchID     int64   `json:"batch_id"`
	ErrDownload []Fails `json:"errs,omitempty"`
}

type FailedDownload struct {
	Issue      apperror.DownloadIssue `json:"issue"`
	URL        string                 `json:"url"`
	Repeatable bool                   `json:"repeatable"`
}

type Fails struct {
	OriginURL string           `json:"origin_url"`
	Fails     []FailedDownload `json:"fails"`
}

const scrapeImagesName = "scrape images handler"

func ScrapeImagesHandler(service *download.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := getCtxLogger(ctx)

		var dto ScrapeImagesDTO
		if err := decodeJSON(&dto, r); err != nil {
			handleError(w, fmt.Errorf("decode:%w", err), logger, scrapeImagesName)
			return
		}

		req := download.ScrapeImagesReq{
			URL:              dto.URL,
			PostSelector:     dto.PostSelector,
			ImageAttr:        dto.ImageAttr,
			NextPageSelector: dto.NextPageSelector,
			NextPageAttr:     dto.NextPageAttr,
			Pages:            dto.Pages,
			Limit:            dto.Limit,
		}

		svcResp, err := service.ScrapePages(ctx, req)
		if err != nil {
			handleError(w, fmt.Errorf("svc.scrapeImages: %w", err), logger, scrapeImagesName)
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

func remapFailedDownloadToDTO(svc []download.FailedDownload) []FailedDownload {
	fails := make([]FailedDownload, 0, len(svc))
	var fail FailedDownload
	for _, f := range svc {
		fail = FailedDownload{
			Issue:      f.Issue,
			URL:        f.URL,
			Repeatable: f.Repeatable,
		}
		fails = append(fails, fail)
	}
	return fails
}

func remapFailsToDTO(svc []download.Fails) []Fails {
	fails := make([]Fails, 0, len(svc))
	var fail Fails
	for _, f := range svc {
		failedDownloads := remapFailedDownloadToDTO(f.Fails)
		fail = Fails{
			OriginURL: f.OriginURL,
			Fails:     failedDownloads,
		}
		fails = append(fails, fail)
	}

	return fails
}
