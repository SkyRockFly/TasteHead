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
	fixtureGetImagesPath = "testdata/fixtures/listImages/rows.sql"
)

func TestListImagesHandler(t *testing.T) {
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
			name: "01_OK",
			req: wantReq{
				body: `{
"dir_path": 1,
"score": 1.00,
"score_type": "model",
"limit": 20,
"cursor": 0,
"next":true
}`,
			},
			want: wantResp{
				code: http.StatusOK,
				body: `{"cursor_next":1,"cursor_prev":1, "has_more":false, "images": 
[{"batch_id":1, "hash":"kekeke1", "id":1, "model_score":1, "rel_path":"lol.png", "tags":[], "user_score":1}]}`,
			},
		},
		{
			name: "#02_OK_ALL_BATCHES",
			req: wantReq{
				body: `{
"dir_path": null,
"score": 1.00,
"score_type": "model",
"limit": 20,
"cursor": 0,
"next":true
}`,
			},
			want: wantResp{
				code: http.StatusOK,
				body: `{"cursor_next":6,"cursor_prev":1, "has_more":false,
"images":[{"batch_id":1, "hash":"kekeke1", "id":1, "model_score":1, "rel_path":"lol.png","tags":[], "user_score":1},
{"batch_id":2, "hash":"kekeke6", "id":6, "model_score":1, "rel_path":"lol6.png","tags":[], "user_score":1}]}`,
			},
		},
		{
			name: "#03_OK_ALL_USER_SCORES_NOT_FOUND",
			req: wantReq{
				body: `{
						"dir_path": null,
						"score": null,
						"score_type": "user",
						"limit": 20,
						"cursor": 0,
						"next":true
						}`,
			},
			want: wantResp{
				code: http.StatusNotFound,
				body: `{"error":"not found"}`,
			},
		},
		{
			name: "#04_OK_ALL_MODEL_SCORES",
			req: wantReq{
				body: `{
"dir_path": null,
"score": null,
"score_type": "model",
"limit": 20,
"cursor": 0,
"next":true
}`,
			},
			want: wantResp{
				code: http.StatusOK,
				body: `{"cursor_next":10,"cursor_prev":1, "has_more":false, "images":[
{"batch_id":1, "hash":"kekeke1", "id":1, "model_score":1, "rel_path":"lol.png","tags":[], "user_score":1},
{"batch_id":1, "hash":"kekeke2", "id":2, "model_score":0.75, "rel_path":"lol2.png","tags":[], "user_score":0.75}, 
{"batch_id":1, "hash":"kekeke3", "id":3, "model_score":0.5, "rel_path":"lol3.png","tags":[], "user_score":0.5}, 
{"batch_id":1, "hash":"kekeke4", "id":4, "model_score":0.25, "rel_path":"lol4.png","tags":[], "user_score":0.25}, 
{"batch_id":1, "hash":"kekeke5", "id":5, "model_score":0, "rel_path":"lol5.png","tags":[], "user_score":0}, 
{"batch_id":2, "hash":"kekeke6", "id":6, "model_score":1, "rel_path":"lol6.png","tags":[], "user_score":1}, 
{"batch_id":2, "hash":"kekeke7", "id":7, "model_score":0.75, "rel_path":"lol7.png","tags":[], "user_score":0.75}, 
{"batch_id":2, "hash":"kekeke8", "id":8, "model_score":0.5, "rel_path":"lol8.png","tags":[], "user_score":0.5}, 
{"batch_id":2, "hash":"kekeke9", "id":9, "model_score":0.25, "rel_path":"lol9.png","tags":[], "user_score":0.25}, 
{"batch_id":2, "hash":"kekeke10", "id":10, "model_score":0, "rel_path":"lol10.png","tags":[], "user_score":0}]}`,
			},
		},
		{
			name: "#05_OK_5_ENTITIES",
			req: wantReq{
				body: `{
"dir_path": null,
"score": null,
"score_type": "model",
"limit": 5,
"cursor": 0,
"next":true
}`,
			},
			want: wantResp{
				code: http.StatusOK,
				body: `{"cursor_next":5, "cursor_prev":1, "has_more":true, "images":[
{"batch_id":1, "hash":"kekeke1", "id":1, "model_score":1, "rel_path":"lol.png","tags":[], "user_score":1}, 
{"batch_id":1, "hash":"kekeke2", "id":2, "model_score":0.75, "rel_path":"lol2.png","tags":[], "user_score":0.75}, 
{"batch_id":1, "hash":"kekeke3", "id":3, "model_score":0.5, "rel_path":"lol3.png","tags":[], "user_score":0.5}, 
{"batch_id":1, "hash":"kekeke4", "id":4, "model_score":0.25, "rel_path":"lol4.png","tags":[], "user_score":0.25}, 
{"batch_id":1, "hash":"kekeke5", "id":5, "model_score":0, "rel_path":"lol5.png","tags":[], "user_score":0}]}`,
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
			name: "#07_PREV_FROM_START",
			req: wantReq{
				body: `{
"dir_path": 0,
"score": 1.00,
"score_type": "mode",
"limit": 10,
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
			name: "#08_OK_PREV_WITH_CURSOR",
			req: wantReq{
				body: `{
"dir_path": null,
"score": null,
"score_type": "model",
"limit": 5,
"cursor": 7,
"next":false
}`,
			},
			want: wantResp{
				code: http.StatusOK,
				body: `{"cursor_next":6, "cursor_prev":2, "has_more":true, "images":[
{"batch_id":1, "hash":"kekeke2", "id":2, "model_score":0.75, "rel_path":"lol2.png","tags":[], "user_score":0.75}, 
{"batch_id":1, "hash":"kekeke3", "id":3, "model_score":0.5, "rel_path":"lol3.png","tags":[], "user_score":0.5}, 
{"batch_id":1, "hash":"kekeke4", "id":4, "model_score":0.25, "rel_path":"lol4.png","tags":[], "user_score":0.25}, 
{"batch_id":1, "hash":"kekeke5", "id":5, "model_score":0, "rel_path":"lol5.png","tags":[], "user_score":0}, 
{"batch_id":2, "hash":"kekeke6", "id":6, "model_score":1, "rel_path":"lol6.png","tags":[], "user_score":1}]}`,
			},
		},
	}

	sut := middlewares.LogMiddleware(ListImagesHandler(downloadSVC))
	require.NoError(t, testutil.LoadFixtures(pool, fixtureGetImagesPath, resetALLFixtures))

	method := http.MethodPost
	hndURL := "/images/list"
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
