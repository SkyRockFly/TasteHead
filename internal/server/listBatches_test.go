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

func TestListBatchesHandler(t *testing.T) {
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
				fixture: `testdata/fixtures/listBatches/finishedBatches.sql`,
			},
			want: wantResp{
				code: http.StatusOK,
				body: `{"batches":[
{"id":1, "name":"kek", "rel_path":"kek"},
{"id":2, "name":"kek2", "rel_path":"kek2"},
{"id":3, "name":"kek3", "rel_path":"kek3"}]}`,
			},
		},
		{
			name: "#02_NO_FINISHED_BATCHES",
			req: wantReq{
				fixture: `testdata/fixtures/listBatches/pending batches.sql`,
			},
			want: wantResp{
				code: http.StatusNotFound,
				body: `{"error":"not found"}`,
			},
		},
	}

	sut := middlewares.LogMiddleware(ListBatchesHandler(imagerowService))

	method := http.MethodGet
	hndURL := "/batch/list"
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
