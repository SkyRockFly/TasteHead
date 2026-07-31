package server

import (
	"net/http"
	"net/http/httptest"
	"scraper/internal/pkg/middlewares"
	"scraper/internal/pkg/testutil"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	fixtureRemoveTag = `testdata/fixtures/removeTag/rows.sql`
)

func TestRemoveTagHandler(t *testing.T) {
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
				body: `{"id":1}`,
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
				body: `{"id":0}`,
			},
			want: wantResp{
				code: http.StatusBadRequest,
				body: `{"error":"bad request"}`,
			},
		},
		{
			name: "#04_DELETED_ID",
			req: wantReq{
				body: `{"id":2}`,
			},
			want: wantResp{
				code: http.StatusNotFound,
				body: `{"error":"not found"}`,
			},
		},
		{
			name: "#05_NO_ID",
			req: wantReq{
				body: `{"id":23}`,
			},
			want: wantResp{
				code: http.StatusNotFound,
				body: `{"error":"not found"}`,
			},
		},
	}

	sut := middlewares.LogMiddleware(RemoveTagHandler(trainingService))

	method := http.MethodDelete
	hndURL := "/tag/delete/"
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, testutil.LoadFixtures(pool, fixtureRemoveTag, resetALLFixtures))

			req := httptest.NewRequest(method, hndURL, strings.NewReader(tt.req.body))

			rr := httptest.NewRecorder()

			sut.ServeHTTP(rr, req)
			assert.Equal(t, tt.want.code, rr.Code)
			assert.JSONEq(t, tt.want.body, rr.Body.String())
		})
	}
}
