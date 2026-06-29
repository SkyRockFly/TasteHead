package server

import (
	"fmt"
	"net/http"
	"scraper/internal/service/download"
)

type ListImagesReq struct {
	DirID     *int64   `json:"dir_path"`
	Score     *float32 `json:"score"`
	ScoreType string   `json:"score_type"`
	Limit     int      `json:"limit"`
	Cursor    int64    `json:"cursor"`
	Next      bool     `json:"next"`
}

type ListImagesResp struct {
	Images     []RowWithTags `json:"images"`
	CursorNext int64         `json:"cursor_next"`
	CursorPrev int64         `json:"cursor_prev"`
	HasMore    bool          `json:"has_more"`
}

type RowWithTags struct {
	ID         int64    `json:"id"`
	Hash       string   `json:"hash"`
	BatchID    int64    `json:"batch_id"`
	RelPath    string   `json:"rel_path"`
	ModelScore float32  `json:"model_score"`
	UserScore  *float32 `json:"user_score,omitempty"`
	Tags       []string `json:"tags"`
}

const listImagesName = "get images handler"

func ListImagesHandler(service *download.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := getCtxLogger(ctx)

		var dto ListImagesReq
		if err := decodeJSON(&dto, r); err != nil {
			handleError(w, fmt.Errorf("decode:%w", err), logger, listImagesName)
			return
		}

		req := download.ListImagesWithTagsReq{
			DirID:     dto.DirID,
			ScoreType: dto.ScoreType,
			Limit:     dto.Limit,
			Cursor:    dto.Cursor,
			Score:     dto.Score,
			Next:      dto.Next,
		}

		svcResp, err := service.ListImagesWithTags(ctx, req)
		if err != nil {
			handleError(w, fmt.Errorf("svc.GetByScore: %w", err), logger, listImagesName)
			return
		}

		resp := ListImagesResp{
			Images:     remapRowsWithTagsToDTO(svcResp.Images),
			CursorNext: svcResp.CursorNext,
			CursorPrev: svcResp.CursorPrev,
			HasMore:    svcResp.HasMore,
		}

		writeJSON(w, http.StatusOK, logger, resp)
	}
}

func remapRowsWithTagsToDTO(svc []download.RowWithTags) []RowWithTags {
	dtos := make([]RowWithTags, 0, len(svc))
	var dto RowWithTags
	for _, row := range svc {
		dto = RowWithTags{
			ID:         row.ID,
			Hash:       row.Hash,
			BatchID:    row.BatchID,
			RelPath:    row.RelPath,
			ModelScore: row.ModelScore,
			UserScore:  row.UserScore,
			Tags:       row.Tags,
		}
		dtos = append(dtos, dto)
	}

	return dtos
}
