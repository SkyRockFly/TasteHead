package server

import (
	"fmt"
	"net/http"
	"scraper/internal/service/download"
)

type FailedImport struct {
	Warn download.ImportIssue
	Path string
}

type ImportLocalDirsResp struct {
	Fails []FailedImport `json:"fails"`
}

const importLocalDirs = "Import local Dirs handler"

func ImportLocalDirsHandler(service *download.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := getCtxLogger(ctx)

		fails, err := service.ImportLocalDirs(ctx)
		if err != nil {
			handleError(w, fmt.Errorf("importLocalDirs:%w", err), logger, importLocalDirs)
			return
		}
		if fails != nil {
			resp := ImportLocalDirsResp{
				Fails: remapFailedImportToDTO(fails),
			}
			writeJSON(w, http.StatusOK, logger, resp)
			return
		}

		resp := ImportLocalDirsResp{
			Fails: nil,
		}

		writeJSON(w, http.StatusOK, logger, resp)
	}
}

func remapFailedImportToDTO(svc []download.FailedImport) []FailedImport {
	fails := make([]FailedImport, 0, len(svc))
	var fail FailedImport
	for _, svcFail := range svc {
		fail = FailedImport{
			Warn: svcFail.Warn,
			Path: svcFail.Path,
		}
		fails = append(fails, fail)
	}
	return fails
}
