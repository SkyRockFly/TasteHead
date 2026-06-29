package server

import (
	"fmt"
	"net/http"
	"scraper/internal/service/download"
)

type MoveImagesDTO struct {
	IDs       []int64 `json:"ids"`
	ToBatchID int64   `json:"toBatchID"`
}

type MoveImagesResp struct {
	Rejects []Reject `json:"rejects"`
}

type Reject struct {
	RejectedID int64  `json:"rejected_id"`
	Reason     string `json:"reason"`
}

const moveImagesName = "moveImagesHandler"

func MoveImagesHandler(service *download.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := getCtxLogger(ctx)

		var dto MoveImagesDTO
		if err := decodeJSON(&dto, r); err != nil {
			handleError(w, fmt.Errorf("decode:%w", err), logger, moveImagesName)
			return
		}

		req := download.MoveImagesReq{
			IDs:       dto.IDs,
			ToBatchID: dto.ToBatchID,
		}

		svcResp, err := service.MoveImages(ctx, req)
		if err != nil {
			handleError(w, fmt.Errorf("MoveImagesHandler:%w", err), logger, moveImagesName)
			return
		}

		dtoRejects := remapRejectsToDTO(svcResp.Rejects)
		resp := MoveImagesResp{
			Rejects: dtoRejects,
		}

		writeJSON(w, http.StatusOK, logger, resp)
	}
}

func remapRejectsToDTO(svc []download.Reject) []Reject {
	dtoRejects := make([]Reject, 0, len(svc))
	var dtoReject Reject
	for _, reject := range svc {
		dtoReject = Reject{
			RejectedID: reject.RejectedID,
			Reason:     reject.Reason,
		}
		dtoRejects = append(dtoRejects, dtoReject)
	}

	return dtoRejects
}
