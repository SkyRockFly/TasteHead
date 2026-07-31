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

const (
	fixtureTrainModel = `testdata/fixtures/trainModel/rows.sql`
)

func TestTrainModelHandler(t *testing.T) {
	type wantReq struct {
		body string
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
			name: "01_OK",
			req: wantReq{
				body: `{"model_name":"test1","tag_id":1}`,
			},
			want: wantResp{
				code: http.StatusCreated,
				body: `{"trained":true}`,
			},
		},
		{
			name: "#02_INVALID_JSON",
			req: wantReq{
				body: `{s}`,
			},
			want: wantResp{
				code: http.StatusBadRequest,
				body: `{"error":"invalid json"}`,
			},
		},
		{
			name: "#03_INVALID_REQ",
			req: wantReq{
				body: `{"model_name":"","tag_id":0}`,
			},
			want: wantResp{
				code: http.StatusBadRequest,
				body: `{"error":"bad request"}`,
			},
		},
		{
			name: "#04_NO_ID",
			req: wantReq{
				body: `{"model_name":"test1","tag_id":5}`,
			},
			want: wantResp{
				code: http.StatusNotFound,
				body: `{"error":"not found"}`,
			},
		},
		{
			name: "#05_NO_IMAGES",
			req: wantReq{
				body: `{"model_name":"test1","tag_id":2}`,
			},
			want: wantResp{
				code: http.StatusNotFound,
				body: `{"error":"not found"}`,
			},
		},
	}

	sut := middlewares.LogMiddleware(TrainModelHandler(downloadSVC))

	method := http.MethodPost
	hndURL := "/model/train"
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, testutil.ClearDir(svcPaths.DownloadDir))
			require.NoError(t, testutil.LoadFixtures(pool, fixtureTrainModel, resetALLFixtures))
			require.NoError(t, testutil.CreateTempDirFromPicFixtureDir(`testdata/fixtures/trainModel/images`,
				filepath.Join(svcPaths.DownloadDir, "batch")))

			req := httptest.NewRequest(method, hndURL, strings.NewReader(tt.req.body))

			rr := httptest.NewRecorder()

			sut.ServeHTTP(rr, req)
			assert.Equal(t, tt.want.code, rr.Code)
			assert.JSONEq(t, tt.want.body, rr.Body.String())
			if tt.want.code == http.StatusOK {
				assert.Equal(t, true, isModelExist(t, filepath.Join(svcPaths.ModelDir, "test1.pt")))
			}
		})
		require.NoError(t, testutil.ClearDir(svcPaths.DownloadDir))
		require.NoError(t, testutil.ClearDir(svcPaths.ModelDir))
	}
}

func isModelExist(t *testing.T, modelPath string) bool {
	t.Helper()
	file, err := os.Stat(modelPath)
	if err != nil {
		return false
	}
	if file.IsDir() {
		return false
	}
	return true
}
