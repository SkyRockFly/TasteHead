package trainingpg

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"scraper/internal/pkg/apperror"
	"scraper/internal/repository/training"
	"slices"
	"strconv"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	sqlCreateRow = `INSERT INTO training (image_id,tag_id)
VALUES ($1,$2) ON CONFLICT (image_id,tag_id) DO NOTHING
RETURNING id`
	sqlRemoveRow = `UPDATE training
SET deleted_at = NOW() at time zone 'utc'
WHERE id = ANY($1::bigint[]) AND deleted_at IS NULL`
	sqlUpdateRowTag = `UPDATE training tr
SET tag_id = $1
WHERE tr.id = ANY($2::bigint[]) AND tr.deleted_at IS NULL
AND tr.tag_id IS DISTINCT FROM $1
AND EXISTS (
SELECT 1
FROM download d
WHERE d.id = tr.image_id
AND d.deleted_at IS NULL)
AND EXISTS (
SELECT 1
FROM tag t
WHERE t.id = $1
AND deleted_at IS NULL)`
	sqlListByTag = `SELECT 
tr.id,
ih.hash,
d.rel_path AS filepath,
d.model_score,
d.user_score,
b.rel_path AS batch_dir,
t.name
FROM training tr
JOIN download d ON tr.image_id = d.id
JOIN batch b ON d.batch_id = b.id
JOIN tag t ON tr.tag_id = t.id
JOIN image_hash ih ON ih.id = d.hash_id
WHERE tr.deleted_at IS NULL AND d.deleted_at IS NULL AND d.user_score IS NOT NULL
AND tr.tag_id = $1`

	sqlCreateTag = `INSERT INTO tag (name,description) VALUES($1,$2)`
	sqlRemoveTag = `UPDATE tag SET deleted_at = NOW() AT TIME ZONE 'utc'
WHERE ID = $1 AND deleted_at IS NULL`
	sqlUpdateTagName = `UPDATE tag SET name = $1 WHERE id = $2 AND deleted_at IS NULL`
	sqlListTags      = `SELECT id,name,description FROM tag 
WHERE deleted_at IS NULL`
	sqlListScoreCompositon = `SELECT COUNT(*) FROM training tr
JOIN download d ON tr.image_id = d.id
WHERE tr.tag_id = $1 AND d.user_score = $2`
	sqlListTagsByImageIDs = `SELECT d.id,t.name FROM tag t
JOIN training tr ON tr.tag_id = t.id
JOIN download d ON tr.image_id = d.id
WHERE d.id = ANY($1::bigint[]) AND d.deleted_at IS NULL AND tr.deleted_at IS NULL
AND t.deleted_at IS NULL
ORDER BY d.id, t.name;`
)

var scoreRange = []float64{0, 0.25, 0.50, 0.75, 1.00}

type Repository struct {
	postgresDB *pgxpool.Pool
}

func NewRepository(postgresDB *pgxpool.Pool) *Repository {
	return &Repository{postgresDB: postgresDB}
}

type ListQueryRowWithNull struct {
	id         int64
	hash       string
	batchID    int
	relPath    string
	tagName    string
	modelScore float32
	userScore  sql.NullFloat64
}

func (r *Repository) CreateRows(ctx context.Context, req training.CreateRowReq) (training.CreateRowResp, error) {
	batch := &pgx.Batch{}
	for _, id := range req.ImageIDs {
		batch.Queue(sqlCreateRow,
			id,
			req.TagID)
	}
	br := r.postgresDB.SendBatch(ctx, batch)
	defer br.Close()

	duplicates := make([]int64, 0)
	for _, imgID := range req.ImageIDs {
		var id int64
		if err := br.QueryRow().Scan(&id); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				duplicates = append(duplicates, imgID)
				continue
			}
			return training.CreateRowResp{}, fmt.Errorf("insert row, image_id: %d: %w", imgID, err)
		}
	}

	resp := training.CreateRowResp{
		NonUniqueIds: duplicates,
	}

	return resp, nil
}

