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

const fixtureHardDeleteImages = `testdata/fixtures/hardDeleteImages/rows.sql`

func TestHardDeleteImagesHandler(t *testing.T) {
	type wantReq struct {
		body string
	}
	type wantResp struct {
		code        int
		body        string
		imagesCount int64
	}
	tests := []struct {
		name string
		req  wantReq
		want wantResp
	}{
		{
			name: "01_OK",
			req: wantReq{
				body: `{"ids":[1,2]}`,
			},
			want: wantResp{
				code:        http.StatusOK,
				body:        `{"reject":[]}`,
				imagesCount: 2,
			},
		},
		{
			name: "#02_INVALID_JSON",
			req: wantReq{
				body: `{s}`,
			},
			want: wantResp{
				code:        http.StatusBadRequest,
				body:        `{"error":"invalid json"}`,
				imagesCount: 4,
			},
		},
		{
			name: "#03_INVALID_REQ_1",
			req: wantReq{
				body: `{"ids":[]}`,
			},
			want: wantResp{
				code:        http.StatusBadRequest,
				body:        `{"error":"bad request"}`,
				imagesCount: 4,
			},
		},
		{
			name: "#04_INVALID_REQ_2",
			req: wantReq{
				body: `{"ids":[0,0,0]}`,
			},
			want: wantResp{
				code:        http.StatusBadRequest,
				body:        `{"error":"bad request"}`,
				imagesCount: 4,
			},
		},
		{
			name: "#05_PARTIALLY",
			req: wantReq{
				body: `{"ids":[3,4,5]}`,
			},
			want: wantResp{
				code:        http.StatusOK,
				body:        `{"reject":[]}`,
				imagesCount: 3,
			},
		},
	}

	sut := middlewares.LogMiddleware(HardDeleteImagesHandler(downloadSVC))

	method := http.MethodDelete
	hndURL := "/images/hardDelete"
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, testutil.LoadFixtures(pool, fixtureHardDeleteImages, resetALLFixtures))
			require.NoError(t, testutil.ClearDir(svcPaths.DownloadDir))

			testPath := filepath.Join(svcPaths.DownloadDir, "dir")
			require.NoError(t, testutil.CreateTempDirFromPicFixtureDir(
				`testdata/fixtures/hardDeleteImages/dir`,
				testPath,
			))
			req := httptest.NewRequest(method, hndURL, strings.NewReader(tt.req.body))

			rr := httptest.NewRecorder()

			sut.ServeHTTP(rr, req)
			assert.Equal(t, tt.want.code, rr.Code)
			assert.JSONEq(t, tt.want.body, rr.Body.String())
			assert.Equal(t, tt.want.imagesCount, countImagesDB(t))
			assert.Equal(t, tt.want.imagesCount, countFiles(t, testPath))
		})
	}
}

func countFiles(t *testing.T, src string) int64 {
	t.Helper()

	entries, err := os.ReadDir(src)
	require.NoError(t, err)
	var counter int64
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		counter++
	}

	return counter
}
