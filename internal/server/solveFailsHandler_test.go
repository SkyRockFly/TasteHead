package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"scraper/internal/pkg/apperror"
	"scraper/internal/pkg/middlewares"
	"scraper/internal/pkg/testutil"
	scrapestate "scraper/internal/scrapeState"
	"scraper/internal/service/download"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testSolveFailsFixture = `testdata/fixtures/solveFails/rows.sql`
)

func TestSolveFailsHandler(t *testing.T) {
	srv := launchTestServer(t)
	defer srv.Close()

	type wantReq struct {
		src        string
		body       string
		prepareDir func(t *testing.T)
	}
	type wantResp struct {
		code           int
		body           string
		filesInDir     int64
		batchesCount   int64
		jsonFailsExist bool
	}
	tests := []struct {
		name string
		req  wantReq
		want wantResp
	}{
		{
			name: "01_2_FAILS",
			req: wantReq{
				src:  "dir",
				body: `{"batch_name":"dir"}`,
				prepareDir: func(t *testing.T) {
					t.Helper()

					report := scrapestate.JSONFailReport{
						OriginURL: srv.URL + "/page/1",
						Fails: []scrapestate.JSONFailedDownload{
							{
								Issue:      apperror.IssueGetRequestError,
								URL:        srv.URL + "/img/image.jpg",
								Repeatable: true,
							},
							{
								Issue:      apperror.IssueGetRequestError,
								URL:        srv.URL + "/img/image2.webp",
								Repeatable: true,
							},
						},
					}

					scrapeInfo := scrapestate.ScrapeManifest{
						PostSelector:     ".post-link",
						ImageAttr:        "href",
						NextPageSelector: ".next-link",
						NextPageAttr:     "href",
					}

					destpath := filepath.Join(svcPaths.DownloadDir, "dir")
					err := testutil.CreateTempDirFromPicFixtureDir(`testdata/fixtures/solveFails/dir`, destpath)
					require.NoError(t, err)

					failFile, err := os.Create(filepath.Join(destpath,
						"fails.jsonl"))
					require.NoError(t, err)
					defer failFile.Close()

					scrapeFile, err := os.Create(filepath.Join(destpath,
						"scrapeManifest.json"))
					require.NoError(t, err)
					defer scrapeFile.Close()

					require.NoError(t, json.NewEncoder(scrapeFile).Encode(scrapeInfo))
					require.NoError(t, json.NewEncoder(failFile).Encode(report))
				},
			},
			want: wantResp{
				code:           http.StatusOK,
				body:           `{"fails":[]}`,
				filesInDir:     3,
				batchesCount:   1,
				jsonFailsExist: false,
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
			name: "04_WITH_UNPROCESSABLE",
			req: wantReq{
				src:  "dir",
				body: `{"batch_name":"dir"}`,
				prepareDir: func(t *testing.T) {
					t.Helper()

					report := scrapestate.JSONFailReport{
						OriginURL: srv.URL + "/page/1",
						Fails: []scrapestate.JSONFailedDownload{
							{
								Issue:      apperror.IssueBadFilename,
								URL:        srv.URL + "/img/image.jpg",
								Repeatable: false,
							},
							{
								Issue:      apperror.IssueGetRequestError,
								URL:        srv.URL + "/img/image2.webp",
								Repeatable: true,
							},
						},
					}

					scrapeInfo := scrapestate.ScrapeManifest{
						PostSelector:     ".post-link",
						ImageAttr:        "href",
						NextPageSelector: ".next-link",
						NextPageAttr:     "href",
					}

					destpath := filepath.Join(svcPaths.DownloadDir, "dir")
					err := testutil.CreateTempDirFromPicFixtureDir(`testdata/fixtures/solveFails/dir`, destpath)
					require.NoError(t, err)

					failFile, err := os.Create(filepath.Join(destpath,
						"fails.jsonl"))
					require.NoError(t, err)
					defer failFile.Close()

					scrapeFile, err := os.Create(filepath.Join(destpath,
						"scrapeManifest.json"))
					require.NoError(t, err)
					defer scrapeFile.Close()

					require.NoError(t, json.NewEncoder(scrapeFile).Encode(scrapeInfo))
					require.NoError(t, json.NewEncoder(failFile).Encode(report))
				},
			},
			want: wantResp{
				code:           http.StatusOK,
				body:           `{"fails":[]}`,
				filesInDir:     2,
				batchesCount:   1,
				jsonFailsExist: false,
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

	sut := middlewares.LogMiddleware(SolveFailsHandler(localDownloadSVC))

	method := http.MethodPost
	hndURL := "/scrape/fails"
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, testutil.ClearDir(svcPaths.DownloadDir))
			require.NoError(t, testutil.LoadFixtures(pool, testSolveFailsFixture, resetALLFixtures))
			if tt.req.prepareDir != nil {
				tt.req.prepareDir(t)
			}

			req := httptest.NewRequest(method, hndURL, strings.NewReader(tt.req.body))
			rr := httptest.NewRecorder()

			sut.ServeHTTP(rr, req)
			assert.Equal(t, tt.want.code, rr.Code)
			assert.JSONEq(t, tt.want.body, rr.Body.String())
			if tt.want.code == http.StatusOK {
				batchPath := filepath.Join(svcPaths.DownloadDir, tt.req.src)
				assert.Equal(t, tt.want.filesInDir, getFilesCountFromDB(t, 1))
				assert.Equal(t, tt.want.filesInDir, getFilesCountByDir(t, batchPath))

				assert.Equal(t, tt.want.batchesCount, countDirs(t, svcPaths.DownloadDir))
				assert.Equal(t, tt.want.batchesCount, countBatchesDB(t))
				assert.Equal(t, false, checkRetryDir(t, batchPath))
				loadPictures(t, 1, svcPaths.DownloadDir)
			}
		})
	}
}

