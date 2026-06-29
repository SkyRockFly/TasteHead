package server

import (
	"fmt"
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

func TestSetModelHandler(t *testing.T) {
	srv := launchTestServer(t)
	defer srv.Close()

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
			name: "#01_OK",
			req: wantReq{
				body: `{"model_name":"15_06_2026.pt"}`,
			},
			want: wantResp{
				code: http.StatusOK,
				body: `{"accepted":true}`,
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
				body: `{"model_name":""}`,
			},
			want: wantResp{
				code: http.StatusBadRequest,
				body: `{"error":"bad request"}`,
			},
		},
		{
			name: "#04_BAD_NAME",
			req: wantReq{
				body: `{"model_name":" model.pt"}`,
			},
			want: wantResp{
				code: http.StatusBadRequest,
				body: `{"error":"bad request"}`,
			},
		},
		{
			name: "#05_BAD_NAME",
			req: wantReq{
				body: `{"model_name":"../model.pt"}`,
			},
			want: wantResp{
				code: http.StatusBadRequest,
				body: `{"error":"bad request"}`,
			},
		},
		{
			name: "#06_BAD_NAME",
			req: wantReq{
				body: `{"model_name":"models/model.pt"}`,
			},
			want: wantResp{
				code: http.StatusBadRequest,
				body: `{"error":"bad request"}`,
			},
		},
	}

	sut := middlewares.LogMiddleware(SetModelHandler(downloadSVC))

	method := http.MethodPost
	hndURL := "/model/set"
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, testutil.ClearDir(svcPaths.ModelDir))
			require.NoError(t, copyFilesFromFixture(`testdata\fixtures\setModel`,
				svcPaths.ModelDir))

			req := httptest.NewRequest(method, hndURL, strings.NewReader(tt.req.body))
			rr := httptest.NewRecorder()

			sut.ServeHTTP(rr, req)
			assert.Equal(t, tt.want.code, rr.Code)
			assert.JSONEq(t, tt.want.body, rr.Body.String())
		})
	}
}

func copyFilesFromFixture(src string, dest string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("readDir: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		srcPath := filepath.Join(src, entry.Name())
		destPath := filepath.Join(dest, entry.Name())
		bytes, err := os.ReadFile(srcPath)
		if err != nil {
			return fmt.Errorf("ReadFile: %w", err)
		}
		if err := os.WriteFile(destPath, bytes, 0o655); err != nil {
			return fmt.Errorf("write file: %w", err)
		}

	}

	return nil
}
