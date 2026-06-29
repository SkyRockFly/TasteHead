package config

import (
	"fmt"
	"os"
	"path/filepath"
	customvalidator "scraper/internal/customValidator"

	"gopkg.in/yaml.v3"
)

type SelectedModelName struct {
	ModelName string `yaml:"model_name"`
}

type SetModelNameReq struct {
	ModelName string `validate:"safeModelName"`
}

func GetModelName(path string) (string, error) {
	appConfig := &SelectedModelName{}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read file: %w", err)
	}
	if err := yaml.Unmarshal(data, appConfig); err != nil {
		return "", fmt.Errorf("yaml unmarshal: %w", err)
	}
	if err := customvalidator.ValidateModelNameValue(appConfig.ModelName); err != nil {
		return "", fmt.Errorf("validate model name: %w", err)
	}

	return appConfig.ModelName, nil
}

func SetModelName(modelName string, path string) error {
	if err := customvalidator.ValidateModelNameValue(modelName); err != nil {
		return fmt.Errorf("validate model name: %w", err)
	}
	appConfig := SelectedModelName{
		ModelName: modelName,
	}

	data, err := yaml.Marshal(appConfig)
	if err != nil {
		return fmt.Errorf("yaml marshal: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir selected model config dir: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}
