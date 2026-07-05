package download

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"scraper/internal/config"
	customvalidator "scraper/internal/customValidator"
	"scraper/internal/pkg/apperror"
	"scraper/internal/pkg/kit"
	imagerow "scraper/internal/repository/imageRow"
	"scraper/internal/repository/scraper"
	scrapestate "scraper/internal/scrapeState"
	imagerowsvc "scraper/internal/service/imageRow"
	trainingsvc "scraper/internal/service/training"
	"strings"
	"time"
	"unicode"

	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"
)

type Service struct {
	scraperRepo scraper.Repository
	imageSVC    *imagerowsvc.Service
	trainingSVC *trainingsvc.Service
	validate    *validator.Validate
	paths       EnvPaths
}

type NewServiceReq struct {
	ScraperRepo scraper.Repository
	ImageSVC    *imagerowsvc.Service
	TrainingSVC *trainingsvc.Service
	Paths       EnvPaths
}

type EnvPaths struct {
	ModelName           string
	DownloadDir         string
	ModelDir            string
	ImportDir           string
	ModelNameConfigPath string
}

func NewService(req NewServiceReq) (*Service, error) {
	validator, err := initValidator()
	if err != nil {
		return &Service{}, fmt.Errorf("init validator")
	}

	return &Service{
		scraperRepo: req.ScraperRepo,
		imageSVC:    req.ImageSVC,
		trainingSVC: req.TrainingSVC,
		validate:    validator,
		paths:       req.Paths,
	}, nil
}

type ScrapeImagesReq struct {
	URL              string `validate:"required,url"`
	PostSelector     string `validate:"required"`
	ImageAttr        string `validate:"required"`
	NextPageSelector string `validate:"required"`
	NextPageAttr     string `validate:"required"`
	Pages            int    `validate:"required,min=1"`
	Limit            int    `validate:"required,min=0"`
}

type File struct {
	Dir     string `validate:"required"`
	Hash    string `validate:"required,relative"`
	RelPath string `validate:"required,relative"`
}

type GetImagesReq struct {
	Path string `validate:"required,relative"`
}

type GetImagesResp struct {
	FullPath string
}

type ImgPath struct {
	Dir  string `validate:"required,relative"`
	Path string `validate:"required,relative"`
}

type LocateDuplicatesResp struct {
	Hash       string
	First      File
	Duplicates []File
}

type DeleteImagesReq struct {
	Files []File `validate:"min=1,dive"`
}

type TrainModelReq struct {
	ModelName string `validate:"required,safeRelPath"`
	TagID     int    `validate:"min=1"`
}

type ParsePageReq struct {
	URL              string
	PostSelector     string
	ImageAttr        string
	NextPageSelector string
	NextPageAttr     string
}

type ParsePageResp struct {
	NextURl string
	Urls    []string
}

type ScrapeImagesResp struct {
	BatchID     int64
	ErrDownload []Fails
}

type Fails struct {
	OriginURL string           `json:"origin_url"`
	Fails     []FailedDownload `json:"fails"`
}

type FailedDownload struct {
	Issue      apperror.DownloadIssue `json:"issue"`
	URL        string                 `json:"url"`
	Repeatable bool                   `json:"repeatable"`
}

type ParseImportDirs struct {
	Paths []string `validate:"min=1"`
}

type splitFilesResp struct {
	KnownFiles  []scraper.File
	UniqueFiles []scraper.File
}

type computeDestPathResp struct {
	Path     string
	BatchID  int64
	Issue    ImportIssue
	IsUpdate bool
}

type MoveImagesReq struct {
	IDs       []int64 `validate:"required,min=1,dive,min=1"`
	ToBatchID int64   `validate:"min=1"`
}

type MoveImagesResp struct {
	Rejects []Reject
}

type Reject struct {
	RejectedID int64
	Reason     string
}

type HardDeleteImageResp struct {
	Rejects []Reject
}

type HardDeleteBatchReq struct {
	IDs []int64 `validate:"min=1,dive,min=1"`
}

type HardDeleteImagesReq struct {
	IDs []int64 `validate:"min=1,dive,min=1"`
}

type ReadBatchScrapeStateReq struct {
	BatchID int `validate:"min=1"`
}

type CreateBatchReq struct {
	Name string `validate:"safeRelPath"`
}

type ReadBatchScrapeStateResp struct {
	NextURL          string
	PostSelector     string
	ImageAttr        string
	NextPageSelector string
	NextPageAttr     string
}

type ListImagesWithTagsReq struct {
	DirID     *int64
	Score     *float32
	ScoreType string
	Limit     int   `validate:"min=1,max=100"`
	Cursor    int64 `validate:"min=0"`
	Next      bool
}

