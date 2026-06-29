package server

import (
	"net/http"
	"net/http/httptest"
	"scraper/internal/pkg/middlewares"
	"scraper/internal/pkg/testutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListModelsHandler(t *testing.T) {
	type wantResp struct {
		code int
		body string
	}
	tests := []struct {
		name string
		want wantResp
	}{
		{
			name: "#01_OK",
			want: wantResp{
				code: http.StatusOK,
				body: `{"models":["15_06_2026.pt", "clip_head_v2.pt"]}`,
			},
		},
	}

	sut := middlewares.LogMiddleware(ListModelsHandler(downloadSVC))

	method := http.MethodGet
	hndURL := "/model/list"
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, testutil.ClearDir(svcPaths.ModelDir))
			require.NoError(t, copyFilesFromFixture(`testdata/fixtures/listModels`,
				svcPaths.ModelDir))

			req := httptest.NewRequest(method, hndURL, nil)
			rr := httptest.NewRecorder()

			sut.ServeHTTP(rr, req)
			assert.Equal(t, tt.want.code, rr.Code)
			assert.JSONEq(t, tt.want.body, rr.Body.String())
		})
	}
}
