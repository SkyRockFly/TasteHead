package server

import (
	"fmt"
	"net/http"
	trainingsvc "scraper/internal/service/training"
)

type ListQueryRow struct {
	ID         int64    `json:"id"`
	Hash       string   `json:"hash"`
	BatchID    int      `json:"batch_id"`
	RelPath    string   `json:"rel_path"`
	TagName    string   `json:"tag_name"`
	ModelScore float32  `json:"model_score"`
	UserScore  *float32 `json:"user_score"`
}

type ListByTagRow struct {
	ID         int64    `json:"id"`
	Hash       string   `json:"hash"`
	BatchPath  string   `json:"batch_path"`
	RelPath    string   `json:"rel_path"`
	TagName    string   `json:"tag_name"`
	ModelScore float32  `json:"model_score"`
	UserScore  *float32 `json:"user_score"`
}

type ListRowsByReqDTO struct {
	TagID     *int     `json:"tag_id"`
	UserScore *float32 `json:"user_score"`
	Limit     int      `json:"limit"`
	Cursor    int64    `json:"cursor"`
	Next      bool     `json:"next"`
}

type ListRowsByReqResp struct {
	Rows       []ListQueryRow `json:"rows"`
	CursorNext int64          `json:"cursor_next"`
	CursorPrev int64          `json:"cursor_prev"`
	HasMore    bool           `json:"has_more"`
}

const listTrainingRowsByReq = "listTrainingRowsByReq handler"

func listTrainingRowsByReqHandler(service *trainingsvc.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := getCtxLogger(ctx)
		var dto ListRowsByReqDTO
		if err := decodeJSON(&dto, r); err != nil {
			handleError(w, fmt.Errorf("decode:%w", err), logger, listTrainingRowsByReq)
			return
		}

		svcReq := trainingsvc.ListRowsByReq{
			TagID:     dto.TagID,
			UserScore: dto.UserScore,
			Limit:     dto.Limit,
			Cursor:    dto.Cursor,
			Next:      dto.Next,
		}

		svcResp, err := service.ListRowsByReq(ctx, svcReq)
		if err != nil {
			handleError(w, fmt.Errorf("ListRowsByReq:%w", err), logger, listTrainingRowsByReq)
			return
		}

		dtoRows := make([]ListQueryRow, 0, len(svcResp.Rows))
		var dtoRow ListQueryRow
		for _, row := range svcResp.Rows {
			dtoRow = remapListQueryToDTO(row)
			dtoRows = append(dtoRows, dtoRow)
		}

		resp := ListRowsByReqResp{
			Rows:       dtoRows,
			CursorNext: svcResp.CursorNext,
			CursorPrev: svcResp.CursorPrev,
			HasMore:    svcResp.HasMore,
		}

		writeJSON(w, http.StatusOK, logger, resp)
	}
}

func remapListQueryToDTO(svc trainingsvc.ListQueryRow) ListQueryRow {
	return ListQueryRow{
		ID:         svc.ID,
		Hash:       svc.Hash,
		BatchID:    svc.BatchID,
		RelPath:    svc.RelPath,
		TagName:    svc.TagName,
		ModelScore: svc.ModelScore,
		UserScore:  svc.UserScore,
	}
}