type ListImagesWithTagsResp struct {
	CursorNext int64
	CursorPrev int64
	Images     []RowWithTags
	HasMore    bool
}

type RowWithTags struct {
	ID         int64
	Hash       string
	BatchID    int64
	RelPath    string
	ModelScore float32
	UserScore  *float32
	Tags       []string
}

type SetModelReq struct {
	ModelName string `validate:"safeModelName"`
}

func (s *Service) ScrapePages(ctx context.Context, req ScrapeImagesReq) (ScrapeImagesResp, error) {
	if err := kit.ValidateStruct(s.validate, req); err != nil {
		return ScrapeImagesResp{}, fmt.Errorf("validate struct: %w: %v", apperror.ErrBadRequest, err)
	}

	dirName := time.Now().Format("2006-01-02_15-04-05")
	dirPath := filepath.Join(s.paths.DownloadDir, dirName)
	if err := os.Mkdir(dirPath, 0766); err != nil {
		return ScrapeImagesResp{}, fmt.Errorf("make download dir: %w", err)
	}

	scrapeMan := scrapestate.ScrapeManifest{
		PostSelector:     req.PostSelector,
		ImageAttr:        req.ImageAttr,
		NextPageSelector: req.NextPageSelector,
		NextPageAttr:     req.NextPageAttr,
	}
	if err := scrapestate.CreateScrapeManifest(dirPath, scrapeMan); err != nil {
		return ScrapeImagesResp{}, fmt.Errorf("createScrapeManifest:%w", err)
	}

	parseReq := ParsePageReq{
		URL:              req.URL,
		PostSelector:     req.PostSelector,
		ImageAttr:        req.ImageAttr,
		NextPageSelector: req.NextPageSelector,
		NextPageAttr:     req.NextPageAttr,
	}
	state, err := scrapestate.NewStateManifest(dirPath)
	if err != nil {
		return ScrapeImagesResp{}, fmt.Errorf("NewStateManifest: %w", err)
	}
	fails, err := s.parseAndDownload(dirPath, parseReq, state, req.Pages)
	if err != nil {
		return ScrapeImagesResp{}, fmt.Errorf("parseAndDownload:%w", err)
	}

	csvRows, err := s.processDownloadedFiles(ctx, dirPath)
	if err != nil {
		return ScrapeImagesResp{}, fmt.Errorf("processDownloadedFiles: %w", err)
	}

	batchID, err := s.writeToNewBatch(ctx, dirName, csvRows)
	if err != nil {
		return ScrapeImagesResp{}, fmt.Errorf("writeToNewBatch: %w", err)
	}

	if err := state.MarkProcessed(); err != nil {
		return ScrapeImagesResp{}, fmt.Errorf("mark processed: %w", err)
	}

	resp := ScrapeImagesResp{
		BatchID:     batchID,
		ErrDownload: fails,
	}

	return resp, nil
}

func (s *Service) GetImages(req GetImagesReq) (GetImagesResp, error) {
	if err := kit.ValidateStruct(s.validate, req); err != nil {
		return GetImagesResp{}, fmt.Errorf("validate struct: %w: %v", apperror.ErrBadRequest, err)
	}
	fullPath := filepath.Join(s.paths.DownloadDir, req.Path)

	resp := GetImagesResp{
		FullPath: fullPath,
	}

	return resp, nil
}

func (s *Service) DeleteImages(ctx context.Context, req DeleteImagesReq) error {
	if err := kit.ValidateStruct(s.validate, req); err != nil {
		return fmt.Errorf("validate struct: %w: %v", apperror.ErrBadRequest, err)
	}

	deleteErr := make([]apperror.Warning, 0)
	for _, file := range req.Files {

		imgPath := filepath.Join(s.paths.DownloadDir, file.Dir, file.RelPath)
		hash, err := s.scraperRepo.ComputeFileHash(imgPath)
		if err != nil {
			de := apperror.Warning{
				Msg: fmt.Sprintf("path: %s/%s\n , hash: %s\n\n", file.Dir, file.RelPath, file.Hash),
				Err: fmt.Errorf("ComputeFileHash: %w\n", err),
			}
			deleteErr = append(deleteErr, de)
			continue
		}

		svcReq := imagerowsvc.GetRowByHashReq{
			Hash: hash,
		}

		_, err = s.imageSVC.GetRowByHash(ctx, svcReq)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			de := apperror.Warning{
				Msg: fmt.Sprintf("path: %s/%s\n , hash: %s\n\n", file.Dir, file.RelPath, file.Hash),
				Err: fmt.Errorf("GetRowByHash: %w\n", err),
			}
			deleteErr = append(deleteErr, de)
			continue
		}
		if err == nil {
			de := apperror.Warning{
				Msg: fmt.Sprintf("path: %s/%s\n , hash: %s\n\n", file.Dir, file.RelPath, file.Hash),
				Err: fmt.Errorf("img: %w\n", apperror.ErrAlreadyExists),
			}
			deleteErr = append(deleteErr, de)
			continue
		}

		imgPath = filepath.Join(s.paths.DownloadDir, file.Dir, file.RelPath)
		if err := s.scraperRepo.DeleteImages(imgPath); err != nil {
			de := apperror.Warning{
				Msg: fmt.Sprintf("path: %s\n", imgPath),
				Err: fmt.Errorf("remove file: %w\n\n", err),
			}
			deleteErr = append(deleteErr, de)
			continue
		}
	}

	if len(deleteErr) != 0 {
		return fmt.Errorf("remove imgs: %w", apperror.NewRemoveImgsError(deleteErr))
	}

	return nil
}

