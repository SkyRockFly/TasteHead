package imgrowpg

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"scraper/internal/pkg/apperror"
	imagerow "scraper/internal/repository/imageRow"
	"slices"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	sqlCreateRow = `
	WITH hash_row AS (
	INSERT INTO image_hash (hash)
	VALUES ($1)
	ON CONFLICT (hash) DO NOTHING
	RETURNING id
	)
INSERT INTO download (hash_id,batch_id,rel_path,model_score,user_score)
SELECT id,$2,$3,$4,$5
FROM hash_row
RETURNING ID;`
	sqlUpdateScore = `UPDATE download SET user_score = $1
WHERE ID = $2 AND deleted_at IS NULL`
	sqlUpdateRowBatchID = `UPDATE download SET batch_id = $1
WHERE ID = $2 AND deleted_at IS NULL`
	sqlRemoveImages = `UPDATE download d SET deleted_at = NOW() AT TIME ZONE 'utc'
FROM batch as b WHERE b.id = d.batch_id AND
d.id = ANY($1::bigint[]) AND d.deleted_at IS NULL AND b.deleted_at IS NULL`
	sqlRestoreImages = `UPDATE download d SET deleted_at = NULL
FROM batch as b WHERE b.id = d.batch_id AND
d.id = ANY($1::bigint[]) AND d.deleted_at IS NOT NULL AND b.deleted_at IS NULL`
	sqlHardDeleteRows = `DELETE FROM download WHERE id = ANY($1::bigint[]) AND deleted_at IS NOT NULL`
	sqlFindHashes     = `SELECT ih.hash FROM image_hash as ih
JOIN download as d on ih.id = d.hash_id
WHERE ih.hash = ANY($1::TEXT[])`
	sqlGetRowByHash = `SELECT d.id,ih.hash,d.batch_id,d.rel_path,d.model_score,d.user_score FROM download as d
JOIN batch as b on d.batch_id = b.id
JOIN image_hash ih on ih.id = d.hash_id
WHERE ih.hash = $1 AND d.deleted_at IS NULL AND b.status = 'finished'`
	sqlGetRowByID = `SELECT d.id,ih.hash,d.batch_id,d.rel_path,d.model_score,d.user_score FROM download as d
JOIN batch as b on d.batch_id = b.id
JOIN image_hash ih on ih.id = d.hash_id
WHERE d.id = $1 AND d.deleted_at IS NULL AND b.status = 'finished'`
	sqlListRowsForDelete = `SELECT d.id, d.rel_path, b.rel_path FROM download as d
JOIN batch b ON b.id = d.batch_id
WHERE d.id = ANY($1::bigint[]) AND d.deleted_at IS NOT NULL`

	sqlCreateBatch = `INSERT INTO batch (name,rel_path) VALUES ($1,$2) RETURNING id`
	sqlRemoveBatch = `UPDATE batch SET deleted_at = NOW() AT TIME ZONE 'utc'
WHERE id = $1 AND deleted_at IS NULL`
	sqlRestoreBatches = `UPDATE batch SET deleted_at = NULL
WHERE id = ANY($1::bigint[]) AND deleted_at IS NOT NULL`
	sqlHardDeleteBatches = `DELETE FROM batch WHERE id = ANY($1::bigint[]) AND deleted_at IS NOT NULL`
	sqlListBatches       = `SELECT id,name,rel_path FROM batch
WHERE deleted_at IS NULL AND status = 'finished'`
	sqlListDeletedBatches = `SELECT id,name,rel_path,deleted_at FROM batch
WHERE deleted_at IS NOT NULL AND status = 'finished'`
	sqlGetBatch = `SELECT id,name,rel_path FROM batch
WHERE name = $1 AND deleted_at IS NULL AND status = 'finished'`
	sqlGetBatchByID = `SELECT id,name,rel_path FROM batch
WHERE id = $1 AND deleted_at IS NULL AND status = 'finished'`
	sqlListDeletedBatchesBYID = `SELECT id,name,rel_path FROM batch
WHERE id = ANY($1::bigint[]) AND deleted_at IS NOT NULL AND status = 'finished'`
	sqlUpdateBatchStatus = `UPDATE batch SET status = $1
WHERE ID = $2 AND deleted_at IS NULL`
	sqlUpdateBatchName = `UPDATE batch SET name = $1
WHERE ID = $2 AND deleted_at IS NULL`
)

