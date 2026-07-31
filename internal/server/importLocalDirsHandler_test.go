package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"scraper/internal/pkg/middlewares"
	"scraper/internal/pkg/testutil"
	"scraper/internal/repository/scraper"
	"scraper/internal/service/download"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	fixtureImportLocalDirs = `testdata/fixtures/importLocalDirs/rows.sql`
)

var ValidExts = []string{".jpg", ".png", ".jpeg", ".webp"}

type wrappedScraperRepo struct {
	scraper.Repository
	CopyCount        int
	CopyFailAt       int
	ProcessFilesFail bool
}

func (r *wrappedScraperRepo) CopyFile(src, dest string) error {
	r.CopyCount++
	if r.CopyCount == r.CopyFailAt && r.CopyFailAt > 0 {
		return fmt.Errorf("copy error")
	}

	return r.Repository.CopyFile(src, dest)
}

func (r *wrappedScraperRepo) ProcessFiles(ctx context.Context, req scraper.ProcessFilesReq) error {
	if r.ProcessFilesFail {
		return fmt.Errorf("process files error")
	}
	return r.Repository.ProcessFiles(ctx, req)
}

func TestImportLocalDirsHandler(t *testing.T) {
	type wantReq struct {
		path        string
		failCopyAt  int
		prepareDir  func(t *testing.T)
		failProcess bool
	}
	type wantResp struct {
		code                 int
		body                 string
		expectedFilesInBatch int64
		expectedFilesInDir   int64
		batchID              int64
		IsDirExist           bool
	}

	tests := []struct {
		name string
		req  wantReq
		want wantResp
	}{
		{
			name: "#01_OK",
			req: wantReq{
				path:        `testdata/fixtures/importLocalDirs/okDir`,
				failCopyAt:  0,
				failProcess: false,
			},
			want: wantResp{
				code: http.StatusOK,
				body: mustJSON(map[string]any{
					"fails": nil,
				}),
				expectedFilesInBatch: 2,
				expectedFilesInDir:   2,
				batchID:              3,
				IsDirExist:           true,
			},
		},
		{
			name: "#02_OK_NO_UNIQUE",
			req: wantReq{
				path:        `testdata/fixtures/importLocalDirs/okDirNoUnique`,
				failCopyAt:  0,
				failProcess: false,
			},
			want: wantResp{
				code: http.StatusOK,
				body: mustJSON(map[string]any{
					"fails": []any{
						map[string]any{
							"Path": slashPath("testdata", "import", "okDirNoUnique"),
							"Warn": "No unique files in dir",
						},
					},
				},
				),
			},
		},
		{
			name: "#03_EMPTY_DIR",
			req: wantReq{
				failCopyAt: 0,
			},
			want: wantResp{
				code: http.StatusNotFound,
				body: `{"error":"not found"}`,
			},
		},
		{
			name: "#04_NOT_A_DIR",
			req: wantReq{
				path:       "",
				failCopyAt: 0,
				prepareDir: func(t *testing.T) {
					t.Helper()
					notADir := filepath.Join(svcPaths.ImportDir, "notADir")
					file, err := os.Create(notADir)
					require.NoError(t, err)
					file.Close()
				},
				failProcess: false,
			},
			want: wantResp{
				code: http.StatusNotFound,
				body: `{"error":"not found"}`,
			},
		},
		{
			name: "#05_ENTRY_EXIST_WITHOUT_DIR",
			req: wantReq{
				path:        `testdata/fixtures/importLocalDirs/entryExistWithoutDir`,
				failCopyAt:  0,
				failProcess: false,
			},
			want: wantResp{
				code: http.StatusOK,
				body: mustJSON(map[string]any{
					"fails": []any{
						map[string]any{
							"Path": slashPath("testdata", "import", "entryExistWithoutDir"),
							"Warn": "Entry exists while dir is not",
						},
					},
				},
				),
			},
		},
		{
			name: "#06_DIR_EXIST_WITHOUT_ENTRY",
			req: wantReq{
				path:       `testdata/fixtures/importLocalDirs/dirExistWithoutEntry`,
				failCopyAt: 0,
				prepareDir: func(t *testing.T) {
					t.Helper()
					require.NoError(t, testutil.CreateTempDirFromPicFixtureDir(
						`testdata/fixtures/importLocalDirs/dirExistWithoutEntry`,
						filepath.Join(svcPaths.DownloadDir, "dirExistWithoutEntry"),
					))
				},
				failProcess: false,
			},
			want: wantResp{
				code: http.StatusOK,
				body: mustJSON(map[string]any{
					"fails": []any{
						map[string]any{
							"Path": slashPath("testdata", "import", "dirExistWithoutEntry"),
							"Warn": "Dir exists while entry is not",
						},
					},
				},
				),
				expectedFilesInBatch: 0,
				expectedFilesInDir:   4,
				batchID:              0,
				IsDirExist:           true,
			},
		},
		{
			name: "#07_UPDATE_DIR",
			req: wantReq{
				path:       `testdata/fixtures/importLocalDirs/update/updateDir`,
				failCopyAt: 0,
				prepareDir: func(t *testing.T) {
					t.Helper()
					require.NoError(t, testutil.CreateTempDirFromPicFixtureDir(
						`testdata/fixtures/importLocalDirs/update/currentDir`,
						filepath.Join(svcPaths.DownloadDir, "updateDir"),
					))
				},
				failProcess: false,
			},
			want: wantResp{
				code: http.StatusOK,
				body: mustJSON(map[string]any{
					"fails": nil,
				}),
				expectedFilesInBatch: 4,
				expectedFilesInDir:   4,
				batchID:              1,
				IsDirExist:           true,
			},
		},
		{
			name: "#08_COPY_ERROR",
			req: wantReq{
				path:        `testdata/fixtures/importLocalDirs/copyError`,
				failCopyAt:  2,
				failProcess: false,
			},
			want: wantResp{
				code: http.StatusOK,
				body: mustJSON(map[string]any{
					"fails": []any{
						map[string]any{
							"Path": slashPath("testdata", "import", "copyError"),
							"Warn": "Copy file error",
						},
					},
				},
				),

				expectedFilesInBatch: 0,
				expectedFilesInDir:   0,
				batchID:              0,
				IsDirExist:           false,
			},
		},
		{
			name: "#09_PROCESS_ERROR",
			req: wantReq{
				path:        `testdata/fixtures/importLocalDirs/copyError`,
				failCopyAt:  0,
				failProcess: true,
			},
			want: wantResp{
				code: http.StatusOK,
				body: mustJSON(map[string]any{
					"fails": []any{
						map[string]any{
							"Path": slashPath("testdata", "import", "copyError"),
							"Warn": "Process files error",
						},
					},
				},
				),
				expectedFilesInBatch: 0,
				expectedFilesInDir:   0,
				batchID:              0,
				IsDirExist:           false,
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

	method := http.MethodGet
	hndURL := "/import/batches"
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, testutil.ClearDir(svcPaths.DownloadDir))
			require.NoError(t, testutil.ClearDir(svcPaths.ImportDir))
			require.NoError(t, testutil.LoadFixtures(pool, fixtureImportLocalDirs, resetALLFixtures))
			if tt.req.path != "" {
				prepareDirFromFixture(t, tt.req.path)
			}

			if tt.req.prepareDir != nil {
				tt.req.prepareDir(t)
			}

			repo := &wrappedScraperRepo{
				Repository:       scraperRepo,
				CopyCount:        0,
				CopyFailAt:       tt.req.failCopyAt,
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

			sut := middlewares.LogMiddleware(ImportLocalDirsHandler(downloadSVC))
			req := httptest.NewRequest(method, hndURL, nil)
			rr := httptest.NewRecorder()

			sut.ServeHTTP(rr, req)
			assert.Equal(t, tt.want.code, rr.Code)
			assert.JSONEq(t, tt.want.body, rr.Body.String())
			assert.Equal(t, tt.want.expectedFilesInBatch, getFilesCountFromDB(t, tt.want.batchID))

			destPath := filepath.Join(svcPaths.DownloadDir, filepath.Base(tt.req.path))

			if tt.req.path != "" {
				assert.Equal(t, tt.want.IsDirExist, isDirExist(destPath))
			}

			if tt.want.IsDirExist {
				assert.Equal(t, tt.want.expectedFilesInDir, getFilesCountByDir(t, destPath))
				assert.Equal(t, false, checkRetryDir(t, destPath))
			}
			if tt.want.expectedFilesInBatch != 0 {
				loadPictures(t, int(tt.want.batchID), svcPaths.DownloadDir)
			}
		})
	}
}

func getFilesCountFromDB(t *testing.T, batchID int64) int64 {
	query := `SELECT COUNT(*) FROM download WHERE batch_id = $1`
	var count int64
	err := pool.QueryRow(context.Background(), query, batchID).Scan(&count)
	require.NoError(t, err)

	return count
}

func getFilesCountByDir(t *testing.T, dir string) int64 {
	fileCount := 0
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if slices.Contains(ValidExts, filepath.Ext(entry.Name())) {
			fileCount++
		}
	}

	return int64(fileCount)
}

func isDirExist(path string) bool {
	file, err := os.Stat(path)
	if err != nil {
		return false
	}

	if !file.IsDir() {
		return false
	}

	return true
}

func prepareDirFromFixture(t *testing.T, path string) {
	t.Helper()
	dirName := filepath.Base(path)
	err := testutil.CreateTempDirFromPicFixtureDir(path, filepath.Join(svcPaths.ImportDir, dirName))
	require.NoError(t, err)
}

func mustJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(b)
}

func slashPath(parts ...string) string {
	return filepath.ToSlash(filepath.Join(parts...))
}