func (s *Service) findAndSplitFiles(ctx context.Context, dirPath string) (splitFilesResp, error) {
	files, err := s.scraperRepo.ComputeHashesInDir(dirPath)
	if err != nil {
		return splitFilesResp{}, fmt.Errorf("computeHashesInDir: %w", err)
	}

	hashes := make([]string, 0, len(files))
	for _, file := range files {
		hashes = append(hashes, file.Hash)
	}

	_, knownHashes, err := s.sortPics(ctx, hashes)
	if err != nil {
		return splitFilesResp{}, fmt.Errorf("returnUniquePics: %w", err)
	}

	knownMap := make(map[string]struct{})
	for _, hash := range knownHashes {
		knownMap[hash] = struct{}{}
	}

	unique := make([]scraper.File, 0, len(files)-len(knownHashes))
	known := make([]scraper.File, 0, len(knownHashes))
	for _, file := range files {
		_, ok := knownMap[file.Hash]
		if ok {
			known = append(known, file)
			continue
		}
		unique = append(unique, file)

	}

	resp := splitFilesResp{
		UniqueFiles: unique,
		KnownFiles:  known,
	}

	return resp, nil
}

func (s *Service) TrainModel(ctx context.Context, req TrainModelReq) error {
	if err := kit.ValidateStruct(s.validate, req); err != nil {
		return fmt.Errorf("validate struct: %w: %v", apperror.ErrBadRequest, err)
	}

	exportReq := trainingsvc.ExportAsSVCReq{
		Tag: req.TagID,
	}

	csvPath, err := s.trainingSVC.ExportAsSVC(ctx, exportReq)
	if err != nil {
		return fmt.Errorf("ExportAsSVC: %w", err)
	}

	outputPath := filepath.Join(s.paths.ModelDir, req.ModelName+".pt")
	trainReq := scraper.TrainModelReq{
		CsvPath:      csvPath,
		DownloadPath: s.paths.DownloadDir,
		OutputPath:   outputPath,
	}
	if err := s.scraperRepo.TrainModel(ctx, trainReq); err != nil {
		return fmt.Errorf("TrainModel: %w", err)
	}

	return nil
}

func (s *Service) ListModels() ([]string, error) {
	models, err := s.scraperRepo.ListModels(s.paths.ModelDir)
	if err != nil {
		return nil, fmt.Errorf("ListModels: %w", err)
	}

	return models, nil
}

func (s *Service) SetModel(req SetModelReq) error {
	if err := kit.ValidateStruct(s.validate, req); err != nil {
		return fmt.Errorf("validate struct: %w: %v", apperror.ErrBadRequest, err)
	}

	modelPath := filepath.Join(s.paths.ModelDir, req.ModelName)
	if _, err := os.Stat(modelPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("model: %w", apperror.ErrNotFound)
		}
		return fmt.Errorf("stat: %w", err)
	}

	if err := config.SetModelName(req.ModelName, s.paths.ModelNameConfigPath); err != nil {
		return fmt.Errorf("SetModelName: %w", err)
	}

	s.paths.ModelName = req.ModelName
	return nil
}

