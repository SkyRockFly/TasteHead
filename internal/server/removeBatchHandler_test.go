package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"scraper/internal/pkg/middlewares"
	"scraper/internal/pkg/testutil"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const fixtureRemoveBatch = `testdata\fixtures\removeBatch\rows.sql`

func TestRemoveBatchHandler(t *testing.T) {
	type wantReq struct {
		body string
	}
	type wantResp struct {
		code            int
		body            string
		batchsRemaining int64
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
				code:            http.StatusOK,
				body:            `{"accepted":true}`,
				batchsRemaining: 1,
			},
		},
		{
			name: "#02_INVALID_JSON",
			req: wantReq{
				body: `{s}`,
			},
			want: wantResp{
				code:            http.StatusBadRequest,
				body:            `{"error":"invalid json"}`,
				batchsRemaining: 2,
			},
		},
		{
			name: "#03_INVALID_REQ",
			req: wantReq{
				body: `{"id":0}`,
			},
			want: wantResp{
				code:            http.StatusBadRequest,
				body:            `{"error":"bad request"}`,
				batchsRemaining: 2,
			},
		},
		{
			name: "#04_DELETED_ID",
			req: wantReq{
				body: `{"id":2}`,
			},
			want: wantResp{
				code:            http.StatusNotFound,
				body:            `{"error":"not found"}`,
				batchsRemaining: 2,
			},
		},
		{
			name: "#05_NO_ID",
			req: wantReq{
				body: `{"id":23}`,
			},
			want: wantResp{
				code:            http.StatusNotFound,
				body:            `{"error":"not found"}`,
				batchsRemaining: 2,
			},
		},
	}

	sut := middlewares.LogMiddleware(RemoveBatchHandler(imagerowService))

	method := http.MethodDelete
	hndURL := "/batch/remove"
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, testutil.LoadFixtures(pool, fixtureRemoveBatch, resetALLFixtures))
			req := httptest.NewRequest(method, hndURL, strings.NewReader(tt.req.body))

			rr := httptest.NewRecorder()

			sut.ServeHTTP(rr, req)
			assert.Equal(t, tt.want.code, rr.Code)
			assert.JSONEq(t, tt.want.body, rr.Body.String())
			assert.Equal(t, countUndeletedBatchesDB(t), tt.want.batchsRemaining)
		})
	}
}

func countUndeletedBatchesDB(t *testing.T) int64 {
	t.Helper()
	sql := `SELECT COUNT(*) FROM batch WHERE deleted_at IS NULL`

	var count int64
	require.NoError(t, pool.QueryRow(context.Background(), sql).Scan(&count))

	return count
}
