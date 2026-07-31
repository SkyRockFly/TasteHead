package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
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

func TestResumeScrapeHandler(t *testing.T) {
	srv := launchTestServer(t)
	defer srv.Close()

	type wantReq struct {
		body       string
		prepareDir func(t *testing.T)
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
			name: "#01_WITH_ONE_IMAGE",
			req: wantReq{
				body: `{"batch_name":"withOneImage"}`,
				prepareDir: func(t *testing.T) {
					t.Helper()
					fixture := `testdata/fixtures/resumeScrape/withOneImage`
					dest := filepath.Join(svcPaths.DownloadDir, "withOneImage")
					require.NoError(t, testutil.CreateTempDirFromPicFixtureDir(fixture, dest))

					state, err := scrapestate.NewStateManifest(dest)
					require.NoError(t, err)
					require.NoError(t, state.SetPage(3, fmt.Sprintf("%s/page/1", srv.URL)))

					scrapeReq := scrapestate.ScrapeManifest{
						PostSelector:     ".post-link",
						ImageAttr:        "href",
						NextPageSelector: ".next-link",
						NextPageAttr:     "href",
					}
					require.NoError(t, scrapestate.CreateScrapeManifest(dest, scrapeReq))
				},
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
				body: `{s}`,
			},
			want: wantResp{
				code: http.StatusBadRequest,
				body: `{"error":"invalid json"}`,
			},
		},
		{
			name: "#03_INVALID_REQ",
			req: wantReq{
				body: `{"batch_name":""}`,
			},
			want: wantResp{
				code: http.StatusBadRequest,
				body: `{"error":"bad request"}`,
			},
		},
		{
			name: "#04_UNPROCESSED",
			req: wantReq{
				body: `{"batch_name":"unprocessed"}`,
				prepareDir: func(t *testing.T) {
					t.Helper()
					fixture := `testdata/fixtures/resumeScrape/unprocessed`
					dest := filepath.Join(svcPaths.DownloadDir, "unprocessed")
					require.NoError(t, testutil.CreateTempDirFromPicFixtureDir(fixture, dest))

					state, err := scrapestate.NewStateManifest(dest)
					require.NoError(t, err)
					require.NoError(t, state.SetPage(0, fmt.Sprintf("%s/page/3", srv.URL)))

					scrapeReq := scrapestate.ScrapeManifest{
						PostSelector:     ".post-link",
						ImageAttr:        "href",
						NextPageSelector: ".next-link",
						NextPageAttr:     "href",
					}
					require.NoError(t, scrapestate.CreateScrapeManifest(dest, scrapeReq))
				},
			},
			want: wantResp{
				code:           http.StatusOK,
				body:           `{"batch_id":1}`,
				remainingPages: 0,
				isProcessed:    true,
			},
		},
		{
			name: "05_CANCELLED_DURING_PAGE_SCRAPE",
			req: wantReq{
				body: `{"batch_name":"cancelledDuringPageScrape"}`,
				prepareDir: func(t *testing.T) {
					t.Helper()
					fixture := `testdata/fixtures/resumeScrape/cancelledDuringPageScrape`
					dest := filepath.Join(svcPaths.DownloadDir, "cancelledDuringPageScrape")
					require.NoError(t, testutil.CreateTempDirFromPicFixtureDir(fixture, dest))

					state, err := scrapestate.NewStateManifest(dest)
					require.NoError(t, err)
					require.NoError(t, state.SetPage(3, fmt.Sprintf("%s/page/1", srv.URL)))

					scrapeReq := scrapestate.ScrapeManifest{
						PostSelector:     ".post-link",
						ImageAttr:        "href",
						NextPageSelector: ".next-link",
						NextPageAttr:     "href",
					}
					require.NoError(t, scrapestate.CreateScrapeManifest(dest, scrapeReq))
				},
			},
			want: wantResp{
				code:           http.StatusOK,
				body:           `{"batch_id":1}`,
				remainingPages: 0,
				isProcessed:    true,
			},
		},
	}

	localSVCPath := download.EnvPaths{
		ModelName:           `taste_head.pt`,
		DownloadDir:         `testdata/runtimeTest`,
		ModelDir:            `testdata/modelsForTest`,
		ModelNameConfigPath: `testdata/config/model.yaml`,
		ImportDir:           `testdata/import`,
	}

	downloadReq := download.NewServiceReq{
		ScraperRepo: scraperRepo,
		ImageSVC:    imagerowService,
		TrainingSVC: trainingService,
		Paths:       localSVCPath,
	}

	localDownloadSVC, err := download.NewService(downloadReq)
	require.NoError(t, err)

	sut := middlewares.LogMiddleware(ResumeScrapeHandler(localDownloadSVC))

	method := http.MethodPost
	hndURL := "/scrape/resume"
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, testutil.ClearDir(svcPaths.DownloadDir))
			require.NoError(t, testutil.ClearFixtures(pool, resetALLFixtures))

			if tt.req.prepareDir != nil {
				tt.req.prepareDir(t)
			}

			req := httptest.NewRequest(method, hndURL, strings.NewReader(tt.req.body))
			rr := httptest.NewRecorder()

			sut.ServeHTTP(rr, req)
			assert.Equal(t, tt.want.code, rr.Code)
			assert.JSONEq(t, tt.want.body, rr.Body.String())

			if tt.want.code == http.StatusOK {
				state := readStates(t, svcPaths.DownloadDir)
				assert.Equal(t, tt.want.remainingPages, state.PagesRemaining)
				assert.Equal(t, tt.want.isProcessed, state.IsProcessed)
				loadPictures(t, 1, svcPaths.DownloadDir)
			}
		})
	}
}