func (s *Service) MoveImages(ctx context.Context, req MoveImagesReq) (MoveImagesResp, error) {
	if err := kit.ValidateStruct(s.validate, req); err != nil {
		return MoveImagesResp{}, fmt.Errorf("validate struct: %w: %v", apperror.ErrBadRequest, err)
	}
	batchReq := imagerowsvc.GetBatchByIDReq{
		ID: req.ToBatchID,
	}

	destBatch, err := s.imageSVC.GetBatchByID(ctx, batchReq)
	if err != nil {
		return MoveImagesResp{}, fmt.Errorf("GetBatchByID: %w", err)
	}

	rejectedIDs := make([]Reject, 0)
	destDir := filepath.Join(s.paths.DownloadDir, destBatch.RelPath)
	thumbDestDir := filepath.Join(destDir, "thumbs")

	for _, id := range req.IDs {
		svcReq := imagerowsvc.GetRowByIDReq{
			ID: id,
		}

		row, err := s.imageSVC.GetRowByID(ctx, svcReq)
		if err != nil {
			if errors.Is(err, apperror.ErrNotFound) {
				rejected := Reject{
					RejectedID: id,
					Reason:     "not found",
				}
				rejectedIDs = append(rejectedIDs, rejected)
				continue
			}

			return MoveImagesResp{}, fmt.Errorf("GetRowByID: %w", err)
		}

		batchReq := imagerowsvc.GetBatchByIDReq{
			ID: row.BatchID,
		}

		fromBatch, err := s.imageSVC.GetBatchByID(ctx, batchReq)
		if err != nil {
			return MoveImagesResp{}, fmt.Errorf("GetBatchByID: %w", err)
		}

		srcPath := filepath.Join(s.paths.DownloadDir, fromBatch.RelPath, row.RelPath)
		destPath := filepath.Join(destDir, row.RelPath)
		if err := os.Rename(srcPath, destPath); err != nil {
			return MoveImagesResp{}, fmt.Errorf("move file: %w", err)
		}

		if err := os.MkdirAll(thumbDestDir, 0o755); err != nil {
			return MoveImagesResp{}, fmt.Errorf("mkdirall thumb path: %w", err)
		}
		srcThumbPath := filepath.Join(s.paths.DownloadDir, fromBatch.RelPath, "thumbs", row.RelPath)
		destThumbPath := filepath.Join(destDir, "thumbs", row.RelPath)
		if err := os.Rename(srcThumbPath, destThumbPath); err != nil {
			return MoveImagesResp{}, fmt.Errorf("move thumb file: %w", err)
		}

		updateReq := imagerowsvc.UpdateRowBatchReq{
			ID:      id,
			BatchID: destBatch.ID,
		}

		if err := s.imageSVC.UpdateRowBatchID(ctx, updateReq); err != nil {
			return MoveImagesResp{}, fmt.Errorf("UpdateRowBatchID: %w", err)
		}
	}

	if len(rejectedIDs) != 0 {
		resp := MoveImagesResp{
			Rejects: rejectedIDs,
		}
		return resp, nil
	}

	return MoveImagesResp{}, nil
}

func (s *Service) CreateBatch(ctx context.Context, req CreateBatchReq) (int64, error) {
	if err := kit.ValidateStruct(s.validate, req); err != nil {
		return 0, fmt.Errorf("validate struct: %w: %v", apperror.ErrBadRequest, err)
	}

	batchPath, err := safePath(s.paths.DownloadDir, req.Name)
	if err != nil {
		return 0, fmt.Errorf("safePath: %w", err)
	}

	if err := os.Mkdir(batchPath, 0o755); err != nil {
		return 0, fmt.Errorf("mkdir: %w", err)
	}

	createReq := imagerowsvc.CreateBatchReq{
		Name:    req.Name,
		RelPath: req.Name,
	}

	batchID, err := s.imageSVC.CreateBatch(ctx, createReq)
	if err != nil {
		return 0, fmt.Errorf("CreateBatch: %w", err)
	}
	return batchID, nil
}

func (s *Service) ReadBatchScrapeState(ctx context.Context, req ReadBatchScrapeStateReq) (ReadBatchScrapeStateResp, error) {
	if err := kit.ValidateStruct(s.validate, req); err != nil {
		return ReadBatchScrapeStateResp{}, fmt.Errorf("validate struct: %w: %v", apperror.ErrBadRequest, err)
	}
	batchReq := imagerowsvc.GetBatchByIDReq{
		ID: int64(req.BatchID),
	}

	batch, err := s.imageSVC.GetBatchByID(ctx, batchReq)
	if err != nil {
		return ReadBatchScrapeStateResp{}, fmt.Errorf("GetBatchByID: %w", err)
	}

	batchPath, err := safePath(s.paths.DownloadDir, batch.RelPath)
	if err != nil {
		return ReadBatchScrapeStateResp{}, fmt.Errorf("safe path: %w", err)
	}

	state, err := scrapestate.ReadStateManifest(batchPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ReadBatchScrapeStateResp{}, fmt.Errorf("ReadStateManifest: %w", apperror.ErrNotFound)
		}
		return ReadBatchScrapeStateResp{}, fmt.Errorf("ReadStateManifest: %w", err)
	}
	scrapeManifest, err := scrapestate.ReadSсrapeManifest(batchPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ReadBatchScrapeStateResp{}, fmt.Errorf("ReadScrapeManifest: %w", apperror.ErrNotFound)
		}
		return ReadBatchScrapeStateResp{}, fmt.Errorf("ReadScrapeManifest: %w", err)
	}

	resp := ReadBatchScrapeStateResp{
		NextURL:          state.NextURLPage,
		PostSelector:     scrapeManifest.PostSelector,
		ImageAttr:        scrapeManifest.ImageAttr,
		NextPageSelector: scrapeManifest.NextPageSelector,
		NextPageAttr:     scrapeManifest.NextPageAttr,
	}

	return resp, nil
}

