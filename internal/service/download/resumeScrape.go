package download

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"scraper/internal/pkg/apperror"
	"scraper/internal/pkg/kit"
	scrapestate "scraper/internal/scrapeState"
	imagerowsvc "scraper/internal/service/imageRow"
	"time"

	"github.com/rs/zerolog/log"
)

type ResumeScrapeReq struct {
	BatchName string `validate:"required"`
}

type pageDownloadResult struct {
	NextURL string
	Fails   []FailedDownload
}

type SolveFailsReq struct {
	DirName string `validate:"required"`
}

func (s *Service) ResumeScrape(ctx context.Context, req ResumeScrapeReq) (ScrapeImagesResp, error) {
	if err := kit.ValidateStruct(s.validate, req); err != nil {
		return ScrapeImagesResp{}, fmt.Errorf("validate struct: %w: %v", apperror.ErrBadRequest, err)
	}

	dirPath := filepath.Join(s.paths.DownloadDir, req.BatchName)
	state, err := scrapestate.ReadStateManifest(dirPath)
	if err != nil {
		return ScrapeImagesResp{}, fmt.Errorf("read state dir: %w", err)
	}

	scrapeDesc, err := scrapestate.ReadSсrapeManifest(dirPath)
	if err != nil {
		return ScrapeImagesResp{}, fmt.Errorf("read scrape manifest:%w", err)
	}

	parseReq := ParsePageReq{
		URL:              state.CurrentPage,
		PostSelector:     scrapeDesc.PostSelector,
		ImageAttr:        scrapeDesc.ImageAttr,
		NextPageSelector: scrapeDesc.NextPageSelector,
		NextPageAttr:     scrapeDesc.NextPageAttr,
	}

	var downloadFails []Fails
	if state.PagesRemaining != 0 {
		fails, err := s.parseAndDownload(dirPath, parseReq, state, state.PagesRemaining)
		if err != nil {
			return ScrapeImagesResp{}, fmt.Errorf("parseAndDownload: %w", err)
		}
		downloadFails = fails
	}

	var batchID int64
	if !state.IsProcessed {
		csvRows, err := s.processDownloadedFiles(ctx, dirPath)
		if err != nil {
			return ScrapeImagesResp{}, fmt.Errorf("processDownloadedFiles: %w", err)
		}
		dirName := filepath.Base(dirPath)

		id, err := s.writeToNewBatch(ctx, dirName, csvRows)
		if err != nil {
			return ScrapeImagesResp{}, fmt.Errorf("writeToNewBatch: %w", err)
		}
		batchID = id

		if err := state.MarkProcessed(); err != nil {
			return ScrapeImagesResp{}, fmt.Errorf("mark processed: %w", err)
		}
	}

	resp := ScrapeImagesResp{
		BatchID:     batchID,
		ErrDownload: downloadFails,
	}

	return resp, nil
}

func (s *Service) ListUnfinishedBatches() ([]string, error) {
	dirs, err := s.scraperRepo.ListUnfinishedBatches(s.paths.DownloadDir)
	if err != nil {
		return nil, fmt.Errorf("ListUnfinishedBatches: %w", err)
	}

	return dirs, nil
}

func (s *Service) ListBatchesWithFails() ([]string, error) {
	dirs, err := s.scraperRepo.ListBatchesWithFails(s.paths.DownloadDir)
	if err != nil {
		return nil, fmt.Errorf("ListBatchesWithFails: %w", err)
	}

	return dirs, nil
}

