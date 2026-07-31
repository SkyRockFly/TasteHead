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

const fixtureRestoreImages = `testdata/fixtures/restoreImages/rows.sql`

func TestRestoreImagesHandler(t *testing.T) {
	type wantReq struct {
		body string
	}
	type wantResp struct {
		code            int
		body            string
		imagesUndeleted int64
	}
	tests := []struct {
		name string
		req  wantReq
		want wantResp
	}{
		{
			name: "#01_OK",
			req: wantReq{
				body: `{"ids":[1,3,4,5]}`,
			},
			want: wantResp{
				code:            http.StatusOK,
				body:            `{"accepted":true}`,
				imagesUndeleted: 4,
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
				imagesUndeleted: 1,
			},
		},
		{
			name: "#03_INVALID_REQ",
			req: wantReq{
				body: `{"ids":[0]}`,
			},
			want: wantResp{
				code:            http.StatusBadRequest,
				body:            `{"error":"bad request"}`,
				imagesUndeleted: 1,
			},
		},
		{
			name: "#04_UNDELETED_ID",
			req: wantReq{
				body: `{"ids":[1]}`,
			},
			want: wantResp{
				code:            http.StatusNotFound,
				body:            `{"error":"not found"}`,
				imagesUndeleted: 1,
			},
		},
		{
			name: "#05_NO_ID",
			req: wantReq{
				body: `{"ids":[23]}`,
			},
			want: wantResp{
				code:            http.StatusNotFound,
				body:            `{"error":"not found"}`,
				imagesUndeleted: 1,
			},
		},
	}

	sut := middlewares.LogMiddleware(RestoreImagesHandler(imagerowService))

	method := http.MethodPut
	hndURL := "/image/restore"
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, testutil.LoadFixtures(pool, fixtureRestoreImages, resetALLFixtures))
			req := httptest.NewRequest(method, hndURL, strings.NewReader(tt.req.body))

			rr := httptest.NewRecorder()

			sut.ServeHTTP(rr, req)
			assert.Equal(t, tt.want.code, rr.Code)
			assert.JSONEq(t, tt.want.body, rr.Body.String())
			assert.Equal(t, countUndeletedImagesDB(t), tt.want.imagesUndeleted)
		})
	}
}

func countUndeletedImagesDB(t *testing.T) int64 {
	t.Helper()
	sql := `SELECT COUNT(*) FROM download WHERE deleted_at IS NULL`

	var count int64
	require.NoError(t, pool.QueryRow(context.Background(), sql).Scan(&count))

	return count
}