type Repository struct {
	postgresDB *pgxpool.Pool
}

func NewRepository(postgresDB *pgxpool.Pool) *Repository {
	return &Repository{postgresDB: postgresDB}
}

type RowWithoutUserScore struct {
	id         int64
	hash       string
	batchID    int64
	relPath    string
	modelScore float32
	userScore  sql.NullFloat64
}

type DeletedRowWithoutUserScore struct {
	id             int64
	hash           string
	batchID        int64
	relPath        string
	modelScore     float32
	userScore      sql.NullFloat64
	imageDeletedAt sql.NullTime
	batchDeletedAt sql.NullTime
}

func (r *Repository) CreateDownloadRows(ctx context.Context, rows []imagerow.CreateRowReq) error {
	batch := &pgx.Batch{}
	for _, row := range rows {
		batch.Queue(sqlCreateRow,
			row.Hash,
			row.BatchID,
			row.Path,
			row.ModelScore,
			row.UserScore)
	}
	br := r.postgresDB.SendBatch(ctx, batch)
	defer br.Close()

	for range rows {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("insert row: %w", err)
		}
	}

	return nil
}

// model_score is expected to always be present.
// Therefore GetByScore uses the following semantics:
// - model + nil score -> all images
// - user  + nil score -> images with user_score IS NULL
// - model/user + score != nil -> exact score bucket
func (r *Repository) ListImages(ctx context.Context, req imagerow.ListImagesReq) (imagerow.ListImagesResp, error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	q := psql.
		Select("d.id, ih.hash, d.batch_id, d.rel_path, d.model_score, d.user_score").
		From("download AS d").
		Join("batch as b ON b.id = d.batch_id").
		Join("image_hash as ih ON ih.id = d.hash_id").
		Where("d.deleted_at IS NULL AND b.deleted_at IS NULL")

	if req.DirID != nil {
		q = q.Where("batch_id = ?", *req.DirID)
	}
	var scoreCol string
	switch req.ScoreType {
	case "user":
		scoreCol = "d.user_score"
	case "model":
		scoreCol = "d.model_score"
	default:
		return imagerow.ListImagesResp{}, fmt.Errorf("invalid score type: %q", req.ScoreType)
	}

	if req.Score != nil {
		q = q.Where(scoreCol+" = ?", *req.Score)
	}
	if req.ScoreType == "user" && req.Score == nil {
		q = q.Where("d.user_score IS NULL")
	}
	limitPlusOne := req.Limit + 1
	if req.Next {
		q = q.Where("d.id > ?", req.Cursor)
		q = q.OrderBy("d.id ASC").Limit(uint64(limitPlusOne))
	} else {
		q = q.Where("d.id < ?", req.Cursor)
		q = q.OrderBy("d.id DESC").Limit(uint64(limitPlusOne))
	}

	sqlStr, args, err := q.ToSql()
	if err != nil {
		return imagerow.ListImagesResp{}, fmt.Errorf("build sql: %w", err)
	}

	rows, err := r.postgresDB.Query(ctx, sqlStr, args...)
	if err != nil {
		return imagerow.ListImagesResp{}, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	var imgs []imagerow.Row
	var img imagerow.Row
	for rows.Next() {
		var row RowWithoutUserScore
		if err := rows.Scan(
			&row.id,
			&row.hash,
			&row.batchID,
			&row.relPath,
			&row.modelScore,
			&row.userScore,
		); err != nil {
			return imagerow.ListImagesResp{}, fmt.Errorf("next: %w", err)
		}

		var userScorePtr *float32
		if row.userScore.Valid {
			v := float32(row.userScore.Float64)
			userScorePtr = &v
		}
		img = imagerow.Row{
			ID:         row.id,
			Hash:       row.hash,
			BatchID:    row.batchID,
			RelPath:    row.relPath,
			ModelScore: row.modelScore,
			UserScore:  userScorePtr,
		}
		imgs = append(imgs, img)
	}
	if err := rows.Err(); err != nil {
		return imagerow.ListImagesResp{}, fmt.Errorf("rows: %w", err)
	}

	n := len(imgs)
	if n == 0 {
		return imagerow.ListImagesResp{}, fmt.Errorf("imgs: %w", apperror.ErrNotFound)
	}

	hasMore := true
	if n < limitPlusOne {
		hasMore = false
		req.Limit = n
	}
	imgs = imgs[:req.Limit]
	if !req.Next {
		slices.Reverse(imgs)
	}
	top := imgs[len(imgs)-1]
	bottom := imgs[0]

	resp := imagerow.ListImagesResp{
		Images:     imgs,
		CursorNext: top.ID,
		CursorPrev: bottom.ID,
		HasMore:    hasMore,
	}

	return resp, nil
}

func (r *Repository) ListDeletedImages(ctx context.Context, req imagerow.ListDeletedImagesReq) (imagerow.ListDeletedImagesResp, error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	q := psql.
		Select(`d.id, ih.hash, d.batch_id, d.rel_path, d.model_score, d.user_score,d.deleted_at,b.deleted_at`).
		From("download AS d").
		Join("batch as b ON b.id = d.batch_id").
		Join("image_hash as ih ON ih.id = d.hash_id").
		Where("(d.deleted_at IS NOT NULL OR b.deleted_at IS NOT NULL)")

	if req.Next {
		q = q.Where(`(?::timestamp IS NULL OR 
(COALESCE(d.deleted_at,b.deleted_at),d.id) < (?::timestamp,?::bigint))`,
			req.CursorDeletedAt, req.CursorDeletedAt, req.CursorID).
			OrderBy(`COALESCE(d.deleted_at,b.deleted_at) DESC, d.id DESC`)
	} else {
		q = q.Where(`(?::timestamp IS NULL OR 
(COALESCE(d.deleted_at,b.deleted_at),d.id) > (?::timestamp,?::bigint))`,
			req.CursorDeletedAt, req.CursorDeletedAt, req.CursorID).
			OrderBy(`COALESCE(d.deleted_at,b.deleted_at) ASC, d.id ASC`)
	}

	limitPlusOne := req.Limit + 1
	q = q.Limit(uint64(limitPlusOne))

	sqlStr, args, err := q.ToSql()
	if err != nil {
		return imagerow.ListDeletedImagesResp{}, fmt.Errorf("build sql: %w", err)
	}

	rows, err := r.postgresDB.Query(ctx, sqlStr, args...)
	if err != nil {
		return imagerow.ListDeletedImagesResp{}, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	var imgs []imagerow.DeletedRow
	var img imagerow.DeletedRow
	for rows.Next() {
		var row DeletedRowWithoutUserScore
		if err := rows.Scan(
			&row.id,
			&row.hash,
			&row.batchID,
			&row.relPath,
			&row.modelScore,
			&row.userScore,
			&row.imageDeletedAt,
			&row.batchDeletedAt,
		); err != nil {
			return imagerow.ListDeletedImagesResp{}, fmt.Errorf("next: %w", err)
		}

		var userScorePtr *float32
		var batchDeletedAt *time.Time
		var imageDeletedAt *time.Time
		if row.userScore.Valid {
			v := float32(row.userScore.Float64)
			userScorePtr = &v
		}
		if row.batchDeletedAt.Valid {
			deleted := row.batchDeletedAt.Time
			batchDeletedAt = &deleted
		}
		if row.imageDeletedAt.Valid {
			deleted := row.imageDeletedAt.Time
			imageDeletedAt = &deleted
		}
		img = imagerow.DeletedRow{
			ID:             row.id,
			Hash:           row.hash,
			BatchID:        row.batchID,
			RelPath:        row.relPath,
			ModelScore:     row.modelScore,
			UserScore:      userScorePtr,
			BatchDeletedAt: batchDeletedAt,
			ImageDeletedAt: imageDeletedAt,
		}
		imgs = append(imgs, img)
	}
	if err := rows.Err(); err != nil {
		return imagerow.ListDeletedImagesResp{}, fmt.Errorf("rows: %w", err)
	}

	n := len(imgs)
	if n == 0 {
		return imagerow.ListDeletedImagesResp{}, fmt.Errorf("imgs: %w", apperror.ErrNotFound)
	}

	hasMore := true
	if n < limitPlusOne {
		hasMore = false
		req.Limit = n
	}
	imgs = imgs[:req.Limit]
	var cursorPrev, cursorNext imagerow.DeletedCursor
	if !req.Next {
		slices.Reverse(imgs)
	}
	top := imgs[0]
	bottom := imgs[len(imgs)-1]

	cursorDeletedAtPrev := effectiveDeletedAt(top)
	cursorPrev = imagerow.DeletedCursor{
		CursorID:        top.ID,
		CursorDeletedAt: &cursorDeletedAtPrev,
	}
	cursorDeletedAtNext := effectiveDeletedAt(bottom)
	cursorNext = imagerow.DeletedCursor{
		CursorID:        bottom.ID,
		CursorDeletedAt: &cursorDeletedAtNext,
	}

	resp := imagerow.ListDeletedImagesResp{
		Images:     imgs,
		CursorPrev: cursorPrev,
		CursorNext: cursorNext,
		HasMore:    hasMore,
	}

	return resp, nil
}

func (r *Repository) UpdateUserScore(ctx context.Context, req imagerow.UpdateUserScoreReq) error {
	for _, score := range req.Scores {
		tag, err := r.postgresDB.Exec(ctx, sqlUpdateScore, score.Score, score.ID)
		if err != nil {
			return fmt.Errorf("query: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return fmt.Errorf("img: %w", apperror.ErrNotFound)
		}
	}

	return nil
}

func (r *Repository) UpdateRowBatchID(ctx context.Context, req imagerow.UpdateRowBatchReq) error {
	tag, err := r.postgresDB.Exec(ctx, sqlUpdateRowBatchID, req.BatchID, req.ID)
	if err != nil {
		return fmt.Errorf("query: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf("rows: %w", apperror.ErrNotFound)
	}

	return nil
}

func (r *Repository) HardDeleteBatches(ctx context.Context, ids []int64) error {
	tag, err := r.postgresDB.Exec(ctx, sqlHardDeleteBatches, ids)
	if err != nil {
		return fmt.Errorf("query: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf("rows: %w", apperror.ErrNotFound)
	}

	return nil
}

func (r *Repository) RemoveImages(ctx context.Context, req imagerow.RemoveImagesReq) error {
	tag, err := r.postgresDB.Exec(ctx, sqlRemoveImages, req.IDs)
	if err != nil {
		return fmt.Errorf("query: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf("rows: %w", apperror.ErrNotFound)
	}

	return nil
}

func (r *Repository) RestoreImages(ctx context.Context, ids []int64) error {
	tag, err := r.postgresDB.Exec(ctx, sqlRestoreImages, ids)
	if err != nil {
		return fmt.Errorf("query: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf("rows: %w", apperror.ErrNotFound)
	}

	return nil
}

func (r *Repository) ListImagesForDelete(ctx context.Context, ids []int64) ([]imagerow.ImageForDelete, error) {
	rows, err := r.postgresDB.Query(ctx, sqlListRowsForDelete, ids)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	rowsForDelete := make([]imagerow.ImageForDelete, 0, len(ids))
	var row imagerow.ImageForDelete
	for rows.Next() {
		if err := rows.Scan(
			&row.ID,
			&row.ImageRelPath,
			&row.BatchRelPath,
		); err != nil {
			return nil, fmt.Errorf("next: %w", err)
		}
		rowsForDelete = append(rowsForDelete, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows: %w", err)
	}

	if len(rowsForDelete) == 0 {
		return nil, fmt.Errorf("batch: %w", apperror.ErrNotFound)
	}

	return rowsForDelete, nil
}

func (r *Repository) HardDeleteImages(ctx context.Context, ids []int64) error {
	tag, err := r.postgresDB.Exec(ctx, sqlHardDeleteRows, ids)
	if err != nil {
		return fmt.Errorf("query: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf("rows: %w", apperror.ErrNotFound)
	}

	return nil
}

func (r *Repository) FindDuplicatesByHash(ctx context.Context, hashes []string) ([]string, error) {
	rows, err := r.postgresDB.Query(ctx, sqlFindHashes, hashes)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	knownHashes := make(map[string]struct{}, 0)
	for _, hash := range hashes {
		knownHashes[hash] = struct{}{}
	}

	var hash string
	duplicates := make([]string, 0)
	for rows.Next() {
		if err := rows.Scan(
			&hash,
		); err != nil {
			return nil, fmt.Errorf("rows: %w", err)
		}
		_, ok := knownHashes[hash]
		if ok {
			duplicates = append(duplicates, hash)
		}
	}

	return duplicates, nil
}

func (r *Repository) GetRowByID(ctx context.Context, id int64) (imagerow.Row, error) {
	var row RowWithoutUserScore
	if err := r.postgresDB.QueryRow(ctx, sqlGetRowByID, id).Scan(
		&row.id,
		&row.hash,
		&row.batchID,
		&row.relPath,
		&row.modelScore,
		&row.userScore,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return imagerow.Row{}, fmt.Errorf("row: %w", apperror.ErrNotFound)
		}
		return imagerow.Row{}, fmt.Errorf("scan: %w", err)
	}
	var userScorePtr *float32
	if row.userScore.Valid {
		v := float32(row.userScore.Float64)
		userScorePtr = &v
	}

	imgrow := imagerow.Row{
		ID:         row.id,
		BatchID:    row.batchID,
		Hash:       row.hash,
		RelPath:    row.relPath,
		ModelScore: row.modelScore,
		UserScore:  userScorePtr,
	}

	return imgrow, nil
}

func (r *Repository) GetRowByHash(ctx context.Context, hash string) (imagerow.Row, error) {
	var row RowWithoutUserScore
	if err := r.postgresDB.QueryRow(ctx, sqlGetRowByHash, hash).Scan(
		&row.id,
		&row.hash,
		&row.batchID,
		&row.relPath,
		&row.modelScore,
		&row.userScore,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return imagerow.Row{}, fmt.Errorf("rows: %w", apperror.ErrNotFound)
		}
		return imagerow.Row{}, fmt.Errorf("scan: %w", err)
	}
	var userScorePtr *float32
	if row.userScore.Valid {
		v := float32(row.userScore.Float64)
		userScorePtr = &v
	}

	imgrow := imagerow.Row{
		ID:         row.id,
		BatchID:    row.batchID,
		Hash:       row.hash,
		RelPath:    row.relPath,
		ModelScore: row.modelScore,
		UserScore:  userScorePtr,
	}

	return imgrow, nil
}

func (r *Repository) CreateBatch(ctx context.Context, req imagerow.CreateBatchReq) (int64, error) {
	var id int64
	err := r.postgresDB.QueryRow(ctx, sqlCreateBatch, req.Name, req.RelPath).Scan(&id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			return 0, fmt.Errorf("batch:%w", apperror.ErrAlreadyExists)
		}
		return 0, fmt.Errorf("query: %w", err)
	}

	return id, nil
}

func (r *Repository) RemoveBatch(ctx context.Context, id int) error {
	tag, err := r.postgresDB.Exec(ctx, sqlRemoveBatch, id)
	if err != nil {
		return fmt.Errorf("query: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf("rows: %w", apperror.ErrNotFound)
	}

	return nil
}

func (r *Repository) RestoreBatches(ctx context.Context, ids []int64) error {
	tag, err := r.postgresDB.Exec(ctx, sqlRestoreBatches, ids)
	if err != nil {
		return fmt.Errorf("query: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf("rows: %w", apperror.ErrNotFound)
	}

	return nil
}

func (r *Repository) ListBatches(ctx context.Context) ([]imagerow.Batch, error) {
	rows, err := r.postgresDB.Query(ctx, sqlListBatches)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	var batches []imagerow.Batch
	var batch imagerow.Batch
	for rows.Next() {
		if err := rows.Scan(
			&batch.ID,
			&batch.Name,
			&batch.RelPath,
		); err != nil {
			return nil, fmt.Errorf("next: %w", err)
		}
		batches = append(batches, batch)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows: %w", err)
	}

	if len(batches) == 0 {
		return nil, fmt.Errorf("batch: %w", apperror.ErrNotFound)
	}

	return batches, nil
}

func (r *Repository) ListDeletedBatches(ctx context.Context) ([]imagerow.Batch, error) {
	rows, err := r.postgresDB.Query(ctx, sqlListDeletedBatches)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	var batches []imagerow.Batch
	var batch imagerow.Batch
	var deletedAt time.Time
	for rows.Next() {
		if err := rows.Scan(
			&batch.ID,
			&batch.Name,
			&batch.RelPath,
			&deletedAt,
		); err != nil {
			return nil, fmt.Errorf("next: %w", err)
		}
		batch.DeletedAt = &deletedAt
		batches = append(batches, batch)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows: %w", err)
	}

	if len(batches) == 0 {
		return nil, fmt.Errorf("batch: %w", apperror.ErrNotFound)
	}

	return batches, nil
}

func (r *Repository) GetBatch(ctx context.Context, name string) (imagerow.Batch, error) {
	var batch imagerow.Batch
	if err := r.postgresDB.QueryRow(ctx, sqlGetBatch, name).Scan(
		&batch.ID,
		&batch.Name,
		&batch.RelPath,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return imagerow.Batch{}, fmt.Errorf("rows: %w", apperror.ErrNotFound)
		}
		return imagerow.Batch{}, fmt.Errorf("scan: %w", err)
	}
	return batch, nil
}

func (r *Repository) GetBatchByID(ctx context.Context, id int64) (imagerow.Batch, error) {
	var batch imagerow.Batch
	if err := r.postgresDB.QueryRow(ctx, sqlGetBatchByID, id).Scan(
		&batch.ID,
		&batch.Name,
		&batch.RelPath,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return imagerow.Batch{}, fmt.Errorf("rows: %w", apperror.ErrNotFound)
		}
		return imagerow.Batch{}, fmt.Errorf("scan: %w", err)
	}
	return batch, nil
}

func (r *Repository) ListDeletedBatchesByID(ctx context.Context, ids []int64) ([]imagerow.Batch, error) {
	rows, err := r.postgresDB.Query(ctx, sqlListDeletedBatchesBYID, ids)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	var batches []imagerow.Batch
	var batch imagerow.Batch
	for rows.Next() {
		if err := rows.Scan(
			&batch.ID,
			&batch.Name,
			&batch.RelPath,
		); err != nil {
			return nil, fmt.Errorf("next: %w", err)
		}
		batches = append(batches, batch)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows: %w", err)
	}

	if len(batches) == 0 {
		return nil, fmt.Errorf("batch: %w", apperror.ErrNotFound)
	}

	return batches, nil
}

func (r *Repository) UpdateBatchStatus(ctx context.Context, req imagerow.UpdateBatchStatusReq) error {
	tag, err := r.postgresDB.Exec(ctx, sqlUpdateBatchStatus, req.Status, req.ID)
	if err != nil {
		return fmt.Errorf("query: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf("batch:%w", apperror.ErrNotFound)
	}

	return nil
}

func (r *Repository) UpdateBatchName(ctx context.Context, req imagerow.UpdateBatchNameReq) error {
	tag, err := r.postgresDB.Exec(ctx, sqlUpdateBatchName, req.BatchName, req.BatchID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			return fmt.Errorf("name:%w", apperror.ErrAlreadyExists)
		}
		return fmt.Errorf("query: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf("batch:%w", apperror.ErrNotFound)
	}

	return nil
}

func effectiveDeletedAt(row imagerow.DeletedRow) time.Time {
	if row.BatchDeletedAt != nil {
		return *row.BatchDeletedAt
	}

	return *row.ImageDeletedAt
}