func (s *Service) HardDeleteImages(ctx context.Context, req HardDeleteImagesReq) (HardDeleteImageResp, error) {
	if err := kit.ValidateStruct(s.validate, req); err != nil {
		return HardDeleteImageResp{}, fmt.Errorf("validate struct: %w: %v", apperror.ErrBadRequest, err)
	}

	rejects := make([]Reject, 0)
	successedIDs := make([]int64, 0, len(req.IDs))

	listReq := imagerowsvc.ListImagesForDeleteReq{
		IDs: req.IDs,
	}

	images, err := s.imageSVC.ListImagesForDelete(ctx, listReq)
	if err != nil {
		return HardDeleteImageResp{}, fmt.Errorf("ListImagesForDelete:%w", err)
	}

	for _, image := range images {
		imgPath, err := safePath(s.paths.DownloadDir, image.BatchRelPath, image.ImageRelPath)
		if err != nil {
			reject := Reject{
				RejectedID: image.ID,
				Reason:     err.Error(),
			}
			rejects = append(rejects, reject)
			continue
		}
		if err := os.Remove(imgPath); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return HardDeleteImageResp{}, fmt.Errorf("os remove file: %w", err)
		}

		imgThumbPath, err := safePath(s.paths.DownloadDir, image.BatchRelPath, "thumbs", image.ImageRelPath)
		if err != nil {
			reject := Reject{
				RejectedID: image.ID,
				Reason:     err.Error(),
			}
			rejects = append(rejects, reject)
			continue
		}
		if err := os.Remove(imgThumbPath); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return HardDeleteImageResp{}, fmt.Errorf("os remove file: %w", err)
		}
		successedIDs = append(successedIDs, image.ID)
	}

	deleteReq := imagerowsvc.HardDeleteImagesReq{
		IDs: successedIDs,
	}
	if err := s.imageSVC.HardDeleteImages(ctx, deleteReq); err != nil {
		return HardDeleteImageResp{}, fmt.Errorf("HardDeleteImages:%w", err)
	}

	if len(rejects) == 0 {
		return HardDeleteImageResp{nil}, nil
	}

	resp := HardDeleteImageResp{
		Rejects: rejects,
	}

	return resp, nil
}

func (s *Service) HardDeleteBatches(ctx context.Context, req HardDeleteBatchReq) error {
	if err := kit.ValidateStruct(s.validate, req); err != nil {
		return fmt.Errorf("validate struct: %w: %v", apperror.ErrBadRequest, err)
	}

	getReq := imagerowsvc.ListDeletedBatchesByID{
		IDs: req.IDs,
	}

	batches, err := s.imageSVC.ListDeletedBatchesByID(ctx, getReq)
	if err != nil {
		return fmt.Errorf("ListDeletedBatchesByID: %w", err)
	}

	batchIDs := make([]int64, 0, len(batches))

	for _, batch := range batches {
		batchPath, err := safePath(s.paths.DownloadDir, batch.RelPath)
		if err != nil {
			return fmt.Errorf("build batch path: %w", err)
		}

		if err := os.RemoveAll(batchPath); err != nil {
			return fmt.Errorf("removeAll:%w", err)
		}

		batchIDs = append(batchIDs, batch.ID)
	}

	deleteReq := imagerowsvc.HardDeleteBatchesReq{
		IDs: batchIDs,
	}

	if err := s.imageSVC.HardDeleteBatches(ctx, deleteReq); err != nil {
		return fmt.Errorf("HardDeleteBatches: %w", err)
	}

	return nil
}

func (s *Service) ListImagesWithTags(ctx context.Context, req ListImagesWithTagsReq) (ListImagesWithTagsResp, error) {
	if err := kit.ValidateStruct(s.validate, req); err != nil {
		return ListImagesWithTagsResp{}, fmt.Errorf("validate struct: %w: %v", apperror.ErrBadRequest, err)
	}

	listReq := imagerowsvc.ListImagesReq{
		DirID:     req.DirID,
		Score:     req.Score,
		ScoreType: req.ScoreType,
		Limit:     req.Limit,
		Cursor:    req.Cursor,
		Next:      req.Next,
	}
	imageResp, err := s.imageSVC.ListImages(ctx, listReq)
	if err != nil {
		return ListImagesWithTagsResp{}, fmt.Errorf("ListImages: %w", err)
	}

	ids := make([]int64, 0, len(imageResp.Images))
	for _, img := range imageResp.Images {
		ids = append(ids, img.ID)
	}

	listTagsIDReq := trainingsvc.ListTagsByImageIDsReq{
		IDs: ids,
	}

	tags, err := s.trainingSVC.ListTagsByImageIDs(ctx, listTagsIDReq)
	if err != nil {
		return ListImagesWithTagsResp{}, fmt.Errorf("ListTagsByImageIDs: %w", err)
	}

	rowsWithTags := makeRowWithTagsSVC(imageResp.Images, tags)

	resp := ListImagesWithTagsResp{
		CursorNext: imageResp.CursorNext,
		CursorPrev: imageResp.CursorPrev,
		Images:     rowsWithTags,
		HasMore:    imageResp.HasMore,
	}

	return resp, nil
}

