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

const fixtureUpdateBatchName = `testdata/fixtures/updateBatchName/rows.sql`

func TestUpdateBatchNameHandler(t *testing.T) {
	type wantReq struct {
		body string
	}
	type wantResp struct {
		code     int
		body     string
		wantName string
	}
	tests := []struct {
		name string
		req  wantReq
		want wantResp
	}{
		{
			name: "#01_OK",
			req: wantReq{
				body: `{"batch_id":1,"batch_name":"lmao"}`,
			},
			want: wantResp{
				code:     http.StatusOK,
				body:     `{"accepted":true}`,
				wantName: "lmao",
			},
		},
		{
			name: "#02_INVALID_REQ",
			req: wantReq{
				body: `{"batch_id":0,"batch_name":""}`,
			},
			want: wantResp{
				code: http.StatusBadRequest,
				body: `{"error":"bad request"}`,
			},
		},
		{
			name: "#03_UPDATE_NON_EXISTING",
			req: wantReq{
				body: `{"batch_id":5,"batch_name":"lmao"}`,
			},
			want: wantResp{
				code: http.StatusNotFound,
				body: `{"error":"not found"}`,
			},
		},
		{
			name: "#04_UPDATE_DELETED",
			req: wantReq{
				body: `{"batch_id":4,"batch_name":"lmao"}`,
			},
			want: wantResp{
				code: http.StatusNotFound,
				body: `{"error":"not found"}`,
			},
		},
		{
			name: "#05_UPDATE_NAME_ALREADY_EXIST",
			req: wantReq{
				body: `{"batch_id":3,"batch_name":"kek2"}`,
			},
			want: wantResp{
				code: http.StatusConflict,
				body: `{"error":"already exists"}`,
			},
		},
		{
			name: "#06_INVALID_JSON",
			req: wantReq{
				body: `{"batch_id":3batch_name":"kek2"}`,
			},
			want: wantResp{
				code: http.StatusBadRequest,
				body: `{"error":"invalid json"}`,
			},
		},
	}

	sut := middlewares.LogMiddleware(UpdateBatchNameHandler(imagerowService))

	method := http.MethodPut
	hndURL := "/batch/update/name"
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, testutil.LoadFixtures(pool, fixtureUpdateBatchName, resetALLFixtures))

			req := httptest.NewRequest(method, hndURL, strings.NewReader(tt.req.body))

			rr := httptest.NewRecorder()

			sut.ServeHTTP(rr, req)
			assert.Equal(t, tt.want.code, rr.Code)
			assert.JSONEq(t, tt.want.body, rr.Body.String())
			if tt.want.code == http.StatusOK {
				checkName(t, tt.want.wantName)
			}
		})
	}
}

func checkName(t *testing.T, name string) {
	t.Helper()
	sql := `SELECT (id) FROM batch WHERE name = $1 AND deleted_at IS NULL AND status = 'finished'`
	var id int64
	err := pool.QueryRow(context.Background(), sql, name).Scan(&id)
	require.NoError(t, err)
}
