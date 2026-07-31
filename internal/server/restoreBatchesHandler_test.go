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

const fixtureRestoreBatches = `testdata/fixtures/restoreBatches/rows.sql`

func TestRestoreBatchesHandler(t *testing.T) {
	type wantReq struct {
		body string
	}
	type wantResp struct {
		code             int
		body             string
		batchesUndeleted int64
	}
	tests := []struct {
		name string
		req  wantReq
		want wantResp
	}{
		{
			name: "#01_OK",
			req: wantReq{
				body: `{"ids":[1,2,3]}`,
			},
			want: wantResp{
				code:             http.StatusOK,
				body:             `{"accepted":true}`,
				batchesUndeleted: 3,
			},
		},
		{
			name: "#02_INVALID_JSON",
			req: wantReq{
				body: `{s}`,
			},
			want: wantResp{
				code:             http.StatusBadRequest,
				body:             `{"error":"invalid json"}`,
				batchesUndeleted: 1,
			},
		},
		{
			name: "#03_INVALID_REQ",
			req: wantReq{
				body: `{"ids":[0]}`,
			},
			want: wantResp{
				code:             http.StatusBadRequest,
				body:             `{"error":"bad request"}`,
				batchesUndeleted: 1,
			},
		},
		{
			name: "#04_UNDELETED_ID",
			req: wantReq{
				body: `{"ids":[1]}`,
			},
			want: wantResp{
				code:             http.StatusNotFound,
				body:             `{"error":"not found"}`,
				batchesUndeleted: 1,
			},
		},
		{
			name: "#05_NO_ID",
			req: wantReq{
				body: `{"ids":[23]}`,
			},
			want: wantResp{
				code:             http.StatusNotFound,
				body:             `{"error":"not found"}`,
				batchesUndeleted: 1,
			},
		},
	}

	sut := middlewares.LogMiddleware(RestoreBatchesHandler(imagerowService))

	method := http.MethodPut
	hndURL := "/batch/restore"
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, testutil.LoadFixtures(pool, fixtureRestoreBatches, resetALLFixtures))
			req := httptest.NewRequest(method, hndURL, strings.NewReader(tt.req.body))

			rr := httptest.NewRecorder()

			sut.ServeHTTP(rr, req)
			assert.Equal(t, tt.want.code, rr.Code)
			assert.JSONEq(t, tt.want.body, rr.Body.String())
			assert.Equal(t, countUndeletedBatchesDB(t), tt.want.batchesUndeleted)
		})
	}
}
