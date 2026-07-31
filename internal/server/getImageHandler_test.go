package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"scraper/internal/pkg/middlewares"
	"scraper/internal/pkg/testutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetImageHandler(t *testing.T) {
	type wantReq struct {
		path string
	}
	type wantResp struct {
		code  int
		bytes int
		body  string
	}
	tests := []struct {
		name string
		req  wantReq
		want wantResp
	}{
		{
			name: "#01_OK",
			req: wantReq{
				path: "_testBatch/image.jpg",
			},
			want: wantResp{
				code:  http.StatusOK,
				bytes: 5390,
			},
		},
		{
			name: "#02_INVALID_REQ",
			req: wantReq{
				path: "",
			},
			want: wantResp{
				code: http.StatusBadRequest,
				body: `{"error":"bad request"}`,
			},
		},
		{
			name: "#03_NO_FILE",
			req: wantReq{
				path: "_testBatch/image45.jpg",
			},
			want: wantResp{
				code:  http.StatusNotFound,
				bytes: 19,
			},
		},
	}

	sut := middlewares.LogMiddleware(GetImageHandler(downloadSVC))

	testDir := filepath.Join(svcPaths.DownloadDir, "_testBatch")
	require.NoError(t, testutil.CreateTempDirFromPicFixtureDir(
		`testdata/fixtures/getImagesTest/images`,
		testDir,
	))

	method := http.MethodGet
	hndURL := "/images/get/pic?path="
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(method, hndURL+tt.req.path, nil)

			rr := httptest.NewRecorder()

			sut.ServeHTTP(rr, req)
			assert.Equal(t, tt.want.code, rr.Code)

			if tt.want.code == http.StatusBadRequest {
				assert.JSONEq(t, tt.want.body, rr.Body.String())
				return
			}

			assertBytes(t, tt.want.bytes, rr.Body.Bytes())
		})
	}

	require.NoError(t, os.RemoveAll(testDir))
}

func assertBytes(t *testing.T, want int, got []byte) {
	t.Helper()

	assert.Equal(t, want, len(got))
}
