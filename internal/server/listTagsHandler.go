package server

import (
	"fmt"
	"net/http"
	trainingsvc "scraper/internal/service/training"
)

type ListTagsResp struct {
	Tags []Tag `json:"tags"`
}

type Tag struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Desc string `json:"desc"`
}

const listTagName = "listTag Handler"

func ListTagsHandler(service *trainingsvc.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := getCtxLogger(ctx)

		svcTags, err := service.ListTags(ctx)
		if err != nil {
			handleError(w, fmt.Errorf("ListTags:%w", err), logger, listTagName)
			return
		}

		dtoTags := make([]Tag, 0, len(svcTags))
		var dtoTag Tag
		for _, tag := range svcTags {
			dtoTag = remapTagToDTO(tag)
			dtoTags = append(dtoTags, dtoTag)
		}

		resp := ListTagsResp{
			Tags: dtoTags,
		}

		writeJSON(w, http.StatusOK, logger, resp)
	}
}

func remapTagToDTO(svc trainingsvc.Tag) Tag {
	return Tag{
		ID:   svc.ID,
		Name: svc.Name,
		Desc: svc.Desc,
	}
}
