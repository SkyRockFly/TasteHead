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

const fixtureListDeletedImages = `testdata\fixtures\listDeletedImages\rows.sql`

func TestListDeletedImagesHandler(t *testing.T) {
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
			name: "01_OK_NEXT",
			req: wantReq{
				body: `{"next":true,"limit":5}`,
			},
			want: wantResp{
				code: http.StatusOK,
				body: `{
  "cursor_next": {"cursor_deleted_at":"2026-04-12T11:00:00Z", "cursor_id":5},
  "cursor_prev": {"cursor_deleted_at":"2026-04-12T15:00:00Z", "cursor_id":1},
  "has_more": true,
  "images": [
    {
      "id": 1,
      "hash": "kekeke1",
      "batch_id": 1,
      "rel_path": "lol.png",
      "model_score": 1,
      "user_score": 1,
      "image_deleted_at": "2026-04-12T15:00:00Z",
      "batch_deleted_at": null
    },
    {
      "id": 2,
      "hash": "kekeke2",
      "batch_id": 1,
      "rel_path": "lol2.png",
      "model_score": 0.75,
      "user_score": 0.75,
      "image_deleted_at": "2026-04-12T14:00:00Z",
      "batch_deleted_at": null
    },
    {
      "id": 3,
      "hash": "kekeke3",
      "batch_id": 1,
      "rel_path": "lol3.png",
      "model_score": 0.5,
      "user_score": 0.5,
      "image_deleted_at": "2026-04-12T13:00:00Z",
      "batch_deleted_at": null
    },
    {
      "id": 4,
      "hash": "kekeke4",
      "batch_id": 1,
      "rel_path": "lol4.png",
      "model_score": 0.25,
      "user_score": 0.25,
      "image_deleted_at": "2026-04-12T12:00:00Z",
      "batch_deleted_at": null
    },
    {
      "id": 5,
      "hash": "kekeke5",
      "batch_id": 1,
      "rel_path": "lol5.png",
      "model_score": 0,
      "user_score": 0,
      "image_deleted_at": "2026-04-12T11:00:00Z",
      "batch_deleted_at": null
    }
  ]
}`,
			},
		},
		{
			name: "#02_OK_WITH_CURSOR_NEXT",
			req: wantReq{
				body: `{"next":true,"limit":5,"cursor_id":5,"cursor_deleted_at":"2026-04-12T11:00:00Z"}`,
			},
			want: wantResp{
				code: http.StatusOK,
				body: `{
  "cursor_next": {"cursor_deleted_at":"2026-04-12T10:00:00Z", "cursor_id":7},
  "cursor_prev": {"cursor_deleted_at":"2026-04-12T10:00:00Z", "cursor_id":11},
  "has_more": false,
  "images": [
    {
      "id": 11,
      "hash": "kekeke10",
      "batch_id": 2,
      "rel_path": "lol10.png",
      "model_score": 0,
      "user_score": 0,
      "image_deleted_at": null,
      "batch_deleted_at": "2026-04-12T10:00:00Z"
    },
    {
      "id": 10,
      "hash": "kekeke9",
      "batch_id": 2,
      "rel_path": "lol9.png",
      "model_score": 0.25,
      "user_score": 0.25,
      "image_deleted_at": null,
      "batch_deleted_at": "2026-04-12T10:00:00Z"
    },
    {
      "id": 9,
      "hash": "kekeke8",
      "batch_id": 2,
      "rel_path": "lol8.png",
      "model_score": 0.5,
      "user_score": 0.5,
      "image_deleted_at": null,
      "batch_deleted_at": "2026-04-12T10:00:00Z"
    },
    {
      "id": 8,
      "hash": "kekeke7",
      "batch_id": 2,
      "rel_path": "lol7.png",
      "model_score": 0.75,
      "user_score": 0.75,
      "image_deleted_at": null,
      "batch_deleted_at": "2026-04-12T10:00:00Z"
    },
    {
      "id": 7,
      "hash": "kekeke6",
      "batch_id": 2,
      "rel_path": "lol6.png",
      "model_score": 1,
      "user_score": 1,
      "image_deleted_at": null,
      "batch_deleted_at": "2026-04-12T10:00:00Z"
    }
  ]
}`,
			},
		},
		{
			name: "#03_OK_WITH_CURSOR_PREV",
			req: wantReq{
				body: `{"next":false,"limit":5,"cursor_id":5,"cursor_deleted_at":"2026-04-12T11:00:00Z"}`,
			},
			want: wantResp{
				code: http.StatusOK,
				body: `{
  "cursor_next": {"cursor_deleted_at":"2026-04-12T12:00:00Z", "cursor_id":4},
  "cursor_prev": {"cursor_deleted_at":"2026-04-12T15:00:00Z", "cursor_id":1},
  "has_more": false,
  "images": [
   {
    "id": 1,
    "hash": "kekeke1",
    "batch_id": 1,
    "rel_path": "lol.png",
    "model_score": 1,
    "user_score": 1,
    "image_deleted_at": "2026-04-12T15:00:00Z",
    "batch_deleted_at": null
  },
  {
    "id": 2,
    "hash": "kekeke2",
    "batch_id": 1,
    "rel_path": "lol2.png",
    "model_score": 0.75,
    "user_score": 0.75,
    "image_deleted_at": "2026-04-12T14:00:00Z",
    "batch_deleted_at": null
  },
  {
    "id": 3,
    "hash": "kekeke3",
    "batch_id": 1,
    "rel_path": "lol3.png",
    "model_score": 0.5,
    "user_score": 0.5,
    "image_deleted_at": "2026-04-12T13:00:00Z",
    "batch_deleted_at": null
  },
  {
    "id": 4,
    "hash": "kekeke4",
    "batch_id": 1,
    "rel_path": "lol4.png",
    "model_score": 0.25,
    "user_score": 0.25,
    "image_deleted_at": "2026-04-12T12:00:00Z",
    "batch_deleted_at": null
  }
  ]
}`,
			},
		},
		{
			name: "#04_PREV_FROM_START",
			req: wantReq{
				body: `{"next":false,"limit":5}`,
			},
			want: wantResp{
				code: http.StatusBadRequest,
				body: `{"error":"bad request"}`,
			},
		},
		{
			name: "#05_INVALID_JSON",
			req: wantReq{
				body: `{"d;}`,
			},
			want: wantResp{
				code: http.StatusBadRequest,
				body: `{"error":"invalid json"}`,
			},
		},
		{
			name: "#06_INVALID_REQ",
			req: wantReq{
				body: `{"next":true,"limit":5,"cursor_id":0,"cursor_deleted_at":"2026-04-12T11:00:00Z"}`,
			},
			want: wantResp{
				code: http.StatusBadRequest,
				body: `{"error": "bad request"}`,
			},
		},
	}

	sut := middlewares.LogMiddleware(ListDeletedImagesHandler(imagerowService))

	method := http.MethodPost
	hndURL := "/images/list/deleted"
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, testutil.LoadFixtures(pool, fixtureListDeletedImages, resetALLFixtures))
			req := httptest.NewRequest(method, hndURL, strings.NewReader(tt.req.body))

			rr := httptest.NewRecorder()

			sut.ServeHTTP(rr, req)
			assert.Equal(t, tt.want.code, rr.Code)
			assert.JSONEq(t, tt.want.body, rr.Body.String())
		})
	}
}
