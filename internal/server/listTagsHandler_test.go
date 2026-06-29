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

func TestListTagsHandler(t *testing.T) {
	type wantReq struct {
		fixture string
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
				fixture: `testdata\fixtures\listTags\finishedTags.sql`,
			},
			want: wantResp{
				code: http.StatusOK,
				body: `{"tags":[
{"id":1, "name":"kek", "desc":"kek"},
{"id":2, "name":"kek2", "desc":"kek"}]}`,
			},
		},
		{
			name: "#02_NO_DELETEG_TAGS",
			req: wantReq{
				fixture: `testdata\fixtures\listTags\deletedTags.sql`,
			},
			want: wantResp{
				code: http.StatusOK,
				body: `{"tags":[]}`,
			},
		},
	}

	sut := middlewares.LogMiddleware(ListTagsHandler(trainingService))

	method := http.MethodGet
	hndURL := "/tag/list"
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, testutil.LoadFixtures(pool, tt.req.fixture, resetALLFixtures))
			req := httptest.NewRequest(method, hndURL, nil)

			rr := httptest.NewRecorder()

			sut.ServeHTTP(rr, req)
			assert.Equal(t, tt.want.code, rr.Code)
			assert.JSONEq(t, tt.want.body, rr.Body.String())
		})
	}
}
