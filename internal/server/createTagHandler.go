package server

import (
	"fmt"
	"net/http"
	trainingsvc "scraper/internal/service/training"
)

type CreateTagDTO struct {
	Name string `json:"name"`
	Desc string `json:"desc,omitempty"`
}

type CreateTagResp struct {
	Accepted bool `json:"accepted"`
}

const createTagName = "createTag Handler"

func CreateTagHandler(service *trainingsvc.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := getCtxLogger(ctx)
		var dto CreateTagDTO
		if err := decodeJSON(&dto, r); err != nil {
			handleError(w, fmt.Errorf("decode:%w", err), logger, createTagName)
			return
		}

		svcReq := trainingsvc.CreateTagReq{
			Name: dto.Name,
			Desc: dto.Desc,
		}

		if err := service.CreateTag(ctx, svcReq); err != nil {
			handleError(w, fmt.Errorf("CreateTag:%w", err), logger, createTagName)
			return
		}

		resp := CreateTagResp{
			Accepted: true,
		}
		writeJSON(w, http.StatusCreated, logger, resp)
	}
}
