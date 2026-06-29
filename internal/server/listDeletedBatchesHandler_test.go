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

const fixtureListDeletedBatches = `testdata\fixtures\listDeletedBatches\rows.sql`

func TestListDeletedBatchesHandler(t *testing.T) {
	type wantResp struct {
		code             int
		body             string
		batchesRemaining int64
	}
	tests := []struct {
		name string
		want wantResp
	}{
		{
			name: "#01_OK",
			want: wantResp{
				code: http.StatusOK,
				body: `{"batches":[
{"deleted_at":"2026-04-12T18:30:00Z", "id":2, "name":"kek2", "rel_path":"kek2"},
{"deleted_at":"2026-04-12T18:30:00Z", "id":3, "name":"kek4", "rel_path":"kek4"}]}`,
				batchesRemaining: 3,
			},
		},
	}

	sut := middlewares.LogMiddleware(ListDeletedBatchesHandler(imagerowService))

	method := http.MethodGet
	hndURL := "/batch/list/deleted"
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, testutil.LoadFixtures(pool, fixtureListDeletedBatches, resetALLFixtures))
			req := httptest.NewRequest(method, hndURL, nil)

			rr := httptest.NewRecorder()

			sut.ServeHTTP(rr, req)
			assert.Equal(t, tt.want.code, rr.Code)
			assert.JSONEq(t, tt.want.body, rr.Body.String())
			assert.Equal(t, countBatchesDB(t), tt.want.batchesRemaining)
		})
	}
}
