package server

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"scraper/internal/pkg/middlewares"
	"scraper/internal/pkg/testutil"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const moveImagesFixture = `testdata/fixtures/moveFilesTest/rows.sql`

func TestMoveImagesHandler(t *testing.T) {
	type wantReq struct {
		body string
	}
	type wantResp struct {
		code       int
		body       string
		filesInDir int64
		filesInDB  int64
	}
	tests := []struct {
		name string
		req  wantReq
		want wantResp
	}{
		{
			name: "01_OK",
			req: wantReq{
				body: `{"ids":[2,3], "toBatchID":1}`,
			},
			want: wantResp{
				code:       http.StatusOK,
				body:       `{"rejects":[]}`,
				filesInDir: 3,
				filesInDB:  3,
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
				filesInDir: 1,
				filesInDB:  1,
			},
		},
		{
			name: "#03_INVALID_REQ",
			req: wantReq{
				body: `{"ids":[0],"toBatchID":0}`,
			},
			want: wantResp{
				code:       http.StatusBadRequest,
				body:       `{"error":"bad request"}`,
				filesInDir: 1,
				filesInDB:  1,
			},
		},
		{
			name: "#04_PARTIAL",
			req: wantReq{
				body: `{"ids":[2,5,7],"toBatchID":1}`,
			},
			want: wantResp{
				code:       http.StatusOK,
				body:       `{"rejects":[{"reason":"not found", "rejected_id":5},{"reason":"not found", "rejected_id":7}]}`,
				filesInDir: 2,
				filesInDB:  2,
			},
		},
	}

	sut := middlewares.LogMiddleware(MoveImagesHandler(downloadSVC))

	method := http.MethodPost
	hndURL := "/images/move"
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, testutil.LoadFixtures(pool, moveImagesFixture, resetALLFixtures))
			require.NoError(t, testutil.ClearDir(svcPaths.DownloadDir))
			require.NoError(t, testutil.CreateTempDirFromPicFixtureDir(
				`testdata/fixtures/moveFilesTest/dir1`,
				filepath.Join(svcPaths.DownloadDir, "dir1"),
			))
			require.NoError(t, testutil.CreateTempDirFromPicFixtureDir(
				`testdata/fixtures/moveFilesTest/dir2`,
				filepath.Join(svcPaths.DownloadDir, "dir2"),
			))
			req := httptest.NewRequest(method, hndURL, strings.NewReader(tt.req.body))

			rr := httptest.NewRecorder()

			sut.ServeHTTP(rr, req)
			assert.Equal(t, tt.want.code, rr.Code)
			assert.JSONEq(t, tt.want.body, rr.Body.String())
			assert.Equal(t, getFilesCountFromDB(t, 1), tt.want.filesInDB)
			dirOnePath := filepath.Join(svcPaths.DownloadDir, "dir1")
			dirOneThumbsPath := filepath.Join(svcPaths.DownloadDir, "dir1", "thumbs")
			assert.Equal(t, tt.want.filesInDir, getFilesCountByDir(t, dirOnePath))
			assert.Equal(t, tt.want.filesInDir, getFilesCountByDir(t, dirOneThumbsPath))
		})
	}
}
