package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"scraper/internal/pkg/middlewares"
	"scraper/internal/pkg/testutil"
	scrapestate "scraper/internal/scrapeState"
	"scraper/internal/service/download"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScrapeImagesHandler(t *testing.T) {
	srv := launchTestServer(t)
	defer srv.Close()
	type wantReq struct {
		body        string
		failProcess bool
	}
	type wantResp struct {
		code           int
		body           string
		remainingPages int
		isProcessed    bool
	}
	tests := []struct {
		name string
		req  wantReq
		want wantResp
	}{
		{
			name: "#01_OK_ABS",
			req: wantReq{
				body: func() string {
					return fmt.Sprintf(`{
"url": "%s/page/1",
"post_selector": ".post-link",
"image_attr": "href",
"next_page_selector": ".next-link",
"next_page_attr": "href",
"pages": 2,
"limit": 20
}`, srv.URL)
				}(),
				failProcess: false,
			},
			want: wantResp{
				code:           http.StatusOK,
				body:           `{"batch_id":1}`,
				remainingPages: 0,
				isProcessed:    true,
			},
		},
		{
			name: "#02_INVALID_JSON",
			req: wantReq{
				body:        `{s}`,
				failProcess: false,
			},
			want: wantResp{
				code: http.StatusBadRequest,
				body: `{"error":"invalid json"}`,
			},
		},
		{
			name: "#03_INVALID_REQ",
			req: wantReq{
				body: `{
  "url": "",
  "post_selector": "",
  "image_attr": "",
  "next_page_selector": "",
  "next_page_attr": "",
  "pages": 0,
  "limit": 0
}`,
			},
			want: wantResp{
				code: http.StatusBadRequest,
				body: `{"error":"bad request"}`,
			},
		},
		{
			name: "#04_OK_REF",
			req: wantReq{
				body: func() string {
					return fmt.Sprintf(`{
"url": "%s/posts?page=1",
"post_selector": ".post-link",
"image_attr": "href",
"next_page_selector": ".next-link",
"next_page_attr": "href",
"pages": 5,
"limit": 20
}`, srv.URL)
				}(),
				failProcess: false,
			},
			want: wantResp{
				code:           http.StatusOK,
				body:           `{"batch_id":1}`,
				remainingPages: 0,
				isProcessed:    true,
			},
		},
		{
			name: "#05_FAIL_PROCESS_FILES",
			req: wantReq{
				body: func() string {
					return fmt.Sprintf(`{
"url": "%s/posts?page=1",
"post_selector": ".post-link",
"image_attr": "href",
"next_page_selector": ".next-link",
"next_page_attr": "href",
"pages": 5,
"limit": 20
}`, srv.URL)
				}(),
				failProcess: true,
			},
			want: wantResp{
				code:           http.StatusInternalServerError,
				body:           `{"error":"service error"}`,
				remainingPages: 0,
				isProcessed:    false,
			},
		},
	}

	localSVCPath := download.EnvPaths{
		ModelName:           `taste_head.pt`,
		DownloadDir:         `testdata\runtimeTest`,
		ModelDir:            `testdata\modelsForTest`,
		ModelNameConfigPath: `testdata\config\model.yaml`,
		ImportDir:           `testdata\import`,
	}

	method := http.MethodPost
	hndURL := "/scrape"
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, testutil.ClearDir(`E:\AI\Embeddings Default City\ScraperSet\internal\server\testdata\runtimeTest`))
			require.NoError(t, testutil.ClearFixtures(pool, resetALLFixtures))
			repo := &wrappedScraperRepo{
				Repository:       scraperRepo,
				CopyCount:        0,
				ProcessFilesFail: tt.req.failProcess,
			}

			downloadReq := download.NewServiceReq{
				ScraperRepo: repo,
				ImageSVC:    imagerowService,
				TrainingSVC: trainingService,
				Paths:       localSVCPath,
			}

			downloadSVC, err := download.NewService(downloadReq)
			require.NoError(t, err)

			sut := middlewares.LogMiddleware(ScrapeImagesHandler(downloadSVC))

			req := httptest.NewRequest(method, hndURL, strings.NewReader(tt.req.body))
			rr := httptest.NewRecorder()

			sut.ServeHTTP(rr, req)
			assert.Equal(t, tt.want.code, rr.Code)
			assert.JSONEq(t, tt.want.body, rr.Body.String())

			if tt.want.code == http.StatusOK || tt.want.code == http.StatusInternalServerError {
				state := readStates(t, svcPaths.DownloadDir)
				assert.Equal(t, tt.want.remainingPages, state.PagesRemaining)
				assert.Equal(t, tt.want.isProcessed, state.IsProcessed)
			}
		})
	}
}

func readStates(t *testing.T, downloadDir string) *scrapestate.StateManifest {
	t.Helper()
	entries, err := os.ReadDir(downloadDir)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.True(t, entries[0].IsDir())

	stateDir := filepath.Join(downloadDir, entries[0].Name())
	state, err := scrapestate.ReadStateManifest(stateDir)
	require.NoError(t, err)

	return state
}

func launchTestServer(t *testing.T) *httptest.Server {
	t.Helper()

	img1, err := os.ReadFile(`testdata\fixtures\scrapeImages\image.jpg`)
	require.NoError(t, err)

	img2, err := os.ReadFile(`testdata\fixtures\scrapeImages\image2.webp`)
	require.NoError(t, err)

	img3, err := os.ReadFile(`testdata\fixtures\scrapeImages\image3.jpg`)
	require.NoError(t, err)
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/page/1":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprintf(w, `
				<html><body>
					<a class="post-link" href="%s/img/image.jpg">img1</a>
					<a class="post-link" href="%s/img/image2.webp">img2</a>
					<a class="next-link" href="%s/page/2">next</a>
				</body></html>`,
				srv.URL, srv.URL, srv.URL,
			)
		case "/page/2":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprintf(w, `
			<html><body>
				<a class="post-link" href="%s/img/image3.jpg">img3</a>
				<a class="next-link" href="%s/page/3">next</a>
			</body></html>
		`, srv.URL, srv.URL)
		case "/page/3":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprint(w, `
			<html><body>
				done
			</body></html>
		`)
		case "/img/image.jpg":
			_, err := w.Write(img1)
			require.NoError(t, err)
		case "/img/image2.webp":
			_, err := w.Write(img2)
			require.NoError(t, err)
		case "/img/image3.jpg":
			_, err := w.Write(img3)
			require.NoError(t, err)
		case "/posts":
			switch r.URL.Query().Get("page") {
			case "1":
				fmt.Fprintf(w, `
				<html><body>
					<a class="post-link" href="%s/img/image.jpg">img1</a>
					<a class="next-link" href="?page=2">next</a>
				</body></html>
			`, srv.URL)
			case "2":
				fmt.Fprintf(w, `
				<html><body>
					<a class="post-link" href="%s/img/image2.webp">img1</a>
					<a class="next-link" href="?page=3">next</a>
				</body></html>
			`, srv.URL)
			case "3":
				fmt.Fprintf(w, `
				<html><body>
					<a class="post-link" href="%s/img/image3.jpg">img1</a>
				</body></html>
			`, srv.URL)
			default:
				http.NotFound(w, r)
			}
		default:
			http.NotFound(w, r)
		}
	}))

	return srv
}
