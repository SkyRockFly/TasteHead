package server

import (
	"fmt"
	"net/http"
	trainingsvc "scraper/internal/service/training"
)

type ScoreCompositionDTO struct {
	TagID int64 `json:"tag_id"`
}

type ScoreCompositionItem struct {
	Score string `json:"score"`
	Count int64  `json:"count"`
}

const listScoreCompositionsName = "scoreCompositionsHandler"

func listScoreCompositionsHandler(service *trainingsvc.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := getCtxLogger(ctx)
		var dto ScoreCompositionDTO
		if err := decodeJSON(&dto, r); err != nil {
			handleError(w, fmt.Errorf("decode:%w", err), logger, listScoreCompositionsName)
			return
		}

		svcReq := trainingsvc.ListRowsScoresCompositionReq{
			TagID: dto.TagID,
		}

		compositions, err := service.ListRowsScoresComposition(ctx, svcReq)
		if err != nil {
			handleError(w, fmt.Errorf("UpdateTagName:%w", err), logger, listScoreCompositionsName)
			return
		}

		resp := remapScoreCompositionItemToDTO(compositions)

		writeJSON(w, http.StatusOK, logger, resp)
	}
}

func remapScoreCompositionItemToDTO(svc []trainingsvc.ScoreCompositionItem) []ScoreCompositionItem {
	dtoItems := make([]ScoreCompositionItem, 0, len(svc))
	for _, item := range svc {
		dtoItem := ScoreCompositionItem{
			Score: item.Score,
			Count: item.Count,
		}
		dtoItems = append(dtoItems, dtoItem)
	}

	return dtoItems
}
