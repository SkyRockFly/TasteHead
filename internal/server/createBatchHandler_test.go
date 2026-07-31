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

const fixtureCreateBatch = `testdata/fixtures/createBatch/rows.sql`

func TestCreateBatchHandler(t *testing.T) {
	type wantReq struct {
		body string
	}
	type wantResp struct {
		code      int
		body      string
		batchName string
	}
	tests := []struct {
		name string
		req  wantReq
		want wantResp
	}{
		{
			name: "#01_OK",
			req: wantReq{
				body: `{"name":"lmao"}`,
			},
			want: wantResp{
				code:      http.StatusCreated,
				body:      `{"batch_id":2}`,
				batchName: "lmao",
			},
		},
		{
			name: "#02_INVALID_JSON",
			req: wantReq{
				body: `{t 3]}`,
			},
			want: wantResp{
				code: http.StatusBadRequest,
				body: `{"error":"invalid json"}`,
			},
		},
		{
			name: "#03_INVALID_REQ",
			req: wantReq{
				body: `{"name":""}`,
			},
			want: wantResp{
				code: http.StatusBadRequest,
				body: `{"error":"bad request"}`,
			},
		},
		{
			name: "#04_CREATE_ALREADY_EXIST",
			req: wantReq{
				body: `{"name":"kek"}`,
			},
			want: wantResp{
				code: http.StatusConflict,
				body: `{"error": "already exists"}`,
			},
		},
	}

	sut := middlewares.LogMiddleware(CreateBatchHandler(downloadSVC))

	method := http.MethodPost
	hndURL := "/batch/create"
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, testutil.LoadFixtures(pool, fixtureCreateBatch, resetALLFixtures))
			req := httptest.NewRequest(method, hndURL, strings.NewReader(tt.req.body))

			rr := httptest.NewRecorder()

			sut.ServeHTTP(rr, req)

			assert.Equal(t, tt.want.code, rr.Code)
			assert.JSONEq(t, tt.want.body, rr.Body.String())
			if tt.want.code == http.StatusCreated {
				checkName(t, tt.want.batchName)
				batchPath := filepath.Join(svcPaths.DownloadDir, tt.want.batchName)
				_, err := os.Stat(batchPath)
				assert.NoError(t, err)
			}
		})
	}
}
