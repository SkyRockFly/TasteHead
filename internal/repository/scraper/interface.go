package scraper

import (
	"context"
	"scraper/internal/pkg/apperror"
)

// Parse
type ParseReq struct {
	URL              string
	PostSelector     string
	ImageAttr        string
	NextPageSelector string
	NextPageAttr     string
}

type ParseResp struct {
	URL     []string
	NextURL string
}

// Process
type ProcessFilesReq struct {
	ModelPath    string
	DownloadPath string
}

type CSVRow struct {
	Path       string
	Dir        string
	Hash       string
	ModelScore float32
	UserScore  *float32
}

type File struct {
	Dir  string
	Hash string
	Path string
}

type DeleteImagesReq struct {
	Dir  string
	Path string
}

type TrainModelReq struct {
	CsvPath      string
	DownloadPath string
	OutputPath   string
}

type Fails struct {
	OriginURL string           `json:"origin_url"`
	Fails     []FailedDownload `json:"fails"`
}

type FailedDownload struct {
	Warn       apperror.DownloadIssue
	URL        string
	Repeatable bool
}

type Repository interface {
	ParseHTML(req ParseReq) (ParseResp, error)
	DownloadPic(downloadPath string, url string) *FailedDownload
	ProcessFiles(ctx context.Context, req ProcessFilesReq) error
	TrainModel(ctx context.Context, req TrainModelReq) error

	// possible future filemanager package

	ListUnfinishedBatches(dirPath string) ([]string, error)
	ListBatchesWithFails(dirPath string) ([]string, error)
	ListModels(dirPath string) ([]string, error)
	MoveRetryDirToParent(retryPath string) error
	FindDirs(root string) ([]string, error)
	CSVToRows(path string) ([]CSVRow, error)
	MoveDir(src, dest string) error
	CopyFile(src, dest string) error
	ComputeHashesInDir(dir string) ([]File, error)
	ComputeFileHash(file string) (string, error)
	DeleteImages(file string) error
}
