package main

import (
	"context"
	"fmt"
	"os/signal"
	"scraper/internal/config"
	"scraper/internal/pkg/db"
	imgrowpg "scraper/internal/repository/imageRow/pg"
	htmlparser "scraper/internal/repository/scraper/html"
	trainingpg "scraper/internal/repository/training/pg"
	"scraper/internal/server"
	"scraper/internal/service/download"
	imagerowsvc "scraper/internal/service/imageRow"
	trainingsvc "scraper/internal/service/training"
	"syscall"

	"github.com/rs/zerolog/log"
)

const modelNameConfigPath = `./config/model.yaml`

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	appConfig, err := config.GetAppConfig()
	if err != nil {
		log.Fatal().Err(fmt.Errorf("GetAppConfig: %w", err)).Msg("main")
	}
	pyPaths := htmlparser.EnvPaths{
		PythonVenvDir: appConfig.Env.PythonVenvDir,
		ScriptsDir:    appConfig.Env.ScriptsDir,
	}

	modelName, err := config.GetModelName(modelNameConfigPath)
	if err != nil {
		log.Fatal().Err(fmt.Errorf("GetModelName: %w", err))
	}

	svcPaths := download.EnvPaths{
		DownloadDir:         appConfig.Env.DownloadDir,
		ImportDir:           appConfig.Env.ImportDir,
		ModelDir:            appConfig.Env.ModelDir,
		ModelName:           modelName,
		ModelNameConfigPath: modelNameConfigPath,
	}

	pool, err := db.InitDB(ctx, appConfig.DB.URL)
	if err != nil {
		log.Fatal().Err(fmt.Errorf("initDB: %w", err)).Msg("main")
	}

	scrapeCFG := htmlparser.ScrapeConfig{
		UserAgent:          appConfig.Scrape.UserAgent,
		DownloadImagePause: appConfig.Scrape.DownloadImagePause,
		DownloadPagePause:  appConfig.Scrape.DownloadPagePause,
	}

	scraperRepo := htmlparser.NewRepository(pyPaths, scrapeCFG)
	imagerowRepo := imgrowpg.NewRepository(pool)
	trainingrepo := trainingpg.NewRepository(pool)

	imgRowService, err := imagerowsvc.NewService(imagerowRepo)
	if err != nil {
		log.Fatal().Err(fmt.Errorf("imgRowservice: %w", err)).Msg("main")
	}

	trainingService, err := trainingsvc.NewService(trainingrepo, appConfig.Env.TrainingDir)
	if err != nil {
		log.Fatal().Err(fmt.Errorf("train svc:%w", err)).Msg("main")
	}

	downloadSVCReq := download.NewServiceReq{
		ScraperRepo: scraperRepo,
		ImageSVC:    imgRowService,
		TrainingSVC: trainingService,
		Paths:       svcPaths,
	}

	downloadSVC, err := download.NewService(downloadSVCReq)
	if err != nil {
		log.Fatal().Err(fmt.Errorf("downloadSVC: %w", err)).Msg("main")
	}

	opts := server.ServerOpts{
		Port:      appConfig.Server.Port,
		ScrapeSVC: downloadSVC,
		TrainSVC:  trainingService,
		ImageSVC:  imgRowService,
	}

	if err := server.StartServer(ctx, opts); err != nil {
		log.Fatal().Err(fmt.Errorf("server: %w", err)).Msg("main")
	}

	log.Info().Msg("cancelled successfully")
}
