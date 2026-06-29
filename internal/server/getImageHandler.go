package server

import (
	"fmt"
	"net/http"
	"scraper/internal/service/download"
)

const getImageName = "get images handler"

func GetImageHandler(service *download.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := getCtxLogger(ctx)

		path := r.URL.Query().Get("path")

		req := download.GetImagesReq{
			Path: path,
		}

		svcResp, err := service.GetImages(req)
		if err != nil {
			handleError(w, fmt.Errorf("svc.getImages: %w", err), logger, getImageName)
			return
		}

		http.ServeFile(w, r, svcResp.FullPath)
	}
}
