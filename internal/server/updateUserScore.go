package server

import (
	"fmt"
	"net/http"
	imagerowsvc "scraper/internal/service/imageRow"
)

type UpdateUserScore struct {
	ID    int64   `json:"id"`
	Score float32 `json:"score"`
}

type UpdateUserScoreDTO struct {
	Scores []UpdateUserScore `json:"scores"`
}

type UpdateUserScoreResp struct {
	Accepted bool `json:"accepted"`
}

const updateUserScoreName = "update user score handler"

func UpdateUserScoreHandler(service *imagerowsvc.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := getCtxLogger(ctx)

		var dto UpdateUserScoreDTO
		if err := decodeJSON(&dto, r); err != nil {
			handleError(w, fmt.Errorf("decode:%w", err), logger, updateUserScoreName)
			return
		}

		svcScores := remapDTOScorestoSVC(dto.Scores)
		req := imagerowsvc.UpdateUserScoreReq{
			Scores: svcScores,
		}

		if err := service.UpdateUserScore(ctx, req); err != nil {
			handleError(w, fmt.Errorf("svc.scrapeImages: %w", err), logger, updateUserScoreName)
			return
		}

		writeJSON(w, http.StatusOK, logger, UpdateUserScoreResp{Accepted: true})
	}
}

func remapDTOScorestoSVC(dtoScores []UpdateUserScore) []imagerowsvc.UpdateUserScore {
	svcScores := make([]imagerowsvc.UpdateUserScore, 0, len(dtoScores))
	var svcScore imagerowsvc.UpdateUserScore
	for _, dtoScore := range dtoScores {
		svcScore.ID = dtoScore.ID
		svcScore.Score = dtoScore.Score
		svcScores = append(svcScores, svcScore)
	}

	return svcScores
}
