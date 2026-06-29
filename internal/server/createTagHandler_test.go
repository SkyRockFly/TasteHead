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
	fixtureCreateTagsPath = "testdata/fixtures/createTrainingRowsTest/rows.sql"
)

func TestCreateTagHandler(t *testing.T) {
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
				body: `{"name":"dataset2","desc":"alolo"}`,
			},
			want: wantResp{
				code: http.StatusCreated,
				body: `{"accepted":true}`,
			},
		},
		{
			name: "#02_INVALID_JSON",
			req: wantReq{
				body: `{"name":dataset1","desc":"alolo"`,
			},
			want: wantResp{
				code: http.StatusBadRequest,
				body: `{"error":"invalid json"}`,
			},
		},
		{
			name: "#03_INVALID_REQ",
			req: wantReq{
				body: `{"name":"","desc":"alolo"}`,
			},
			want: wantResp{
				code: http.StatusBadRequest,
				body: `{"error":"bad request"}`,
			},
		},
		{
			name: "#04_DUPLICATE_NAME",
			req: wantReq{
				body: `{"name":"dataset1","desc":""}`,
			},
			want: wantResp{
				code: http.StatusConflict,
				body: `{"error":"trying to create duplicate of unique entity"}`,
			},
		},
	}

	require.NoError(t, testutil.LoadFixtures(pool, fixtureCreateTagsPath, resetALLFixtures))

	sut := middlewares.LogMiddleware(CreateTagHandler(trainingService))

	method := http.MethodPost
	hndURL := "/tag/create"
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(method, hndURL, strings.NewReader(tt.req.body))

			rr := httptest.NewRecorder()

			sut.ServeHTTP(rr, req)

			assert.Equal(t, tt.want.code, rr.Code)
			assert.JSONEq(t, tt.want.body, rr.Body.String())
		})
	}
}
