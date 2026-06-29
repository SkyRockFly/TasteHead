package imagerowsvc

import (
	"context"
	"fmt"
	"path/filepath"
	"reflect"
	"scraper/internal/pkg/apperror"
	"scraper/internal/pkg/kit"
	imagerow "scraper/internal/repository/imageRow"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
)

type Service struct {
	imageRowRepo imagerow.Repository
	validate     *validator.Validate
}

func NewService(imagerowRepo imagerow.Repository) (*Service, error) {
	validate, err := initValidator()
	if err != nil {
		return nil, fmt.Errorf("initValidator: %w", err)
	}
	return &Service{
		imageRowRepo: imagerowRepo,
		validate:     validate,
	}, nil
}

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

type Batch struct {
	ID        int64
	Name      string
	RelPath   string
	DeletedAt *time.Time
}

type ImageForDelete struct {
	ID           int64
	BatchRelPath string
	ImageRelPath string
}

type UpdateUserScore struct {
	ID    int64   `validate:"required,min=1"`
	Score float32 `validate:"userScore"`
}

type UpdateUserScoreReq struct {
	Scores []UpdateUserScore `validate:"min=1,dive"`
}

type GetImagesInfoReq struct {
	DirID     *int64
	Score     *float32
	ScoreType string `validate:"required,scoreType"`
	Limit     int    `validate:"min=0,max=100"`
	Cursor    int64  `validate:"min=0"`
}

type GetImagesInfoResp struct {
	Images  []Row
	Cursor  int64
	HasMore bool
}

type RemoveImgReq struct {
	IDs []int64 `validate:"required,min=1,dive,min=1"`
}

type CreateFromBatchRow struct {
	Hash       string   `validate:"required"`
	BatchID    int64    `validate:"min=1"`
	Path       string   `validate:"required,relative"`
	ModelScore float32  `validate:"min=0,max=1"`
	UserScore  *float32 `validate:"userScore"`
}

type CreateDownloadRowsReq struct {
	Rows []CreateFromBatchRow `validate:"required"`
}

type ListImagesReq struct {
	DirID     *int64   `validate:"IDwithPTR"`
	Score     *float32 `validate:"userScore"`
	ScoreType string   `validate:"scoreType"`
	Limit     int      `validate:"min=1,max=100"`
	Cursor    int64    `validate:"min=0"`
	Next      bool
}