func (s *Service) processDownloadedFiles(ctx context.Context, dirPath string) ([]scraper.CSVRow, error) {
	files, err := s.findAndSplitFiles(ctx, dirPath)
	if err != nil {
		return nil, fmt.Errorf("findAndSplitFiles: %w", err)
	}

	for _, file := range files.KnownFiles {
		imgPath := filepath.Join(s.paths.DownloadDir, file.Dir, file.Path)
		if err := s.scraperRepo.DeleteImages(imgPath); err != nil {
			log.Error().Err(fmt.Errorf("path:%s :%w", imgPath, err)).Msg("scrape pages")
		}
	}

	if len(files.UniqueFiles) == 0 {
		rows := make([]scraper.CSVRow, 0)
		return rows, nil
	}

	csvRows, err := s.parseDownloadedPics(ctx, dirPath)
	if err != nil {
		return nil, fmt.Errorf("parseDownloadedPics: %w", err)
	}

	return csvRows, nil
}

func (s *Service) parseAndDownload(
	dirPath string,
	req ParsePageReq,
	state *scrapestate.StateManifest,
	pages int,
) ([]Fails, error) {
	allFails := make([]Fails, 0)

	for i := range pages {
		if err := state.SetPage(pages-i, req.URL); err != nil {
			return nil, fmt.Errorf("begin page state: %w", err)
		}

		result, err := s.downloadPageOnce(dirPath, req)
		if err != nil {
			return nil, fmt.Errorf("downloadPageOnce")
		}

		if len(result.Fails) > 0 {
			apiReport := Fails{
				OriginURL: req.URL,
				Fails:     result.Fails,
			}

			jsonReport := scrapestate.JSONFailReport{
				OriginURL: req.URL,
				Fails:     remapToJSONFailedDownload(result.Fails),
			}

			allFails = append(allFails, apiReport)

			if err := scrapestate.AppendFailReport(dirPath, jsonReport); err != nil {
				return nil, fmt.Errorf("append fail report: %w", err)
			}

			time.Sleep(800 * time.Millisecond)
		}

		pagesAfterThis := pages - i - 1

		if result.NextURL == "" {
			if err := state.ClearRemainingPages(); err != nil {
				return nil, fmt.Errorf("clear remaining state pages: %w", err)
			}
			break
		}

		if pagesAfterThis == 0 {
			if err := state.SetNextURLPage(result.NextURL); err != nil {
				return nil, fmt.Errorf("setNextURLPage: %w", err)
			}
			if err := state.ClearRemainingPages(); err != nil {
				return nil, fmt.Errorf("clear remaining state pages: %w", err)
			}
			break
		}

		req.URL = result.NextURL
	}

	if len(allFails) == 0 {
		return nil, nil
	}

	return allFails, nil
}

func (s *Service) writeToNewBatch(ctx context.Context, dirName string, csvRows []scraper.CSVRow) (int64, error) {
	batch := imagerowsvc.CreateBatchReq{
		Name:    dirName,
		RelPath: dirName,
	}
	batchID, err := s.imageSVC.CreateBatch(ctx, batch)
	if err != nil {
		return 0, fmt.Errorf("create batch: %w", err)
	}

	createRows := remapCSVtoCreateRow(csvRows, batchID)
	createReq := imagerowsvc.CreateDownloadRowsReq{
		Rows: createRows,
	}
	if err := s.imageSVC.CreateDownloadRows(ctx, createReq); err != nil {
		return 0, fmt.Errorf("createFromBatch: %w", err)
	}

	updateBatchReq := imagerowsvc.UpdateBatchStatusReq{
		ID:     batchID,
		Status: string(imagerow.BatchStatusFinished),
	}
	if err := s.imageSVC.UpdateBatchStatus(ctx, updateBatchReq); err != nil {
		return 0, fmt.Errorf("updateBatchStatus: %w", err)
	}

	return batchID, nil
}

