package imagerow

import (
	"context"
	"time"
)

type Row struct {
	ID         int64
	Hash       string
	BatchID    int64
	RelPath    string
	ModelScore float32
	UserScore  *float32
	DeletedAt  time.Time
}

type DeletedRow struct {
	ID             int64
	Hash           string
	BatchID        int64
	RelPath        string
	ModelScore     float32
	UserScore      *float32
	ImageDeletedAt *time.Time
	BatchDeletedAt *time.Time
}

type ImageForDelete struct {
	ID           int64
	BatchRelPath string
	ImageRelPath string
}

type Batch struct {
	ID        int64
	Name      string
	RelPath   string
	Status    string
	DeletedAt *time.Time
}

type CreateBatchReq struct {
	Name    string
	RelPath string
}

type CreateRowReq struct {
	Hash       string
	BatchID    int64
	Path       string
	ModelScore float32
	UserScore  *float32
}

type UpdateUserScore struct {
	ID    int64
	Score float32
}

type UpdateUserScoreReq struct {
	Scores []UpdateUserScore
}

type ListImagesReq struct {
	DirID     *int64
	Score     *float32
	ScoreType string
	Limit     int
	Cursor    int64
	Next      bool
}

type ListDeletedImagesReq struct {
	Limit           int
	CursorID        int64
	CursorDeletedAt *time.Time
	Next            bool
}

type ListImagesResp struct {
	CursorNext int64
	CursorPrev int64
	Images     []Row
	HasMore    bool
}

type DeletedCursor struct {
	CursorID        int64
	CursorDeletedAt *time.Time
}

type ListDeletedImagesResp struct {
	CursorPrev DeletedCursor
	CursorNext DeletedCursor
	Images     []DeletedRow
	HasMore    bool
}

type RemoveImagesReq struct {
	IDs []int64
}

type BatchStatus string

const (
	BatchStatusPending  BatchStatus = "pending"
	BatchStatusFinished BatchStatus = "finished"
)

type UpdateBatchStatusReq struct {
	ID     int64
	Status BatchStatus
}

type UpdateRowBatchReq struct {
	ID      int64
	BatchID int64
}

type UpdateBatchNameReq struct {
	BatchID   int64
	BatchName string
}

type Repository interface {
	CreateDownloadRows(ctx context.Context, rows []CreateRowReq) error
	ListImages(ctx context.Context, req ListImagesReq) (ListImagesResp, error)
	ListDeletedImages(ctx context.Context, req ListDeletedImagesReq) (ListDeletedImagesResp, error)
	UpdateUserScore(ctx context.Context, req UpdateUserScoreReq) error
	UpdateRowBatchID(ctx context.Context, req UpdateRowBatchReq) error
	RemoveImages(ctx context.Context, req RemoveImagesReq) error
	HardDeleteImages(сtx context.Context, ids []int64) error
	FindDuplicatesByHash(ctx context.Context, hashes []string) ([]string, error)
	GetRowByHash(ctx context.Context, hash string) (Row, error)
	GetRowByID(ctx context.Context, id int64) (Row, error)
	ListImagesForDelete(ctx context.Context, ids []int64) ([]ImageForDelete, error)
	RestoreImages(ctx context.Context, ids []int64) error

	CreateBatch(ctx context.Context, req CreateBatchReq) (int64, error)
	RemoveBatch(ctx context.Context, id int) error
	HardDeleteBatches(ctx context.Context, ids []int64) error
	ListBatches(ctx context.Context) ([]Batch, error)
	ListDeletedBatches(ctx context.Context) ([]Batch, error)
	GetBatch(ctx context.Context, name string) (Batch, error)
	GetBatchByID(ctx context.Context, id int64) (Batch, error)
	ListDeletedBatchesByID(ctx context.Context, ids []int64) ([]Batch, error)
	UpdateBatchStatus(ctx context.Context, req UpdateBatchStatusReq) error
	UpdateBatchName(ctx context.Context, req UpdateBatchNameReq) error
	RestoreBatches(ctx context.Context, ids []int64) error
}
