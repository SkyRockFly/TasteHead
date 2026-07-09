package server

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"scraper/internal/pkg/testutil"
	imgrowpg "scraper/internal/repository/imageRow/pg"
	htmlparser "scraper/internal/repository/scraper/html"
	trainingpg "scraper/internal/repository/training/pg"
	"scraper/internal/service/download"
	imagerowsvc "scraper/internal/service/imageRow"
	trainingsvc "scraper/internal/service/training"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	resetALLFixtures = `TRUNCATE TABLE batch,download,tag,training,image_hash RESTART IDENTITY CASCADE;`
)

var svcPaths = download.EnvPaths{
	ModelName:           `taste_head.pt`,
	DownloadDir:         filepath.Join("testdata", "runtimeTest"),
	ModelDir:            filepath.Join("testdata", "models"),
	ModelNameConfigPath: filepath.Join("testdata", "config", "model.yaml"),
	ImportDir:           filepath.Join("testdata", "import"),
}

type toAbsStruct struct {
	ModelDir    string
	ImportDir   string
	DownloadDir string
	VenvDir     string
	ScriptsDir  string
	TrainingDir string
}

var (
	pool            *pgxpool.Pool
	scraperRepo     *htmlparser.Repository
	imagerowRepo    *imgrowpg.Repository
	trainingRepo    *trainingpg.Repository
	imagerowService *imagerowsvc.Service
	trainingService *trainingsvc.Service
	downloadSVC     *download.Service
)

type TestCfg struct {
	DBurl         string `yaml:"dbURL"`
	PythonVenv    string
	PythonScripts string
}

func ConfigureFromENV() (*TestCfg, error) {
	var cfg TestCfg
	cfg.DBurl = os.Getenv("DB_URL")
	cfg.PythonScripts = os.Getenv("TASTEHEAD_SCRIPTS_DIR")
	cfg.PythonVenv = os.Getenv("TASTEHEAD_PYTHON_VENV_DIR")
	return &cfg, nil
}

func TestMain(m *testing.M) {
	cfg, err := ConfigureFromENV()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	absEnv := toAbsStruct{
		ModelDir:    svcPaths.ModelDir,
		DownloadDir: svcPaths.DownloadDir,
		VenvDir:     cfg.PythonVenv,
		ScriptsDir:  cfg.PythonScripts,
		ImportDir:   svcPaths.ImportDir,
		TrainingDir: `testdata\training`,
	}

	if err := makeAbs(&absEnv); err != nil {
		log.Fatalf("toAbs: %v", err)
	}

	pool, err = testutil.SetupPgxPool(cfg.DBurl)
	if err != nil {
		log.Fatalf("pgxPool: %v", err)
	}
	defer pool.Close()

	pyPaths := htmlparser.EnvPaths{
		PythonVenvDir: absEnv.VenvDir,
		ScriptsDir:    absEnv.ScriptsDir,
	}

	svcPaths.DownloadDir = absEnv.DownloadDir
	svcPaths.ImportDir = absEnv.ImportDir
	svcPaths.ModelDir = absEnv.ModelDir

	scraperRepo = htmlparser.NewRepository(pyPaths)
	imagerowRepo = imgrowpg.NewRepository(pool)
	trainingRepo = trainingpg.NewRepository(pool)

	imagerowService, err = imagerowsvc.NewService(imagerowRepo)
	if err != nil {
		log.Fatalf("imgrow.svc: %v", err)
	}

	trainingService, err = trainingsvc.NewService(trainingRepo, absEnv.TrainingDir)
	if err != nil {
		log.Fatalf("training.svc: %v", err)
	}

	downloadReq := download.NewServiceReq{
		ScraperRepo: scraperRepo,
		ImageSVC:    imagerowService,
		TrainingSVC: trainingService,
		Paths:       svcPaths,
	}

	downloadSVC, err = download.NewService(downloadReq)
	if err != nil {
		log.Fatalf("download.svc: %v", err)
	}

	code := m.Run()
	os.Exit(code)
}

func makeAbs(env *toAbsStruct) error {
	var err error

	env.DownloadDir, err = absPath(env.DownloadDir, "download dir")
	if err != nil {
		return err
	}

	env.ModelDir, err = absPath(env.ModelDir, "model dir")
	if err != nil {
		return err
	}

	env.ImportDir, err = absPath(env.ImportDir, "import dir")
	if err != nil {
		return err
	}

	env.ScriptsDir, err = absPath(env.ScriptsDir, "scripts dir")
	if err != nil {
		return err
	}

	env.TrainingDir, err = absPath(env.TrainingDir, "training dir")
	if err != nil {
		return err
	}

	env.VenvDir, err = absPath(env.VenvDir, "python venv dir")
	if err != nil {
		return err
	}

	return nil
}

func absPath(p string, name string) (string, error) {
	if p == "" {
		return "", fmt.Errorf("%s is empty", name)
	}

	abs, err := filepath.Abs(p)
	if err != nil {
		return "", fmt.Errorf("abs %s: %w", name, err)
	}

	return filepath.Clean(abs), nil
}
