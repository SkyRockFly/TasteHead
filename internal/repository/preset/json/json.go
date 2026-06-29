package presetJson

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"scraper/internal/repository/preset"
)

type Repository struct {
	JSONPath string
}

func NewRepository(jsonPath string) *Repository {
	return &Repository{
		JSONPath: jsonPath,
	}
}

func (r *Repository) Create(profile preset.Profile) error {
	bytes, err := json.Marshal(profile)
	if err != nil {
		return fmt.Errorf("marshal JSON:%w", err)
	}
	fpath := filepath.Join(r.JSONPath, profile.Name)
	f, err := os.Create(fpath)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer f.Close()
	if _, err := f.Write(bytes); err != nil {
		return fmt.Errorf("write in file: %w", err)
	}

	return nil
}

func (r *Repository) Delete(name string) error {
	fpath := filepath.Join(r.JSONPath, name)
	if err := os.Remove(fpath); err != nil {
		return fmt.Errorf("delete file: %w", err)
	}
	return nil
}

func (r *Repository) List() ([]preset.Profile, error) {
	profiles := make([]preset.Profile, 0)
	if err := filepath.WalkDir(r.JSONPath, func(path string, d fs.DirEntry, WalkErr error) error {
		if WalkErr != nil {
			return fmt.Errorf("walk dir: %w", WalkErr)
		}
		if d.IsDir() {
			return fmt.Errorf("dirs are not supported")
		}
		bytes, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read file: %w", err)
		}
		var profile preset.Profile
		if err := json.Unmarshal(bytes, &profile); err != nil {
			return fmt.Errorf("unmarshal json: %w", err)
		}
		profiles = append(profiles, profile)

		return nil
	}); err != nil {
		return nil, fmt.Errorf("walk dir: %w", err)
	}

	return profiles, nil
}
