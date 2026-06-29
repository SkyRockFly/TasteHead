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

const fixtureGetScrapeState = `testdata\fixtures\getScrapeState\batches.sql`

func TestGetScrapeStateHandler(t *testing.T) {
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
				body: `{"id":2}`,
			},
			want: wantResp{
				code: http.StatusOK,
				body: `{"image_attr":"data-file-url", 
"next_page_attr":"href", "next_page_selector":"a[id*='paginator-next']", "next_url":"http://127.0.0.1:49981/posts?page=4", "post_selector":"article"}`,
			},
		},
		{
			name: "#02_INVALID_REQ",
			req: wantReq{
				body: `{"id":0}`,
			},
			want: wantResp{
				code: http.StatusBadRequest,
				body: `{"error":"bad request"}`,
			},
		},
		{
			name: "#03_NO_STATE",
			req: wantReq{
				body: `{"id":1}`,
			},
			want: wantResp{
				body: `{"error":"not found"}`,
				code: http.StatusNotFound,
			},
		},
	}

	sut := middlewares.LogMiddleware(GetScrapeStateHandler(downloadSVC))

	method := http.MethodPost
	hndURL := "/scrape/get/state"
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(method, hndURL, strings.NewReader(tt.req.body))
			require.NoError(t, testutil.LoadFixtures(pool, fixtureGetScrapeState, resetALLFixtures))
			require.NoError(t, testutil.ClearDir(svcPaths.DownloadDir))
			require.NoError(t, testutil.CreateTempDirFromPicFixtureDir(
				`testdata\fixtures\getScrapeState\withoutState`,
				filepath.Join(svcPaths.DownloadDir, "withoutState"),
			))
			require.NoError(t, testutil.CreateTempDirFromPicFixtureDir(
				`testdata\fixtures\getScrapeState\withState`,
				filepath.Join(svcPaths.DownloadDir, "withState"),
			))

			rr := httptest.NewRecorder()

			sut.ServeHTTP(rr, req)
			assert.Equal(t, tt.want.code, rr.Code)
			assert.JSONEq(t, tt.want.body, rr.Body.String())
		})
	}
}
