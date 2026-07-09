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
	fixtureUpdateTagName = `testdata/fixtures/updateTagName/rows.sql`
)

func TestUpdateTagNameHandler(t *testing.T) {
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
				body: `{"id":1,"update_name":"dataset4"}`,
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
				body: `{"id":0,"update_name":""}`,
			},
			want: wantResp{
				code: http.StatusBadRequest,
				body: `{"error":"bad request"}`,
			},
		},
		{
			name: "#04_NO_ID",
			req: wantReq{
				body: `{"id":23,"update_name":"dataset10"}`,
			},
			want: wantResp{
				code: http.StatusNotFound,
				body: `{"error":"not found"}`,
			},
		},
		{
			name: "#05_UPDATE_DELETED",
			req: wantReq{
				body: `{"id":3,"update_name":"dataset4"}`,
			},
			want: wantResp{
				code: http.StatusNotFound,
				body: `{"error":"not found"}`,
			},
		},
		{
			name: "#06_UPDATE_WITH_EXISTING_NAME",
			req: wantReq{
				body: `{"id":1,"update_name":"dataset2"}`,
			},
			want: wantResp{
				code: http.StatusConflict,
				body: `{"error":"trying to create duplicate of unique entity"}`,
			},
		},
	}

	sut := middlewares.LogMiddleware(UpdateTagNameHandler(trainingService))

	method := http.MethodPost
	hndURL := "/tag/update/name"
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, testutil.LoadFixtures(pool, fixtureUpdateTagName, resetALLFixtures))

			req := httptest.NewRequest(method, hndURL, strings.NewReader(tt.req.body))

			rr := httptest.NewRecorder()

			sut.ServeHTTP(rr, req)
			assert.Equal(t, tt.want.code, rr.Code)
			assert.JSONEq(t, tt.want.body, rr.Body.String())
		})
	}
}
