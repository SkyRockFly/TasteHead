package training

import (
	"context"
	"time"
)

type Row struct {
	ID        int64
	ImageID   int64
	TagID     int
	DeletedAt time.Time
}

type CreateRowReq struct {
	ImageIDs []int64
	TagID    int
}

type CreateRowResp struct {
	NonUniqueIds []int64
}

type ListRowsByReq struct {
	TagID     *int
	UserScore *float32
	Limit     int
	Cursor    int64
	Next      bool
}

type ListByReqRow struct {
	ID         int64
	Hash       string
	BatchID    int
	RelPath    string
	TagName    string
	ModelScore float32
	UserScore  *float32
}

type ListByTagRow struct {
	ID         int64
	Hash       string
	BatchPath  string
	RelPath    string
	TagName    string
	ModelScore float32
	UserScore  *float32
}

type ListRowsByResp struct {
	Rows       []ListByReqRow
	CursorNext int64
	CursorPrev int64
	HasMore    bool
}

type UpdateRowTagReq struct {
	IDs   []int64
	TagID int
}

type Tag struct {
	ID        int
	Name      string
	Desc      string
	DeletedAt time.Time
}

type CreateTagReq struct {
	Name string
	Desc string
}

type UpdateTagNameReq struct {
	ID         int
	UpdateName string
}

type ScoreCompositionItem struct {
	Score string
	Count int64
}

type ImageIDstoTags map[int64][]string

type Repository interface {
	CreateRows(ctx context.Context, rows CreateRowReq) (CreateRowResp, error)
	RemoveRows(ctx context.Context, id []int64) error
	ListRowsByReq(ctx context.Context, req ListRowsByReq) (ListRowsByResp, error)
	UpdateRowTag(ctx context.Context, req UpdateRowTagReq) error
	ListRowsByTag(ctx context.Context, tag int) ([]ListByTagRow, error)
	ListRowsScoresComposition(ctx context.Context, tagID int64) ([]ScoreCompositionItem, error)

	CreateTag(ctx context.Context, req CreateTagReq) error
	RemoveTag(ctx context.Context, id int) error
	UpdateTagName(ctx context.Context, req UpdateTagNameReq) error
	ListTags(ctx context.Context) ([]Tag, error)
	ListTagsByImageIDs(ctx context.Context, ids []int64) (ImageIDstoTags, error)
}
