package scrapestate

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"scraper/internal/pkg/apperror"
)

// Struct for updating state.json, all changes through funcs are saving in json file
type StateManifest struct {
	CurrentPage    string `json:"current_page"`
	NextURLPage    string `json:"next_page"`
	PagesRemaining int    `json:"pages_remaining"`
	IsProcessed    bool   `json:"is_processed"`

	dirPath string `json:"-"`
}

type ScrapeManifest struct {
	PostSelector     string `json:"post_selector"`
	ImageAttr        string `json:"image_attr"`
	NextPageSelector string `json:"next_page_selector"`
	NextPageAttr     string `json:"next_page_attr"`
}

type JSONFailReport struct {
	OriginURL string               `json:"origin_url"`
	Fails     []JSONFailedDownload `json:"fails"`
}

type JSONFailedDownload struct {
	Issue      apperror.DownloadIssue `json:"issue"`
	URL        string                 `json:"url"`
	Repeatable bool                   `json:"repeatable"`
}

func NewStateManifest(dirPath string) (*StateManifest, error) {
	s := &StateManifest{
		dirPath: filepath.Join(dirPath, "state.json"),
	}

	if err := s.SaveJSON(); err != nil {
		return nil, fmt.Errorf("save json: %w", err)
	}

	return s, nil
}

func (s *StateManifest) SetPage(pagesRemaining int, currentPage string) error {
	s.CurrentPage = currentPage
	s.PagesRemaining = pagesRemaining

	if err := s.SaveJSON(); err != nil {
		return fmt.Errorf("save json: %w", err)
	}

	return nil
}

func (s *StateManifest) SaveJSON() error {
	if s.dirPath == "" {
		return fmt.Errorf("state manifest file path is empty")
	}

	file, err := os.Create(s.dirPath)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}

	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")

	if err := enc.Encode(s); err != nil {
		file.Close()
		return fmt.Errorf("encode: %w", err)
	}

	if err := file.Close(); err != nil {
		return fmt.Errorf("close file: %w", err)
	}

	return nil
}

func (s *StateManifest) SetNextURLPage(url string) error {
	s.NextURLPage = url
	if err := s.SaveJSON(); err != nil {
		return fmt.Errorf("save json: %w", err)
	}

	return nil
}

func (s *StateManifest) NextPage() error {
	s.PagesRemaining -= 1
	if err := s.SaveJSON(); err != nil {
		return fmt.Errorf("save json: %w", err)
	}

	return nil
}

func (s *StateManifest) ClearRemainingPages() error {
	s.PagesRemaining = 0
	if err := s.SaveJSON(); err != nil {
		return fmt.Errorf("save json: %w", err)
	}

	return nil
}

func (s *StateManifest) MarkProcessed() error {
	s.IsProcessed = true
	if err := s.SaveJSON(); err != nil {
		return fmt.Errorf("save json: %w", err)
	}

	return nil
}

func ReadStateManifest(dirPath string) (*StateManifest, error) {
	filePath := filepath.Join(dirPath, "state.json")

	file, err := os.Open(filePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("state file: %w", err)
		}
		return nil, fmt.Errorf("open state file: %w", err)
	}
	defer file.Close()

	var s StateManifest
	if err := json.NewDecoder(file).Decode(&s); err != nil {
		return nil, fmt.Errorf("decode state: %w", err)
	}

	s.dirPath = filePath

	return &s, nil
}

func CreateScrapeManifest(dirPath string, report ScrapeManifest) error {
	filePath := filepath.Join(dirPath, "scrapeManifest.json")
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}

	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")

	if err := enc.Encode(report); err != nil {
		file.Close()
		return fmt.Errorf("encode: %w", err)
	}

	if err := file.Close(); err != nil {
		return fmt.Errorf("close file: %w", err)
	}

	return nil
}

func ReadSсrapeManifest(dirPath string) (ScrapeManifest, error) {
	filePath := filepath.Join(dirPath, "scrapeManifest.json")
	b, err := os.ReadFile(filePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ScrapeManifest{}, fmt.Errorf("state file: %w", err)
		}
		return ScrapeManifest{}, fmt.Errorf("create file:%w", err)
	}

	var scrape ScrapeManifest
	if err := json.Unmarshal(b, &scrape); err != nil {
		return ScrapeManifest{}, fmt.Errorf("unmarshal state:%w", err)
	}

	return scrape, nil
}

func AppendFailReport(dirPath string, report JSONFailReport) error {
	failPath := filepath.Join(dirPath, "fails.jsonl")

	file, err := os.OpenFile(failPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open fail report: %w", err)
	}
	defer file.Close()

	if err := json.NewEncoder(file).Encode(report); err != nil {
		return fmt.Errorf("encode fail report: %w", err)
	}

	return nil
}

func ReadFails(dirPath string) ([]JSONFailReport, error) {
	filePath := filepath.Join(dirPath, "fails.jsonl")
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("read file:%w", err)
	}
	defer file.Close()

	fails := make([]JSONFailReport, 0)
	dec := json.NewDecoder(file)
	var fail JSONFailReport
	for {
		if err := dec.Decode(&fail); err != nil {

			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("decode jsonl row: %w", err)
		}
		fails = append(fails, fail)
	}

	if len(fails) == 0 {
		return nil, nil
	}

	return fails, nil
}