func (s *Service) SolveFails(ctx context.Context, req SolveFailsReq) (allFails []Fails, err error) {
	if err := kit.ValidateStruct(s.validate, req); err != nil {
		return nil, fmt.Errorf("validate struct: %w: %v", apperror.ErrBadRequest, err)
	}
	dirPath := filepath.Join(s.paths.DownloadDir, req.DirName)

	failReport, err := scrapestate.ReadFails(dirPath)
	if err != nil {
		return nil, fmt.Errorf("read fails: %w", err)
	}

	parseInfo, err := scrapestate.ReadSсrapeManifest(dirPath)
	if err != nil {
		return nil, fmt.Errorf("ReadScrapeManifest: %w", err)
	}

	retryName := "_retry" + time.Now().Format("2006-01-02_15-04-05")
	retryPath := filepath.Join(dirPath, retryName)

	if err := os.Mkdir(retryPath, 0o755); err != nil {
		return nil, fmt.Errorf("create retry dir: %w", err)
	}

	commited := false
	defer func() {
		if err == nil || commited {
			return
		}

		if err := os.RemoveAll(retryPath); err != nil {
			log.Error().Err(fmt.Errorf("remove retryDir after fail:%w", err)).Msg("SolveFails")
		}
	}()

	allFails = make([]Fails, 0)

	for _, pageFail := range failReport {
		directFails, needReparse := s.retryURLsFromFails(pageFail, retryPath)

		if needReparse {
			parseReq := ParsePageReq{
				URL:              pageFail.OriginURL,
				PostSelector:     parseInfo.PostSelector,
				ImageAttr:        parseInfo.ImageAttr,
				NextPageSelector: parseInfo.NextPageSelector,
				NextPageAttr:     parseInfo.NextPageAttr,
			}

			result, err := s.downloadPageOnce(retryPath, parseReq)
			if err != nil {
				return nil, fmt.Errorf("redownload page: %w", err)
			}

			directFails = result.Fails
		}

		if len(directFails) > 0 {
			allFails = append(allFails, Fails{
				OriginURL: pageFail.OriginURL,
				Fails:     directFails,
			})
			reportDownloads := remapToJSONFailedDownload(directFails)

			reportFail := scrapestate.JSONFailReport{
				OriginURL: pageFail.OriginURL,
				Fails:     reportDownloads,
			}
			if err := scrapestate.AppendFailReport(retryPath, reportFail); err != nil {
				return nil, fmt.Errorf("AppendFailReport: %w", err)
			}
		}
	}

	csvRows, err := s.processDownloadedFiles(ctx, retryPath)
	if err != nil {
		return nil, fmt.Errorf("processDownloadedFiles: %w", err)
	}

	if len(csvRows) == 0 {
		if len(allFails) != 0 {
			if err := s.scraperRepo.MoveRetryDirToParent(retryPath); err != nil {
				return nil, fmt.Errorf("MoveRetryDirToParent: %w", err)
			}
			return nil, nil
		}
		failPath := filepath.Join(dirPath, "fails.jsonl")
		if err := os.Remove(failPath); err != nil {
			return nil, fmt.Errorf("remove parent fails.jsonl:%w", err)
		}
		if err := os.Remove(retryPath); err != nil {
			return nil, fmt.Errorf("remove retry dir: %w", err)
		}
		return nil, nil
	}

	getReq := imagerowsvc.GetBatchReq{
		Name: req.DirName,
	}

	respBatch, err := s.imageSVC.GetBatch(ctx, getReq)
	if err != nil {
		return nil, fmt.Errorf("GetBatch: %w", err)
	}

	if err := s.writeToExistingBatch(ctx, respBatch.ID, csvRows); err != nil {
		return nil, fmt.Errorf("writeToExistingBatch: %w", err)
	}

	if err := s.scraperRepo.MoveRetryDirToParent(retryPath); err != nil {
		return nil, fmt.Errorf("MoveRetryDirToParent: %w", err)
	}
	commited = true

	if len(allFails) == 0 {
		failPath := filepath.Join(dirPath, "fails.jsonl")
		if err := os.Remove(failPath); err != nil {
			return nil, fmt.Errorf("remove parent fails.jsonl:%w", err)
		}
		return nil, nil
	}

	return allFails, nil
}

func (s *Service) retryURLsFromFails(pageFail scrapestate.JSONFailReport, retryPath string) ([]FailedDownload, bool) {
	allFails := make([]FailedDownload, 0)

	for _, f := range pageFail.Fails {
		if !f.Repeatable {
			continue
		}

		downloadFail := s.scraperRepo.DownloadPic(retryPath, f.URL)
		if downloadFail == nil {
			continue
		}

		apiFail := remapToFailedDownloadSVC(*downloadFail)

		if downloadFail.Warn == apperror.IssueGetRequestError {
			return nil, true
		}

		allFails = append(allFails, apiFail)
	}

	return allFails, false
}

func (s *Service) downloadPageOnce(dirPath string, parseReq ParsePageReq) (pageDownloadResult, error) {
	parseResp, err := s.parsePage(parseReq)
	if err != nil {
		return pageDownloadResult{}, fmt.Errorf("parse page: %w", err)
	}

	downloadFails, err := s.downloadPics(dirPath, parseResp.Urls)
	if err != nil {
		return pageDownloadResult{}, fmt.Errorf("download pics: %w", err)
	}

	result := pageDownloadResult{
		NextURL: parseResp.NextURl,
		Fails:   downloadFails,
	}

	return result, nil
}
