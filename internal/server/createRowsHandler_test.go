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

const (
	fixtureRowsPath = "testdata/fixtures/createTrainingRowsTest/rows.sql"
)

func TestCreateTrainingRowsHandler(t *testing.T) {
	type wantReq struct {
		body string
	}
	type wantResp struct {
		code  int
		body  string
		files int64
	}
	tests := []struct {
		name string
		req  wantReq
		want wantResp
	}{
		{
			name: "#01_OK",
			req: wantReq{
				body: `{"tag_id":1,"image_ids": [1, 2, 3]}`,
			},
			want: wantResp{
				code:  http.StatusCreated,
				body:  `{"duplicates":[1, 2, 3]}`,
				files: 3,
			},
		},
		{
			name: "#02_INVALID_JSON",
			req: wantReq{
				body: `{tag_id":1,"image_ids": 1, 2, 3]}`,
			},
			want: wantResp{
				code:  http.StatusBadRequest,
				body:  `{"error":"invalid json"}`,
				files: 0,
			},
		},
		{
			name: "#03_INVALID_REQ",
			req: wantReq{
				body: `{"tag_id":0,"image_ids": []}`,
			},
			want: wantResp{
				code:  http.StatusBadRequest,
				body:  `{"error":"bad request"}`,
				files: 0,
			},
		},
		{
			name: "#04_UNIQUES_WITH_DUPLICATES",
			req: wantReq{
				body: `{"tag_id":1,"image_ids": [3,4,5]}`,
			},
			want: wantResp{
				code:  http.StatusCreated,
				body:  `{"duplicates": [3]}`,
				files: 5,
			},
		},
	}

	require.NoError(t, testutil.LoadFixtures(pool, fixtureRowsPath, resetALLFixtures))

	sut := middlewares.LogMiddleware(CreateTrainingRowsHandler(trainingService))

	method := http.MethodPost
	hndURL := "/training/create"
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(method, hndURL, strings.NewReader(tt.req.body))

			rr := httptest.NewRecorder()

			sut.ServeHTTP(rr, req)

			assert.Equal(t, tt.want.code, rr.Code)
			assert.JSONEq(t, tt.want.body, rr.Body.String())
			if tt.want.code == http.StatusCreated {
				assert.Equal(t, tt.want.files, getfilesFromDBByTag(t, 1))
			}
		})
	}
}

func getfilesFromDBByTag(t *testing.T, tagID int) int64 {
	query := `SELECT COUNT(*) FROM training WHERE tag_id = $1`
	var count int64
	err := pool.QueryRow(context.Background(), query, tagID).Scan(&count)
	require.NoError(t, err)

	return count
}
