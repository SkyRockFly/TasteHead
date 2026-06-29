package server

import (
	"fmt"
	"net/http"
	imagerowsvc "scraper/internal/service/imageRow"
	"time"
)

type DeletedRow struct {
	ID             int64      `json:"id"`
	Hash           string     `json:"hash"`
	BatchID        int64      `json:"batch_id"`
	RelPath        string     `json:"rel_path"`
	ModelScore     float32    `json:"model_score"`
	UserScore      *float32   `json:"user_score"`
	ImageDeletedAt *time.Time `json:"image_deleted_at"`
	BatchDeletedAt *time.Time `json:"batch_deleted_at"`
}

type ListDeletedImagesDTO struct {
	Limit           int        `json:"limit"`
	CursorID        int64      `json:"cursor_id"`
	CursorDeletedAt *time.Time `json:"cursor_deleted_at"`
	Next            bool       `json:"next"`
}

type DeletedCursor struct {
	CursorID        int64      `json:"cursor_id"`
	CursorDeletedAt *time.Time `json:"cursor_deleted_at"`
}

type ListDeletedImagesResp struct {
	Images     []DeletedRow  `json:"images"`
	CursorNext DeletedCursor `json:"cursor_next"`
	CursorPrev DeletedCursor `json:"cursor_prev"`
	HasMore    bool          `json:"has_more"`
}

const listDeletedImagesName = "ListDeletedImagesHandler"

func ListDeletedImagesHandler(service *imagerowsvc.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := getCtxLogger(ctx)

		var dto ListDeletedImagesDTO
		if err := decodeJSON(&dto, r); err != nil {
			handleError(w, fmt.Errorf("decode:%w", err), logger, listDeletedImagesName)
			return
		}

		req := imagerowsvc.ListDeletedImagesReq{
			CursorID:        dto.CursorID,
			CursorDeletedAt: dto.CursorDeletedAt,
			Limit:           dto.Limit,
			Next:            dto.Next,
		}

		svcResp, err := service.ListDeletedImages(ctx, req)
		if err != nil {
			handleError(w, fmt.Errorf("svc.ListDeletedImages: %w", err), logger, listDeletedImagesName)
			return
		}
		dtoImages := remapDeletedRowsToDTO(svcResp.Images)

		resp := ListDeletedImagesResp{
			Images:     dtoImages,
			CursorNext: remapDeletedCursorToDTO(svcResp.CursorNext),
			CursorPrev: remapDeletedCursorToDTO(svcResp.CursorPrev),
			HasMore:    svcResp.HasMore,
		}

		writeJSON(w, http.StatusOK, logger, resp)
	}
}

func remapDeletedCursorToDTO(svc imagerowsvc.DeletedCursor) DeletedCursor {
	return DeletedCursor{
		CursorID:        svc.CursorID,
		CursorDeletedAt: svc.CursorDeletedAt,
	}
}

func remapDeletedRowsToDTO(svc []imagerowsvc.DeletedRow) []DeletedRow {
	dtoRows := make([]DeletedRow, 0, len(svc))
	var dtoRow DeletedRow
	for _, row := range svc {
		dtoRow = DeletedRow{
			ID:             row.ID,
			Hash:           row.Hash,
			BatchID:        row.BatchID,
			RelPath:        row.RelPath,
			ModelScore:     row.ModelScore,
			UserScore:      row.UserScore,
			ImageDeletedAt: row.ImageDeletedAt,
			BatchDeletedAt: row.BatchDeletedAt,
		}
		dtoRows = append(dtoRows, dtoRow)
	}

	return dtoRows
}
