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
	listScoreCompositionsFixture = `testdata\fixtures\listScoresCompositions\rows.sql`
)

func Test_listScoreCompositionsHandler(t *testing.T) {
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
				body: `{"tag_id":1}`,
			},
			want: wantResp{
				code: http.StatusOK,
				body: `[{"count":2, "score":"0.00"},
{"count":2, "score":"0.25"},
{"count":2, "score":"0.50"},
{"count":2, "score":"0.75"},
{"count":2, "score":"1.00"}]`,
			},
		},
		{
			name: "#02_BAD_JSON",
			req: wantReq{
				body: `{"tag_:1}`,
			},
			want: wantResp{
				code: http.StatusBadRequest,
				body: `{"error":"invalid json"}`,
			},
		},
		{
			name: "#03_BAD_REQUEST",
			req: wantReq{
				body: `{"tag_id":0}`,
			},
			want: wantResp{
				code: http.StatusBadRequest,
				body: `{"error":"bad request"}`,
			},
		},
		{
			name: "#04_NO_TAGS",
			req: wantReq{
				body: `{"tag_id":2}`,
			},
			want: wantResp{
				code: http.StatusNotFound,
				body: `{"error":"not found"}`,
			},
		},
	}

	sut := middlewares.LogMiddleware(listScoreCompositionsHandler(trainingService))

	method := http.MethodPost
	hndURL := "/training/scores/composition"
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, testutil.LoadFixtures(pool, listScoreCompositionsFixture, resetALLFixtures))
			req := httptest.NewRequest(method, hndURL, strings.NewReader(tt.req.body))

			rr := httptest.NewRecorder()

			sut.ServeHTTP(rr, req)
			assert.Equal(t, tt.want.code, rr.Code)
			assert.JSONEq(t, tt.want.body, rr.Body.String())
		})
	}
}