func (s *Service) writeToExistingBatch(ctx context.Context, batchID int64, csvRows []scraper.CSVRow) error {
	createRows := remapCSVtoCreateRow(csvRows, batchID)
	createReq := imagerowsvc.CreateDownloadRowsReq{
		Rows: createRows,
	}
	if err := s.imageSVC.CreateDownloadRows(ctx, createReq); err != nil {
		return fmt.Errorf("createDownloadRows: %w", err)
	}

	return nil
}

func (s *Service) parsePage(req ParsePageReq) (ParsePageResp, error) {
	repoReq := scraper.ParseReq{
		URL:              req.URL,
		PostSelector:     req.PostSelector,
		ImageAttr:        req.ImageAttr,
		NextPageSelector: req.NextPageSelector,
		NextPageAttr:     req.NextPageAttr,
	}

	urls := make([]string, 0)

	parseResp, err := s.scraperRepo.ParseHTML(repoReq)
	if err != nil {
		return ParsePageResp{}, fmt.Errorf("parseResp: %w", err)
	}
	urls = append(urls, parseResp.URL...)
	time.Sleep(1000 * time.Millisecond)

	nextURL, err := makeURL(parseResp.NextURL, repoReq.URL)
	if err != nil {
		return ParsePageResp{}, fmt.Errorf("makeURL: %w", err)
	}
	if nextURL == nil {
		resp := ParsePageResp{
			NextURl: "",
			Urls:    urls,
		}
		return resp, nil
	}
	log.Debug().Str("next url is:", nextURL.String()).Msg("parsed html")

	resp := ParsePageResp{
		NextURl: nextURL.String(),
		Urls:    urls,
	}

	return resp, nil
}

func (s *Service) downloadPics(dirPath string, urls []string) ([]FailedDownload, error) {
	failed := make([]FailedDownload, 0)
	for _, picURL := range urls {
		fail := s.scraperRepo.DownloadPic(dirPath, picURL)
		if fail != nil {
			svc := remapToFailedDownloadSVC(*fail)
			failed = append(failed, svc)
		}
	}

	if len(failed) != 0 {
		return failed, nil
	}

	return nil, nil
}

func (s *Service) parseDownloadedPics(ctx context.Context, dirPath string) ([]scraper.CSVRow, error) {
	processReq := scraper.ProcessFilesReq{
		ModelPath:    filepath.Join(s.paths.ModelDir, s.paths.ModelName),
		DownloadPath: dirPath,
	}
	if err := s.scraperRepo.ProcessFiles(ctx, processReq); err != nil {
		return nil, fmt.Errorf("process files: %w", err)
	}

	csvPath := filepath.Join(dirPath, "output.csv")
	csvRows, err := s.scraperRepo.CSVToRows(csvPath)
	if err != nil {
		return nil, fmt.Errorf("to csvRows: %w", err)
	}

	return csvRows, nil
}

func safePath(base string, elems ...string) (string, error) {
	baseClean := filepath.Clean(base)

	if !filepath.IsAbs(baseClean) {
		return "", fmt.Errorf("base path must be absolute: %q", base)
	}

	parts := make([]string, 0, len(elems)+1)
	parts = append(parts, baseClean)

	for _, elem := range elems {
		if elem == "" {
			return "", fmt.Errorf("empty path element")
		}

		if filepath.IsAbs(elem) {
			return "", fmt.Errorf("absolute path is not allowed: %q", elem)
		}

		clean := filepath.Clean(elem)

		if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(os.PathSeparator)) {
			return "", fmt.Errorf("unsafe relative path: %q", elem)
		}

		parts = append(parts, clean)
	}

	full := filepath.Join(parts...)

	rel, err := filepath.Rel(baseClean, full)
	if err != nil {
		return "", fmt.Errorf("rel path: %w", err)
	}

	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("path escapes base: %q", full)
	}

	return full, nil
}

func (s *Service) sortPics(ctx context.Context, hashes []string) (unique []string,
	known []string, err error,
) {
	svcReq := imagerowsvc.FindDuplicatesReq{
		Hashes: hashes,
	}

	knownHashes, err := s.imageSVC.FindDuplicatesByHash(ctx, svcReq)
	if err != nil {
		return nil, nil, fmt.Errorf("find existing hashes: %w", err)
	}

	knownMap := make(map[string]struct{}, len(hashes))
	for _, hash := range knownHashes {
		knownMap[hash] = struct{}{}
	}

	uniqueHashes := make([]string, 0, len(hashes)-len(knownHashes))
	for _, hash := range hashes {
		_, ok := knownMap[hash]
		if ok {
			continue
		}
		uniqueHashes = append(uniqueHashes, hash)
	}

	return uniqueHashes, knownHashes, nil
}

