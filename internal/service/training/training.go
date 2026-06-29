package trainingsvc

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"scraper/internal/pkg/apperror"
	"scraper/internal/pkg/kit"
	"scraper/internal/repository/training"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
)

var ListQueryRowHeaders = []string{
	"id",
	"dir",
	"path",
	"user_score",
}

type Service struct {
	trainingRepo training.Repository
	validate     *validator.Validate
	trainingPath string
}

func NewService(trainingRepo training.Repository, trainingPath string) (*Service, error) {
	validator, err := initValidator()
	if err != nil {
		return nil, fmt.Errorf("initValidator: %w", err)
	}
	return &Service{
		trainingRepo: trainingRepo,
		validate:     validator,
		trainingPath: trainingPath,
	}, nil
}

type ListQueryRow struct {
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

type Tag struct {
	ID        int
	Name      string
	Desc      string
	DeletedAt time.Time
}

type CreateRowReq struct {
	ImageIDs []int64 `validate:"required,min=1,dive,min=1"`
	TagID    int     `validate:"min=1"`
}

type CreateTrainingRowsResp struct {
	Duplicates []int64
}

type RemoveRowReq struct {
	ID int64 `validate:"required,min=1"`
}

type ListRowsByReq struct {
	TagID     *int     `validate:"IDwithPTR"`
	UserScore *float32 `validate:"userScore"`
	Limit     int      `validate:"min=1"`
	Cursor    int64    `validate:"min=0"`
	Next      bool
}

type ListRowsByResp struct {
	Rows       []ListQueryRow
	CursorNext int64
	CursorPrev int64
	HasMore    bool
}

type UpdateRowTagReq struct {
	IDs   []int64 `validate:"min=1"`
	TagID int     `validate:"min=1"`
}

type CreateTagReq struct {
	Name string `validate:"required"`
	Desc string
}

type RemoveTagReq struct {
	ID int `validate:"min=1"`
}

type UpdateTagNameReq struct {
	ID         int    `validate:"min=1"`
	UpdateName string `validate:"required"`
}

type ExportAsSVCReq struct {
	Tag int `validate:"min=1"`
}

type ScoreCompositionItem struct {
	Score string
	Count int64
}

type ImageIDstoTags map[int64][]string

type ListTagsByImageIDsReq struct {
	IDs []int64 `validate:"min=1,dive,min=1"`
}

type ListRowsScoresCompositionReq struct {
	TagID int64 `validate:"required,min=1"`
}

func (s *Service) CreateRows(ctx context.Context, req CreateRowReq) (CreateTrainingRowsResp, error) {
	if err := kit.ValidateStruct(s.validate, req); err != nil {
		return CreateTrainingRowsResp{}, fmt.Errorf("validate struct: %w: %v", apperror.ErrBadRequest, err)
	}

	repoReq := training.CreateRowReq{
		TagID:    req.TagID,
		ImageIDs: req.ImageIDs,
	}

	repoResp, err := s.trainingRepo.CreateRows(ctx, repoReq)
	if err != nil {
		return CreateTrainingRowsResp{}, fmt.Errorf("createRows: %w", err)
	}

	resp := CreateTrainingRowsResp{
		Duplicates: repoResp.NonUniqueIds,
	}

	return resp, nil
}

func (s *Service) RemoveRow(ctx context.Context, req RemoveRowReq) error {
	if err := kit.ValidateStruct(s.validate, req); err != nil {
		return fmt.Errorf("validate struct: %w: %v", apperror.ErrBadRequest, err)
	}

	if err := s.trainingRepo.RemoveRow(ctx, req.ID); err != nil {
		return fmt.Errorf("remove row: %w", err)
	}

	return nil
}

func (s *Service) ListTagsByImageIDs(ctx context.Context, req ListTagsByImageIDsReq) (ImageIDstoTags, error) {
	if err := kit.ValidateStruct(s.validate, req); err != nil {
		return nil, fmt.Errorf("validate struct: %w: %v", apperror.ErrBadRequest, err)
	}

	tags, err := s.trainingRepo.ListTagsByImageIDs(ctx, req.IDs)
	if err != nil {
		return nil, fmt.Errorf("ListTagsByImageIDs: %w", err)
	}

	resp := remapImageIDsToTagsSVC(tags)
	return resp, nil
}

func (s *Service) ListRowsByReq(ctx context.Context, req ListRowsByReq) (ListRowsByResp, error) {
	if err := kit.ValidateStruct(s.validate, req); err != nil {
		return ListRowsByResp{}, fmt.Errorf("validate struct: %w: %v", apperror.ErrBadRequest, err)
	}

	repoReq := training.ListRowsByReq{
		TagID:     req.TagID,
		UserScore: req.UserScore,
		Limit:     req.Limit,
		Cursor:    req.Cursor,
		Next:      req.Next,
	}

	repoResp, err := s.trainingRepo.ListRowsByReq(ctx, repoReq)
	if err != nil {
		return ListRowsByResp{}, fmt.Errorf("ListRowsByReq: %w", err)
	}

	svcList := make([]ListQueryRow, 0, len(repoResp.Rows))
	var svcItem ListQueryRow
	for _, item := range repoResp.Rows {
		svcItem = remapListByReqRowToSVC(item)
		svcList = append(svcList, svcItem)
	}

	svcResp := ListRowsByResp{
		Rows:       svcList,
		CursorNext: repoResp.CursorNext,
		CursorPrev: repoResp.CursorPrev,
		HasMore:    repoResp.HasMore,
	}

	return svcResp, nil
}

func (s *Service) ListRowsScoresComposition(ctx context.Context, req ListRowsScoresCompositionReq) ([]ScoreCompositionItem, error) {
	if err := kit.ValidateStruct(s.validate, req); err != nil {
		return nil, fmt.Errorf("validate struct: %w: %v", apperror.ErrBadRequest, err)
	}

	compositions, err := s.trainingRepo.ListRowsScoresComposition(ctx, req.TagID)
	if err != nil {
		return nil, fmt.Errorf("ListRowsScoresCompositions:%w", err)
	}

	resp := remapScoreCompositionItemToSVC(compositions)

	return resp, nil
}

func (s *Service) UpdateRowTag(ctx context.Context, req UpdateRowTagReq) error {
	if err := kit.ValidateStruct(s.validate, req); err != nil {
		return fmt.Errorf("validate struct: %w: %v", apperror.ErrBadRequest, err)
	}

	repoReq := training.UpdateRowTagReq{
		IDs:   req.IDs,
		TagID: req.TagID,
	}

	if err := s.trainingRepo.UpdateRowTag(ctx, repoReq); err != nil {
		return fmt.Errorf("UpdateRowTag: %w", err)
	}

	return nil
}

func (s *Service) CreateTag(ctx context.Context, req CreateTagReq) error {
	if err := kit.ValidateStruct(s.validate, req); err != nil {
		return fmt.Errorf("validate struct: %w: %v", apperror.ErrBadRequest, err)
	}

	repoReq := training.CreateTagReq{
		Name: req.Name,
		Desc: req.Desc,
	}

	if err := s.trainingRepo.CreateTag(ctx, repoReq); err != nil {
		return fmt.Errorf("CreateTag: %w", err)
	}

	return nil
}

func (s *Service) RemoveTag(ctx context.Context, req RemoveTagReq) error {
	if err := kit.ValidateStruct(s.validate, req); err != nil {
		return fmt.Errorf("validate struct: %w: %v", apperror.ErrBadRequest, err)
	}

	if err := s.trainingRepo.RemoveTag(ctx, req.ID); err != nil {
		return fmt.Errorf("RemoveTag: %w", err)
	}

	return nil
}

func (s *Service) UpdateTagName(ctx context.Context, req UpdateTagNameReq) error {
	if err := kit.ValidateStruct(s.validate, req); err != nil {
		return fmt.Errorf("validate struct: %w: %v", apperror.ErrBadRequest, err)
	}

	repoReq := training.UpdateTagNameReq{
		ID:         req.ID,
		UpdateName: req.UpdateName,
	}

	if err := s.trainingRepo.UpdateTagName(ctx, repoReq); err != nil {
		return fmt.Errorf("UpdateTagName: %w", err)
	}

	return nil
}

func (s *Service) ListTags(ctx context.Context) ([]Tag, error) {
	repoTags, err := s.trainingRepo.ListTags(ctx)
	if err != nil {
		return nil, fmt.Errorf("ListTags: %w", err)
	}

	svcTags := make([]Tag, 0, len(repoTags))
	var svcTag Tag
	for _, tag := range repoTags {
		svcTag = remapTagsToSVC(tag)
		svcTags = append(svcTags, svcTag)
	}

	return svcTags, nil
}

func (s *Service) ExportAsSVC(ctx context.Context, req ExportAsSVCReq) (string, error) {
	if err := kit.ValidateStruct(s.validate, req); err != nil {
		return "", fmt.Errorf("validate struct: %w: %v", apperror.ErrBadRequest, err)
	}

	rows, err := s.trainingRepo.ListRowsByTag(ctx, req.Tag)
	if err != nil {
		return "", fmt.Errorf("ListRowsByTag: %w", err)
	}

	svcRows := make([]ListByTagRow, 0, len(rows))
	var svcRow ListByTagRow
	for _, row := range rows {
		svcRow = remapListByTagRowToSVC(row)
		svcRows = append(svcRows, svcRow)
	}

	csvRows := listQueryToCSV(svcRows)

	tagName := rows[0].TagName
	csvName := "output.csv"
	csvDir := filepath.Join(s.trainingPath, tagName)
	csvFile := filepath.Join(csvDir, csvName)
	if err := os.MkdirAll(csvDir, 0766); err != nil {
		return "", fmt.Errorf("mkdir: %w", err)
	}

	file, err := os.Create(csvFile)
	if err != nil {
		return "", fmt.Errorf("os create: %w", err)
	}

	writer := csv.NewWriter(file)
	if err := writer.WriteAll(csvRows); err != nil {
		return "", fmt.Errorf("writeAll csv:%w", err)
	}

	return csvFile, nil
}

func remapImageIDsToTagsSVC(repo training.ImageIDstoTags) ImageIDstoTags {
	svc := make(ImageIDstoTags, len(repo))
	for k, v := range repo {
		svc[k] = v
	}

	return svc
}

func remapListByReqRowToSVC(repo training.ListByReqRow) ListQueryRow {
	return ListQueryRow{
		ID:         repo.ID,
		Hash:       repo.Hash,
		BatchID:    repo.BatchID,
		RelPath:    repo.RelPath,
		TagName:    repo.TagName,
		ModelScore: repo.ModelScore,
		UserScore:  repo.UserScore,
	}
}

func remapListByTagRowToSVC(repo training.ListByTagRow) ListByTagRow {
	return ListByTagRow{
		ID:         repo.ID,
		Hash:       repo.Hash,
		BatchPath:  repo.BatchPath,
		RelPath:    repo.RelPath,
		TagName:    repo.TagName,
		ModelScore: repo.ModelScore,
		UserScore:  repo.UserScore,
	}
}

func remapTagsToSVC(repo training.Tag) Tag {
	return Tag{
		ID:        repo.ID,
		Name:      repo.Name,
		Desc:      repo.Desc,
		DeletedAt: repo.DeletedAt,
	}
}

func remapScoreCompositionItemToSVC(repo []training.ScoreCompositionItem) []ScoreCompositionItem {
	svcItems := make([]ScoreCompositionItem, 0, len(repo))

	for _, repoItem := range repo {
		svcItem := ScoreCompositionItem{
			Score: repoItem.Score,
			Count: repoItem.Count,
		}
		svcItems = append(svcItems, svcItem)
	}

	return svcItems
}

func listQueryToCSV(rows []ListByTagRow) [][]string {
	csvRows := make([][]string, 0, len(rows)+1)
	csvRows = append(csvRows, ListQueryRowHeaders)

	for _, row := range rows {
		id := strconv.Itoa(int(row.ID))
		userScore := strconv.FormatFloat(float64(*row.UserScore), 'f', 2, 32)
		csvRow := []string{
			id,
			row.BatchPath,
			row.RelPath,
			userScore,
		}
		csvRows = append(csvRows, csvRow)
	}

	return csvRows
}

func initValidator() (*validator.Validate, error) {
	v := validator.New()

	if err := v.RegisterValidation("userScore", validateUserScore, true); err != nil {
		return nil, fmt.Errorf("validate userScore: %w", err)
	}

	if err := v.RegisterValidation("IDwithPTR", validateIDorNull, true); err != nil {
		return nil, fmt.Errorf("validate IDwithPTR: %w", err)
	}

	v.RegisterStructValidation(validateListImagesReq, ListRowsByReq{})

	return v, nil
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

	if score >= 0 && score <= 1 {
		return true
	}

	return false
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

func validateListImagesReq(sl validator.StructLevel) {
	req := sl.Current().Interface().(ListRowsByReq)

	hasId := req.Cursor > 0

	if !req.Next && !hasId {
		sl.ReportError(req.Next, "Next", "next", "cursorpair", "")
	}
}
