package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"scraper/internal/pkg/middlewares"
	"scraper/internal/pkg/testutil"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const fixtureHardDeleteBatches = `testdata/fixtures/hardDeleteBatches/rows.sql`

func TestHardDeleteBatchHandler(t *testing.T) {
	type wantReq struct {
		body string
	}
	type wantResp struct {
		code       int
		body       string
		batchCount int64
	}
	tests := []struct {
		name string
		req  wantReq
		want wantResp
	}{
		{
			name: "#01_OK",
			req: wantReq{
				body: `{"ids":[2,3]}`,
			},
			want: wantResp{
				code:       http.StatusOK,
				body:       `{"accepted":true}`,
				batchCount: 1,
			},
		},
		{
			name: "#02_INVALID_JSON",
			req: wantReq{
				body: `{s}`,
			},
			want: wantResp{
				code:       http.StatusBadRequest,
				body:       `{"error":"invalid json"}`,
				batchCount: 3,
			},
		},
		{
			name: "#03_INVALID_REQ",
			req: wantReq{
				body: `{"ids":[0]}`,
			},
			want: wantResp{
				code:       http.StatusBadRequest,
				body:       `{"error":"bad request"}`,
				batchCount: 3,
			},
		},
		{
			name: "#04_NOT_SOFT_DELETED_ID",
			req: wantReq{
				body: `{"ids":[1]}`,
			},
			want: wantResp{
				code:       http.StatusNotFound,
				body:       `{"error":"not found"}`,
				batchCount: 3,
			},
		},
	}

	sut := middlewares.LogMiddleware(HardDeleteBatchesHandler(downloadSVC))

	method := http.MethodDelete
	hndURL := "/batch/hardDelete"
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, testutil.LoadFixtures(pool, fixtureHardDeleteBatches, resetALLFixtures))
			require.NoError(t, testutil.ClearDir(svcPaths.DownloadDir))
			require.NoError(t, testutil.CreateTempDirFromPicFixtureDir(
				`testdata/fixtures/hardDeleteBatches/dir`,
				filepath.Join(svcPaths.DownloadDir, "dir"),
			))
			require.NoError(t, testutil.CreateTempDirFromPicFixtureDir(
				`testdata/fixtures/hardDeleteBatches/dir2`,
				filepath.Join(svcPaths.DownloadDir, "dir2"),
			))
			req := httptest.NewRequest(method, hndURL, strings.NewReader(tt.req.body))

			rr := httptest.NewRecorder()

			sut.ServeHTTP(rr, req)
			assert.Equal(t, tt.want.code, rr.Code)
			assert.JSONEq(t, tt.want.body, rr.Body.String())
			assert.Equal(t, tt.want.batchCount, countBatchesDB(t))

			dirPath := filepath.Join(svcPaths.DownloadDir, "dir")
			if tt.want.code == http.StatusOK {
				_, err := os.Stat(dirPath)
				assert.Error(t, err)
			}
		})
	}
}