type ListDeletedImagesReq struct {
	Limit           int   `validate:"min=1,max=100"`
	CursorID        int64 `validate:"min=0"`
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

type FindDuplicatesReq struct {
	Hashes []string `validate:"required,min=1,dive,required"`
}

type GetRowByHashReq struct {
	Hash string `validate:"required"`
}

type CreateBatchReq struct {
	Name    string `validate:"required"`
	RelPath string `validate:"required,relative"`
}

type RemoveBatchReq struct {
	ID int `validate:"min=1"`
}

type GetBatchReq struct {
	Name string `validate:"required"`
}

type UpdateBatchStatusReq struct {
	ID     int64  `validate:"min=1"`
	Status string `validate:"batch"`
}

type UpdateBatchNameReq struct {
	BatchID   int64  `validate:"min=1"`
	BatchName string `validate:"required"`
}

type GetRowByIDReq struct {
	ID int64 `validate:"min=1"`
}

type GetBatchByIDReq struct {
	ID int64 `validate:"min=1"`
}

type ListDeletedBatchesByID struct {
	IDs []int64 `validate:"min=1,dive,min=1"`
}

type UpdateRowBatchReq struct {
	ID      int64 `validate:"min=1"`
	BatchID int64 `validate:"min=1"`
}

type HardDeleteBatchesReq struct {
	IDs []int64 `validate:"min=1,dive,min=1"`
}

type HardDeleteImagesReq struct {
	IDs []int64 `validate:"min=1,dive,min=1"`
}

type ListImagesForDeleteReq struct {
	IDs []int64 `validate:"min=1,dive,min=1"`
}

type RestoreImagesReq struct {
	IDs []int64 `validate:"min=1,dive,min=1"`
}

type RestoreBatchesReq struct {
	IDs []int64 `validate:"min=1,dive,min=1"`
}

func (s *Service) CreateDownloadRows(ctx context.Context, req CreateDownloadRowsReq) error {
	if err := kit.ValidateStruct(s.validate, req); err != nil {
		return fmt.Errorf("validate struct: %w: %v", apperror.ErrBadRequest, err)
	}

	repoRows := make([]imagerow.CreateRowReq, 0, len(req.Rows))
	var repoRow imagerow.CreateRowReq
	for _, row := range req.Rows {
		repoRow = remapToRepoCreateRows(row)
		repoRows = append(repoRows, repoRow)
	}

	if err := s.imageRowRepo.CreateDownloadRows(ctx, repoRows); err != nil {
		return fmt.Errorf("CreateDownloadRows: %w", err)
	}

	return nil
}

func (s *Service) ListImages(ctx context.Context, req ListImagesReq) (ListImagesResp, error) {
	if err := kit.ValidateStruct(s.validate, req); err != nil {
		return ListImagesResp{}, fmt.Errorf("validate struct: %w: %v", apperror.ErrBadRequest, err)
	}

	repoReq := imagerow.ListImagesReq{
		DirID:     req.DirID,
		Score:     req.Score,
		ScoreType: req.ScoreType,
		Limit:     req.Limit,
		Cursor:    req.Cursor,
		Next:      req.Next,
	}

	repoResp, err := s.imageRowRepo.ListImages(ctx, repoReq)
	if err != nil {
		return ListImagesResp{}, fmt.Errorf("GetByScore: %w", err)
	}

	svcRows := make([]Row, 0, len(repoResp.Images))
	var svcRow Row
	for _, row := range repoResp.Images {
		svcRow = remapRepoToSVC(row)
		svcRows = append(svcRows, svcRow)
	}

	resp := ListImagesResp{
		CursorNext: repoResp.CursorNext,
		CursorPrev: repoResp.CursorPrev,
		Images:     svcRows,
		HasMore:    repoResp.HasMore,
	}

	return resp, nil
}

func (s *Service) ListDeletedImages(ctx context.Context, req ListDeletedImagesReq) (ListDeletedImagesResp, error) {
	if err := kit.ValidateStruct(s.validate, req); err != nil {
		return ListDeletedImagesResp{}, fmt.Errorf("validate struct: %w: %v", apperror.ErrBadRequest, err)
	}

	repoReq := imagerow.ListDeletedImagesReq{
		Limit:           req.Limit,
		CursorID:        req.CursorID,
		CursorDeletedAt: req.CursorDeletedAt,
		Next:            req.Next,
	}

	repoResp, err := s.imageRowRepo.ListDeletedImages(ctx, repoReq)
	if err != nil {
		return ListDeletedImagesResp{}, fmt.Errorf("GetByScore: %w", err)
	}

	svcRows := make([]DeletedRow, 0, len(repoResp.Images))
	var svcRow DeletedRow
	for _, row := range repoResp.Images {
		svcRow = remapDeletedRowToSVC(row)
		svcRows = append(svcRows, svcRow)
	}

	resp := ListDeletedImagesResp{
		CursorPrev: remapDeletedCursorToSVC(repoResp.CursorPrev),
		CursorNext: remapDeletedCursorToSVC(repoResp.CursorNext),
		Images:     svcRows,
		HasMore:    repoResp.HasMore,
	}

	return resp, nil
}

func (s *Service) ListImagesForDelete(ctx context.Context, req ListImagesForDeleteReq) ([]ImageForDelete, error) {
	if err := kit.ValidateStruct(s.validate, req); err != nil {
		return nil, fmt.Errorf("validate struct: %w: %v", apperror.ErrBadRequest, err)
	}

	repoResp, err := s.imageRowRepo.ListImagesForDelete(ctx, req.IDs)
	if err != nil {
		return nil, fmt.Errorf("ListImagesForDelete:%w", err)
	}

	resp := remapImageForDeleteToSVC(repoResp)
	return resp, nil
}

func (s *Service) GetRowByID(ctx context.Context, req GetRowByIDReq) (Row, error) {
	if err := kit.ValidateStruct(s.validate, req); err != nil {
		return Row{}, fmt.Errorf("validate struct: %w: %v", apperror.ErrBadRequest, err)
	}

	repoRow, err := s.imageRowRepo.GetRowByID(ctx, req.ID)
	if err != nil {
		return Row{}, fmt.Errorf("GetRowByID: %w", err)
	}

	svcRow := remapRepoToSVC(repoRow)

	return svcRow, nil
}

func (s *Service) UpdateUserScore(ctx context.Context, req UpdateUserScoreReq) error {
	if err := kit.ValidateStruct(s.validate, req); err != nil {
		return fmt.Errorf("validate struct: %w: %v", apperror.ErrBadRequest, err)
	}
	repoScores := RemapSVCScoresToRepo(req.Scores)

	svcReq := imagerow.UpdateUserScoreReq{
		Scores: repoScores,
	}
	if err := s.imageRowRepo.UpdateUserScore(ctx, svcReq); err != nil {
		return fmt.Errorf("updateUserScore: %w", err)
	}

	return nil
}

func (s *Service) UpdateRowBatchID(ctx context.Context, req UpdateRowBatchReq) error {
	if err := kit.ValidateStruct(s.validate, req); err != nil {
		return fmt.Errorf("validate struct: %w: %v", apperror.ErrBadRequest, err)
	}

	svcReq := imagerow.UpdateRowBatchReq{
		ID:      req.ID,
		BatchID: req.BatchID,
	}

	if err := s.imageRowRepo.UpdateRowBatchID(ctx, svcReq); err != nil {
		return fmt.Errorf("updateRowBatchID: %w", err)
	}

	return nil
}

func (s *Service) RemoveImages(ctx context.Context, req RemoveImgReq) error {
	if err := kit.ValidateStruct(s.validate, req); err != nil {
		return fmt.Errorf("validate struct: %w: %v", apperror.ErrBadRequest, err)
	}

	svcReq := imagerow.RemoveImagesReq{
		IDs: req.IDs,
	}
	if err := s.imageRowRepo.RemoveImages(ctx, svcReq); err != nil {
		return fmt.Errorf("remove imgs: %w", err)
	}

	return nil
}

func (s *Service) RestoreImages(ctx context.Context, req RestoreImagesReq) error {
	if err := kit.ValidateStruct(s.validate, req); err != nil {
		return fmt.Errorf("validate struct: %w: %v", apperror.ErrBadRequest, err)
	}

	if err := s.imageRowRepo.RestoreImages(ctx, req.IDs); err != nil {
		return fmt.Errorf("RestoreImages: %w", err)
	}

	return nil
}

func (s *Service) HardDeleteImages(ctx context.Context, req HardDeleteImagesReq) error {
	if err := kit.ValidateStruct(s.validate, req); err != nil {
		return fmt.Errorf("validate struct: %w: %v", apperror.ErrBadRequest, err)
	}

	if err := s.imageRowRepo.HardDeleteImages(ctx, req.IDs); err != nil {
		return fmt.Errorf("HardDeleteImages: %w", err)
	}

	return nil
}

func (s *Service) FindDuplicatesByHash(ctx context.Context, req FindDuplicatesReq) ([]string, error) {
	if err := kit.ValidateStruct(s.validate, req); err != nil {
		return nil, fmt.Errorf("validate struct: %w: %v", apperror.ErrBadRequest, err)
	}

	hashes, err := s.imageRowRepo.FindDuplicatesByHash(ctx, req.Hashes)
	if err != nil {
		return nil, fmt.Errorf("FindDuplicatesByHash: %w", err)
	}

	return hashes, nil
}

func (s *Service) GetRowByHash(ctx context.Context, req GetRowByHashReq) (Row, error) {
	if err := kit.ValidateStruct(s.validate, req); err != nil {
		return Row{}, fmt.Errorf("validate struct: %w: %v", apperror.ErrBadRequest, err)
	}

	repoRow, err := s.imageRowRepo.GetRowByHash(ctx, req.Hash)
	if err != nil {
		return Row{}, fmt.Errorf("GetRowByHash: %w", err)
	}

	row := remapRepoToSVC(repoRow)
	return row, nil
}

func (s *Service) CreateBatch(ctx context.Context, req CreateBatchReq) (int64, error) {
	if err := kit.ValidateStruct(s.validate, req); err != nil {
		return 0, fmt.Errorf("validate struct: %w: %v", apperror.ErrBadRequest, err)
	}

	repoReq := imagerow.CreateBatchReq{
		Name:    req.Name,
		RelPath: req.RelPath,
	}

	id, err := s.imageRowRepo.CreateBatch(ctx, repoReq)
	if err != nil {
		return 0, fmt.Errorf("CreateBatch: %w", err)
	}

	updateBatchReq := UpdateBatchStatusReq{
		ID:     id,
		Status: string(imagerow.BatchStatusFinished),
	}

	if err := s.UpdateBatchStatus(ctx, updateBatchReq); err != nil {
		return 0, fmt.Errorf("UpdateBatchStatus:%w", err)
	}

	return id, nil
}

func (s *Service) RemoveBatch(ctx context.Context, req RemoveBatchReq) error {
	if err := kit.ValidateStruct(s.validate, req); err != nil {
		return fmt.Errorf("validate struct: %w: %v", apperror.ErrBadRequest, err)
	}

	if err := s.imageRowRepo.RemoveBatch(ctx, req.ID); err != nil {
		return fmt.Errorf("RemoveBatch: %w", err)
	}

	return nil
}

func (s *Service) RestoreBatches(ctx context.Context, req RestoreBatchesReq) error {
	if err := kit.ValidateStruct(s.validate, req); err != nil {
		return fmt.Errorf("validate struct: %w: %v", apperror.ErrBadRequest, err)
	}

	if err := s.imageRowRepo.RestoreBatches(ctx, req.IDs); err != nil {
		return fmt.Errorf("RestoreBatches: %w", err)
	}

	return nil
}

func (s *Service) HardDeleteBatches(ctx context.Context, req HardDeleteBatchesReq) error {
	if err := kit.ValidateStruct(s.validate, req); err != nil {
		return fmt.Errorf("validate struct: %w: %v", apperror.ErrBadRequest, err)
	}

	if err := s.imageRowRepo.HardDeleteBatches(ctx, req.IDs); err != nil {
		return fmt.Errorf("RemoveBatch: %w", err)
	}

	return nil
}

func (s *Service) ListBatches(ctx context.Context) ([]Batch, error) {
	dirs, err := s.imageRowRepo.ListBatches(ctx)
	if err != nil {
		return nil, fmt.Errorf("ListBatches: %w", err)
	}

	svcDirs := make([]Batch, 0, len(dirs))
	for _, dir := range dirs {
		svcDir := RemapBatchToSVC(dir)
		svcDirs = append(svcDirs, svcDir)
	}

	return svcDirs, nil
}

func (s *Service) ListDeletedBatches(ctx context.Context) ([]Batch, error) {
	batches, err := s.imageRowRepo.ListDeletedBatches(ctx)
	if err != nil {
		return nil, fmt.Errorf("ListDeletedBatches: %w", err)
	}

	svcBatches := make([]Batch, 0, len(batches))
	for _, dir := range batches {
		svcDir := RemapBatchToSVC(dir)
		svcBatches = append(svcBatches, svcDir)
	}

	return svcBatches, nil
}

func (s *Service) GetBatch(ctx context.Context, req GetBatchReq) (Batch, error) {
	if err := kit.ValidateStruct(s.validate, req); err != nil {
		return Batch{}, fmt.Errorf("validate struct: %w: %v", apperror.ErrBadRequest, err)
	}

	repoBatch, err := s.imageRowRepo.GetBatch(ctx, req.Name)
	if err != nil {
		return Batch{}, fmt.Errorf("GetBatch: %w", err)
	}

	batch := RemapBatchToSVC(repoBatch)

	return batch, nil
}

func (s *Service) GetBatchByID(ctx context.Context, req GetBatchByIDReq) (Batch, error) {
	if err := kit.ValidateStruct(s.validate, req); err != nil {
		return Batch{}, fmt.Errorf("validate struct: %w: %v", apperror.ErrBadRequest, err)
	}

	repoBatch, err := s.imageRowRepo.GetBatchByID(ctx, req.ID)
	if err != nil {
		return Batch{}, fmt.Errorf("GetBatchByID: %w", err)
	}

	batch := RemapBatchToSVC(repoBatch)

	return batch, nil
}

func (s *Service) ListDeletedBatchesByID(ctx context.Context, req ListDeletedBatchesByID) ([]Batch, error) {
	if err := kit.ValidateStruct(s.validate, req); err != nil {
		return nil, fmt.Errorf("validate struct: %w: %v", apperror.ErrBadRequest, err)
	}

	repoBatch, err := s.imageRowRepo.ListDeletedBatchesByID(ctx, req.IDs)
	if err != nil {
		return nil, fmt.Errorf("ListDeletedBatchesByID: %w", err)
	}

	batches := make([]Batch, 0, len(repoBatch))
	var batch Batch
	for _, repo := range repoBatch {
		batch = RemapBatchToSVC(repo)
		batches = append(batches, batch)
	}

	return batches, nil
}

func (s *Service) UpdateBatchStatus(ctx context.Context, req UpdateBatchStatusReq) error {
	if err := kit.ValidateStruct(s.validate, req); err != nil {
		return fmt.Errorf("validate struct: %w: %v", apperror.ErrBadRequest, err)
	}

	var batchStatus imagerow.BatchStatus
	switch req.Status {
	case string(imagerow.BatchStatusFinished):
		batchStatus = imagerow.BatchStatusFinished
	case string(imagerow.BatchStatusPending):
		batchStatus = imagerow.BatchStatusPending
	default:
		return fmt.Errorf("unknown batch status: %w", apperror.ErrBadRequest)
	}

	repoReq := imagerow.UpdateBatchStatusReq{
		ID:     req.ID,
		Status: batchStatus,
	}
	if err := s.imageRowRepo.UpdateBatchStatus(ctx, repoReq); err != nil {
		return fmt.Errorf("UpdateBatchStatus: %w", err)
	}

	return nil
}

func (s *Service) UpdateBatchName(ctx context.Context, req UpdateBatchNameReq) error {
	if err := kit.ValidateStruct(s.validate, req); err != nil {
		return fmt.Errorf("validate struct: %w: %v", apperror.ErrBadRequest, err)
	}

	repoReq := imagerow.UpdateBatchNameReq{
		BatchID:   req.BatchID,
		BatchName: req.BatchName,
	}
	if err := s.imageRowRepo.UpdateBatchName(ctx, repoReq); err != nil {
		return fmt.Errorf("UpdateBatchStatus: %w", err)
	}

	return nil
}

func remapImageForDeleteToSVC(repo []imagerow.ImageForDelete) []ImageForDelete {
	svcImages := make([]ImageForDelete, 0, len(repo))
	for _, row := range repo {
		svcImage := ImageForDelete{
			ID:           row.ID,
			BatchRelPath: row.BatchRelPath,
			ImageRelPath: row.ImageRelPath,
		}
		svcImages = append(svcImages, svcImage)
	}
	return svcImages
}

func remapRepoToSVC(repo imagerow.Row) Row {
	return Row{
		ID:         repo.ID,
		Hash:       repo.Hash,
		BatchID:    repo.BatchID,
		RelPath:    repo.RelPath,
		ModelScore: repo.ModelScore,
		UserScore:  repo.UserScore,
		DeletedAt:  repo.DeletedAt,
	}
}

func remapDeletedRowToSVC(repo imagerow.DeletedRow) DeletedRow {
	return DeletedRow{
		ID:             repo.ID,
		Hash:           repo.Hash,
		BatchID:        repo.BatchID,
		RelPath:        repo.RelPath,
		ModelScore:     repo.ModelScore,
		UserScore:      repo.UserScore,
		ImageDeletedAt: repo.ImageDeletedAt,
		BatchDeletedAt: repo.BatchDeletedAt,
	}
}

func remapDeletedCursorToSVC(repo imagerow.DeletedCursor) DeletedCursor {
	return DeletedCursor{
		CursorID:        repo.CursorID,
		CursorDeletedAt: repo.CursorDeletedAt,
	}
}

func remapToRepoCreateRows(svc CreateFromBatchRow) imagerow.CreateRowReq {
	return imagerow.CreateRowReq{
		Hash:       svc.Hash,
		BatchID:    svc.BatchID,
		Path:       svc.Path,
		ModelScore: svc.ModelScore,
		UserScore:  svc.UserScore,
	}
}

func RemapSVCScoresToRepo(svcScores []UpdateUserScore) []imagerow.UpdateUserScore {
	repoScores := make([]imagerow.UpdateUserScore, 0, len(svcScores))
	var repoScore imagerow.UpdateUserScore
	for _, svcScore := range svcScores {
		repoScore.ID = svcScore.ID
		repoScore.Score = svcScore.Score
		repoScores = append(repoScores, repoScore)
	}

	return repoScores
}

func RemapBatchToSVC(batch imagerow.Batch) Batch {
	return Batch{
		ID:        batch.ID,
		Name:      batch.Name,
		RelPath:   batch.RelPath,
		DeletedAt: batch.DeletedAt,
	}
}

func initValidator() (*validator.Validate, error) {
	v := validator.New()

	if err := v.RegisterValidation("relative", validateRelative); err != nil {
		return nil, fmt.Errorf("validate relative: %w", err)
	}

	if err := v.RegisterValidation("userScore", validateUserScore, true); err != nil {
		return nil, fmt.Errorf("validate userScore: %w", err)
	}

	if err := v.RegisterValidation("IDwithPTR", validateIDorNull, true); err != nil {
		return nil, fmt.Errorf("validate IDwithPTR: %w", err)
	}

	if err := v.RegisterValidation("scoreType", validateScoreType); err != nil {
		return nil, fmt.Errorf("validate scoreType: %w", err)
	}

	if err := v.RegisterValidation("batch", validateBatch); err != nil {
		return nil, fmt.Errorf("validate batch: %w", err)
	}

	v.RegisterStructValidation(validateListDeletedImagesReq, ListDeletedImagesReq{})
	v.RegisterStructValidation(validateListImagesReq, ListImagesReq{})

	return v, nil
}

func validateRelative(fl validator.FieldLevel) bool {
	s := strings.TrimSpace(fl.Field().String())
	return !filepath.IsAbs(s)
}

func validateUserScore(fl validator.FieldLevel) bool {
	field := fl.Field()

	var score float64
	switch field.Kind() {
	case reflect.Pointer:
		if field.IsNil() {
			return true
		}
	case reflect.Float32, reflect.Float64:
		score = field.Float()
	case reflect.String:
		s := strings.TrimSpace(field.String())
		v, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return false
		}
		score = v
	default:
		return false
	}

	switch score {
	case 0, 0.25, 0.5, 0.75, 1:
		return true
	default:
		return false
	}
}

