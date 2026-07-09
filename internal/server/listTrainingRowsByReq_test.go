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
	listTrainingRowsFixture = `testdata/fixtures/listTrainingRows/rows.sql`
)

func Test_listTrainingRowsByReqHandler(t *testing.T) {
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
				body: `{
"tag_id": 1,
"user_score": 1.00,
"limit": 20,
"cursor": 0,
"next":true
}`,
			},
			want: wantResp{
				code: http.StatusOK,
				body: `{"cursor_next":1,"cursor_prev":1, "has_more":false, "rows": 
[{"batch_id":1, "hash":"kekeke1", "id":1, "model_score":1, "rel_path":"lol.png", "tag_name":"dataset1", "user_score":1}]}`,
			},
		},
		{
			name: "#02_OK_ALL_TAGS",
			req: wantReq{
				body: `{
"tag_id": null,
"user_score": 1.00,
"limit": 20,
"cursor": 0,
"next":true
}`,
			},
			want: wantResp{
				code: http.StatusOK,
				body: `{"cursor_next":3,"cursor_prev":1, "has_more":false,
"rows":[{"batch_id":1, "hash":"kekeke1", "id":1, "model_score":1, "rel_path":"lol.png","tag_name":"dataset1", "user_score":1},
{"batch_id":1, "hash":"kekeke3", "id":3, "model_score":1, "rel_path":"lol3.png","tag_name":"dataset2", "user_score":1}]}`,
			},
		},
		{
			name: "#03_OK_ALL_MODEL_SCORES",
			req: wantReq{
				body: `{
"tag_id": null,
"user_score": null,
"limit": 20,
"cursor": 0,
"next":true
}`,
			},
			want: wantResp{
				code: http.StatusOK,
				body: `{"cursor_next":4,"cursor_prev":1, "has_more":false, "rows":[
{"batch_id":1, "hash":"kekeke1", "id":1, "model_score":1, "rel_path":"lol.png", "tag_name":"dataset1", "user_score":1},
{"batch_id":1, "hash":"kekeke2", "id":2, "model_score":0.75, "rel_path":"lol2.png","tag_name":"dataset1", "user_score":0.75}, 
{"batch_id":1, "hash":"kekeke3", "id":3, "model_score":1, "rel_path":"lol3.png","tag_name":"dataset2", "user_score":1}, 
{"batch_id":1, "hash":"kekeke4", "id":4, "model_score":0.25, "rel_path":"lol4.png","tag_name":"dataset2",  "user_score":0.25}]}`,
			},
		},
		{
			name: "#05_OK_2_ENTITIES",
			req: wantReq{
				body: `{
"tag_id": null,
"user_score": null,
"limit": 2,
"cursor": 0,
"next":true
}`,
			},
			want: wantResp{
				code: http.StatusOK,
				body: `{"cursor_next":2,"cursor_prev":1, "has_more":true, "rows":[
{"batch_id":1, "hash":"kekeke1", "id":1, "model_score":1, "rel_path":"lol.png","tag_name":"dataset1", "user_score":1}, 
{"batch_id":1, "hash":"kekeke2", "id":2, "model_score":0.75, "rel_path":"lol2.png","tag_name":"dataset1", "user_score":0.75}]}`,
			},
		},
		{
			name: "#06_INVALID_JSON",
			req: wantReq{
				body: `{"d;}`,
			},
			want: wantResp{
				code: http.StatusBadRequest,
				body: `{"error":"invalid json"}`,
			},
		},
		{
			name: "#07_INVALID_REQ",
			req: wantReq{
				body: `{
"tag_id": 0,
"user_score": -1.50,
"limit": -10,
"cursor": -100,
"next":true
}`,
			},
			want: wantResp{
				code: http.StatusBadRequest,
				body: `{"error": "bad request"}`,
			},
		},
		{
			name: "#08_PREV_FROM_START",
			req: wantReq{
				body: `{
"tag_id": null,
"user_score": null,
"limit": 2,
"cursor": 0,
"next":false
}`,
			},
			want: wantResp{
				code: http.StatusBadRequest,
				body: `{"error": "bad request"}`,
			},
		},
		{
			name: "#09_OK_PREV_WITH_CURSOR",
			req: wantReq{
				body: `{
"tag_id": null,
"user_score": null,
"limit": 2,
"cursor": 4,
"next":false
}`,
			},
			want: wantResp{
				code: http.StatusOK,
				body: `{"cursor_next":3, "cursor_prev":2, "has_more":true, "rows":[
{"batch_id":1, "hash":"kekeke2", "id":2, "model_score":0.75, "rel_path":"lol2.png", "tag_name":"dataset1", "user_score":0.75}, 
{"batch_id":1, "hash":"kekeke3", "id":3, "model_score":1, "rel_path":"lol3.png", "tag_name":"dataset2", "user_score":1}]}`,
			},
		},
	}

	sut := middlewares.LogMiddleware(listTrainingRowsByReqHandler(trainingService))
	require.NoError(t, testutil.LoadFixtures(pool, listTrainingRowsFixture, resetALLFixtures))

	method := http.MethodPost
	hndURL := "/training/list/req"
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
