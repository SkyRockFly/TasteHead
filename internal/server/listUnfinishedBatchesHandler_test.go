package server

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"scraper/internal/pkg/middlewares"
	"scraper/internal/pkg/testutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListUnfinishedBatchesHandler(t *testing.T) {
	type wantReq struct {
		prepareDir func(t *testing.T)
	}
	type wantResp struct {
		code int
		body string
	}
	tests := []struct {
		name string
		req  wantReq
		want wantResp
	}{
		{
			name: "#01_OK",
			req: wantReq{
				prepareDir: func(t *testing.T) {
					t.Helper()
					fixture := `testdata\fixtures\listUnfinishedBatches\okDir\batchWithPages`
					dest := filepath.Join(svcPaths.DownloadDir, "batchWithPages")
					require.NoError(t, testutil.CreateTempDirFromPicFixtureDir(fixture, dest))

					fixture = `testdata\fixtures\listUnfinishedBatches\okDir\okBatch`
					dest = filepath.Join(svcPaths.DownloadDir, "okBatch")
					require.NoError(t, testutil.CreateTempDirFromPicFixtureDir(fixture, dest))

					fixture = `testdata\fixtures\listUnfinishedBatches\okDir\unprocessedBatch`
					dest = filepath.Join(svcPaths.DownloadDir, "unprocessedBatch")
					require.NoError(t, testutil.CreateTempDirFromPicFixtureDir(fixture, dest))
				},
			},
			want: wantResp{
				code: http.StatusOK,
				body: `{"batches":["batchWithPages", "unprocessedBatch"]}`,
			},
		},
		{
			name: "#02_NO_DIR",
			req: wantReq{
				prepareDir: nil,
			},
			want: wantResp{
				code: http.StatusNotFound,
				body: `{"error":"not found"}`,
			},
		},
	}

	sut := middlewares.LogMiddleware(ListUnfinishedBatchesHandler(downloadSVC))

	method := http.MethodGet
	hndURL := "/batches/list/unfinished"
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, testutil.ClearDir(svcPaths.DownloadDir))
			if tt.req.prepareDir != nil {
				tt.req.prepareDir(t)
			}
			req := httptest.NewRequest(method, hndURL, nil)

			rr := httptest.NewRecorder()

			sut.ServeHTTP(rr, req)

			assert.Equal(t, tt.want.code, rr.Code)
			assert.JSONEq(t, tt.want.body, rr.Body.String())
		})
	}
}