func remapToFailedDownloadSVC(repo scraper.FailedDownload) FailedDownload {
	return FailedDownload{
		Issue:      repo.Warn,
		URL:        repo.URL,
		Repeatable: repo.Repeatable,
	}
}

func remapCSVtoCreateRow(rows []scraper.CSVRow, batchID int64) []imagerowsvc.CreateFromBatchRow {
	imageRows := make([]imagerowsvc.CreateFromBatchRow, 0, len(rows))

	var imageRow imagerowsvc.CreateFromBatchRow
	for _, row := range rows {
		imageRow = imagerowsvc.CreateFromBatchRow{
			Hash:       row.Hash,
			BatchID:    batchID,
			Path:       row.Path,
			ModelScore: row.ModelScore,
			UserScore:  row.UserScore,
		}
		imageRows = append(imageRows, imageRow)
	}

	return imageRows
}

func remapToJSONFailedDownload(svc []FailedDownload) []scrapestate.JSONFailedDownload {
	fails := make([]scrapestate.JSONFailedDownload, 0, len(svc))
	var fail scrapestate.JSONFailedDownload
	for _, f := range svc {
		fail = scrapestate.JSONFailedDownload{
			Issue:      f.Issue,
			URL:        f.URL,
			Repeatable: f.Repeatable,
		}
		fails = append(fails, fail)
	}

	return fails
}

func makeRowWithTagsSVC(images []imagerowsvc.Row, tags trainingsvc.ImageIDstoTags) []RowWithTags {
	rowsWithTags := make([]RowWithTags, 0, len(images))

	for _, img := range images {
		rowTags := tags[img.ID]
		if rowTags == nil {
			rowTags = []string{}
		}
		rowWithTags := RowWithTags{
			ID:         img.ID,
			Hash:       img.Hash,
			BatchID:    img.BatchID,
			RelPath:    img.RelPath,
			ModelScore: img.ModelScore,
			UserScore:  img.UserScore,
			Tags:       rowTags,
		}
		rowsWithTags = append(rowsWithTags, rowWithTags)
	}

	return rowsWithTags
}

func makeURL(nextURL, currentURL string) (*url.URL, error) {
	nextURL = strings.TrimSpace(nextURL)
	if nextURL == "" {
		return nil, nil
	}

	next, err := url.Parse(nextURL)
	if err != nil {
		return nil, fmt.Errorf("parse next url: %w", err)
	}

	current, err := url.Parse(currentURL)
	if err != nil {
		return nil, fmt.Errorf("parse current url: %w", err)
	}

	if current.Host == "" || current.Scheme == "" || current.Path == "" {
		return nil, fmt.Errorf("currentURL,no host or scheme or path: %w", apperror.ErrBadRequest)
	}

	resolved := current.ResolveReference(next)
	if resolved.Scheme == "" || resolved.Host == "" {
		return nil, fmt.Errorf("resolved next url invalid: %w", apperror.ErrBadRequest)
	}

	return resolved, nil
}

func initValidator() (*validator.Validate, error) {
	v := validator.New()

	if err := v.RegisterValidation("relative", validateRelative); err != nil {
		return nil, fmt.Errorf("validate reg relative: %w", err)
	}
	if err := v.RegisterValidation("safeRelPath", validateRelPath); err != nil {
		return nil, fmt.Errorf("validate reg safeRelPath ")
	}
	if err := v.RegisterValidation("safeModelName",
		func(fl validator.FieldLevel) bool {
			return customvalidator.ValidateModelNameValue(fl.Field().String()) == nil
		}); err != nil {
		return nil, fmt.Errorf("validate reg safeRelPath ")
	}
	return v, nil
}

func validateRelative(fl validator.FieldLevel) bool {
	s := strings.TrimSpace(fl.Field().String())
	return !filepath.IsAbs(s)
}

func validateRelPath(fl validator.FieldLevel) bool {
	p := fl.Field().String()
	if p == "" {
		return false
	}

	if strings.TrimSpace(p) != p {
		return false
	}

	if len(p) > 512 {
		return false
	}

	// NUL.
	if strings.ContainsRune(p, 0) {
		return false
	}

	for _, r := range p {
		if unicode.IsControl(r) {
			return false
		}
	}

	if strings.Contains(p, `\`) {
		return false
	}

	// Unix absolute path.
	if strings.HasPrefix(p, "/") {
		return false
	}

	// Windows drive path: C:/...
	if len(p) >= 2 && p[1] == ':' {
		return false
	}

	cleaned := path.Clean(p)

	// a/../b -> b
	// a/./b  -> a/b
	// a//b   -> a/b
	// batch/ -> batch
	if cleaned != p {
		return false
	}

	parts := strings.SplitSeq(p, "/")
	for part := range parts {
		if part == "" || part == "." || part == ".." {
			return false
		}
	}

	return true
}