func validateIDorNull(fl validator.FieldLevel) bool {
	field := fl.Field()

	var id int64
	switch field.Kind() {
	case reflect.Pointer:
		if field.IsNil() {
			return true
		}
	case reflect.Int, reflect.Int64:
		id = field.Int()
	case reflect.String:
		s := strings.TrimSpace(field.String())
		v, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return false
		}
		id = v
	default:
		return false
	}

	return id >= 1
}

func validateScoreType(fl validator.FieldLevel) bool {
	s := strings.ToLower(strings.TrimSpace(fl.Field().String()))
	return s == "user" || s == "model"
}

func validateBatch(fl validator.FieldLevel) bool {
	s := strings.ToLower(strings.TrimSpace(fl.Field().String()))
	return s == "pending" || s == "finished"
}

func validateListDeletedImagesReq(sl validator.StructLevel) {
	req := sl.Current().Interface().(ListDeletedImagesReq)

	hasDate := req.CursorDeletedAt != nil
	hasId := req.CursorID > 0

	if hasDate != hasId {
		sl.ReportError(req.CursorID, "CursorID", "cursor_id", "cursorpair", "")
		sl.ReportError(req.CursorDeletedAt, "CursorDeletedAt", "cursor_deleted_at", "cursorpair", "")
	}

	if !req.Next && !hasDate && !hasId {
		sl.ReportError(req.Next, "Next", "next", "cursorpair", "")
	}
}

func validateListImagesReq(sl validator.StructLevel) {
	req := sl.Current().Interface().(ListImagesReq)

	hasId := req.Cursor > 0

	if !req.Next && !hasId {
		sl.ReportError(req.Next, "Next", "next", "cursorpair", "")
	}
}