func (r *Repository) ListTagsByImageIDs(ctx context.Context, ids []int64) (training.ImageIDstoTags, error) {
	rows, err := r.postgresDB.Query(ctx, sqlListTagsByImageIDs, ids)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	tags := make(training.ImageIDstoTags, 0)
	var imageID int64
	var tagName string
	for rows.Next() {
		if err := rows.Scan(
			&imageID,
			&tagName,
		); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		tags[imageID] = append(tags[imageID], tagName)
	}

	return tags, nil
}

func (r *Repository) RemoveRows(ctx context.Context, id []int64) error {
	tag, err := r.postgresDB.Exec(ctx, sqlRemoveRow, id)
	if err != nil {
		return fmt.Errorf("query: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf("row: %w", apperror.ErrNotFound)
	}

	return nil
}

func (r *Repository) ListRowsByReq(ctx context.Context, req training.ListRowsByReq) (training.ListRowsByResp, error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	q := psql.Select("tr.id",
		"ih.hash",
		"d.rel_path AS filepath",
		"d.model_score",
		"d.user_score",
		"d.batch_id AS batch_id",
		"t.name").
		From("training tr").
		Join("download d ON tr.image_id = d.id").
		Join("tag t ON tr.tag_id = t.id").
		Join("image_hash ih ON ih.id = d.hash_id").
		Where("tr.deleted_at IS NULL AND d.deleted_at IS NULL")

	if req.TagID != nil {
		q = q.Where("tr.tag_id = ?", *req.TagID)
	}

	if req.UserScore != nil {
		q = q.Where("d.user_score = ?", *req.UserScore)
	}

	limitPlusOne := req.Limit + 1
	if req.Next {
		q = q.Where("tr.id > ?", req.Cursor)
		q = q.OrderBy("tr.id ASC").Limit(uint64(limitPlusOne))
	} else {
		q = q.Where("tr.id < ?", req.Cursor)
		q = q.OrderBy("tr.id DESC").Limit(uint64(limitPlusOne))
	}

	sqlStr, args, err := q.ToSql()
	if err != nil {
		return training.ListRowsByResp{}, fmt.Errorf("build sql: %w", err)
	}

	rows, err := r.postgresDB.Query(ctx, sqlStr, args...)
	if err != nil {
		return training.ListRowsByResp{}, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	var trains []training.ListByReqRow
	var train training.ListByReqRow
	for rows.Next() {
		var row ListQueryRowWithNull
		if err := rows.Scan(
			&row.id,
			&row.hash,
			&row.relPath,
			&row.modelScore,
			&row.userScore,
			&row.batchID,
			&row.tagName,
		); err != nil {
			return training.ListRowsByResp{}, fmt.Errorf("next: %w", err)
		}

		var userScorePtr *float32
		if row.userScore.Valid {
			v := float32(row.userScore.Float64)
			userScorePtr = &v
		}
		train = training.ListByReqRow{
			ID:         row.id,
			Hash:       row.hash,
			BatchID:    row.batchID,
			RelPath:    row.relPath,
			ModelScore: row.modelScore,
			UserScore:  userScorePtr,
			TagName:    row.tagName,
		}
		trains = append(trains, train)
	}
	if err := rows.Err(); err != nil {
		return training.ListRowsByResp{}, fmt.Errorf("rows: %w", err)
	}

	n := len(trains)
	if n == 0 {
		return training.ListRowsByResp{}, fmt.Errorf("imgs: %w", apperror.ErrNotFound)
	}

	hasMore := true
	if n < limitPlusOne {
		hasMore = false
		req.Limit = n
	}
	trains = trains[:req.Limit]
	if !req.Next {
		slices.Reverse(trains)
	}
	top := trains[len(trains)-1]
	bottom := trains[0]

	resp := training.ListRowsByResp{
		Rows:       trains,
		CursorNext: top.ID,
		CursorPrev: bottom.ID,
		HasMore:    hasMore,
	}

	return resp, nil
}

func (r *Repository) ListRowsByTag(ctx context.Context, tag int) ([]training.ListByTagRow, error) {
	rows, err := r.postgresDB.Query(ctx, sqlListByTag, tag)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	trainRows := make([]training.ListByTagRow, 0)
	var trainRow training.ListByTagRow
	for rows.Next() {
		if err := rows.Scan(
			&trainRow.ID,
			&trainRow.Hash,
			&trainRow.RelPath,
			&trainRow.ModelScore,
			&trainRow.UserScore,
			&trainRow.BatchPath,
			&trainRow.TagName,
		); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		trainRows = append(trainRows, trainRow)

	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows: %w", err)
	}

	if len(trainRows) == 0 {
		return nil, fmt.Errorf("rows: %w", apperror.ErrNotFound)
	}

	return trainRows, nil
}

func (r *Repository) ListRowsScoresComposition(ctx context.Context, tagID int64) ([]training.ScoreCompositionItem, error) {
	compositions := make([]training.ScoreCompositionItem, 0, len(scoreRange))
	zeroCompositions := 0
	for _, score := range scoreRange {
		var count int64
		if err := r.postgresDB.QueryRow(ctx, sqlListScoreCompositon, tagID, score).Scan(&count); err != nil {
			return nil, fmt.Errorf("query: %w", err)
		}
		scoreInString := strconv.FormatFloat(score, 'f', 2, 64)
		scoreItem := training.ScoreCompositionItem{
			Score: scoreInString,
			Count: count,
		}
		compositions = append(compositions, scoreItem)

		if count == 0 {
			zeroCompositions++
		}

	}

	if zeroCompositions == len(scoreRange) {
		return nil, fmt.Errorf("scores: %w", apperror.ErrNotFound)
	}

	return compositions, nil
}

func (r *Repository) UpdateRowTag(ctx context.Context, req training.UpdateRowTagReq) error {
	tag, err := r.postgresDB.Exec(ctx, sqlUpdateRowTag, req.TagID, req.IDs)
	if err != nil {
		return fmt.Errorf("query: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf("row:%w", apperror.ErrNotFound)
	}

	return nil
}

func (r *Repository) CreateTag(ctx context.Context, req training.CreateTagReq) error {
	if _, err := r.postgresDB.Exec(ctx, sqlCreateTag, req.Name, req.Desc); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == "23505" {
				return fmt.Errorf("query: %w , duplicate name: %s", apperror.ErrUniqueEntity, req.Name)
			}
			return fmt.Errorf("query:%w", err)
		}
	}

	return nil
}

func (r *Repository) RemoveTag(ctx context.Context, id int) error {
	tag, err := r.postgresDB.Exec(ctx, sqlRemoveTag, id)
	if err != nil {
		return fmt.Errorf("query: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf("tag:%w", apperror.ErrNotFound)
	}

	return nil
}

func (r *Repository) UpdateTagName(ctx context.Context, req training.UpdateTagNameReq) error {
	tag, err := r.postgresDB.Exec(ctx, sqlUpdateTagName, req.UpdateName, req.ID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == "23505" {
				return fmt.Errorf("query: %w , duplicate name: %s", apperror.ErrUniqueEntity, req.UpdateName)
			}
			return fmt.Errorf("query:%w", err)
		}
		return fmt.Errorf("query: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf("row: %w", apperror.ErrNotFound)
	}

	return nil
}

func (r *Repository) ListTags(ctx context.Context) ([]training.Tag, error) {
	rows, err := r.postgresDB.Query(ctx, sqlListTags)
	if err != nil {
		return nil, fmt.Errorf("query:%w", err)
	}
	defer rows.Close()

	tags := make([]training.Tag, 0)
	var oneTag training.Tag
	for rows.Next() {
		if err := rows.Scan(
			&oneTag.ID,
			&oneTag.Name,
			&oneTag.Desc,
		); err != nil {
			return nil, fmt.Errorf("scan:%w", err)
		}
		tags = append(tags, oneTag)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows: %w", err)
	}

	return tags, nil
}