func countDirs(t *testing.T, src string) int64 {
	t.Helper()

	entries, err := os.ReadDir(src)
	require.NoError(t, err)
	var counter int64
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		counter++
	}

	return counter
}

func countBatchesDB(t *testing.T) int64 {
	t.Helper()
	sql := `SELECT COUNT(*) FROM batch`

	var count int64
	require.NoError(t, pool.QueryRow(context.Background(), sql).Scan(&count))

	return count
}

func countImagesDB(t *testing.T) int64 {
	t.Helper()
	sql := `SELECT COUNT(*) FROM download`

	var count int64
	require.NoError(t, pool.QueryRow(context.Background(), sql).Scan(&count))

	return count
}

func checkRetryDir(t *testing.T, src string) bool {
	t.Helper()

	entries, err := os.ReadDir(src)
	require.NoError(t, err)

	isRetryInDir := false
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		if strings.HasPrefix(entry.Name(), "_retry") {
			isRetryInDir = true
			break
		}
	}

	return isRetryInDir
}

func loadPictures(t *testing.T, batchID int, downloadPath string) {
	t.Helper()

	sql := `SELECT rel_path FROM download WHERE batch_id = $1`

	rows, err := pool.Query(context.Background(), sql, batchID)
	require.NoError(t, err)
	defer rows.Close()

	imgPaths := make([]string, 0)

	for rows.Next() {
		var imgPath string
		require.NoError(t, rows.Scan(&imgPath))
		imgPaths = append(imgPaths, imgPath)
	}

	require.NoError(t, rows.Err())

	sql = `SELECT rel_path FROM batch WHERE id = $1`

	var batchPath string
	require.NoError(t, pool.QueryRow(context.Background(), sql, batchID).Scan(&batchPath))

	for _, imgPath := range imgPaths {
		fullPath := filepath.Join(downloadPath, batchPath, imgPath)

		_, err := os.ReadFile(fullPath)
		require.NoError(t, err)

		thumbsPath := filepath.Join(downloadPath, batchPath, "thumbs", imgPath)
		_, err = os.ReadFile(thumbsPath)
		require.NoError(t, err)
	}
}
