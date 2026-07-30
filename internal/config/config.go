package config

import (
	"fmt"
	"os"
	"path/filepath"
	"scraper/internal/pkg/kit"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog"
	"gopkg.in/yaml.v3"
)

type AppConfig struct {
	Log    LoggerConfig `yaml:"logger" validate:"required"`
	Env    EnvConfig    `yaml:"env_paths" validate:"required"`
	DB     DBConfig     `yaml:"db_config" validate:"required"`
	Server ServerConfig `yaml:"server" validate:"required"`
	Scrape ScrapeConfig `yaml:"scrape_config" validate:"required"`
}

type ServerConfig struct {
	Port int `yaml:"port" validate:"min=1,max=65535"`
}

type EnvConfig struct {
	DownloadDir   string `yaml:"download_dir" validate:"required"`
	ModelDir      string `yaml:"model_dir" validate:"required"`
	ImportDir     string `yaml:"import_dir" validate:"required"`
	PythonVenvDir string `yaml:"python_venv_dir" validate:"required"`
	ScriptsDir    string `yaml:"scripts_dir" validate:"required"`
	TrainingDir   string `yaml:"training_dir" validate:"required"`
}

type LoggerConfig struct {
	Timestamp   string `yaml:"timestamp" validate:"required,timestamp"`
	FormatLevel string `yaml:"formatlevel" validate:"required,formatlevel"`
	Level       string `yaml:"level" validate:"required,loglevel"`
}

type ScrapeConfig struct {
	UserAgent          string        `yaml:"user_agent" validate:"required"`
	DownloadImagePause time.Duration `yaml:"download_image_pause" validate:"gt=0s"`
	DownloadPagePause  time.Duration `yaml:"download_page_pause" validate:"gt=0s"`
}

type DBConfig struct {
	URL string `yaml:"url" validate:"url"`
}

func initValidator() (*validator.Validate, error) {
	v := validator.New()

	if err := v.RegisterValidation("loglevel", validateZerologLevel); err != nil {
		return nil, fmt.Errorf("validate logLevel: %w", err)
	}

	if err := v.RegisterValidation("formatlevel", validateFormatLevel); err != nil {
		return nil, fmt.Errorf("validate formatLevel: %w", err)
	}

	if err := v.RegisterValidation("timestamp", validateTimestamp); err != nil {
		return nil, fmt.Errorf("validate timestamp: %w", err)
	}

	return v, nil
}

func validateZerologLevel(fl validator.FieldLevel) bool {
	s := strings.TrimSpace(fl.Field().String())
	_, err := zerolog.ParseLevel(s)
	return err == nil
}

func validateFormatLevel(fl validator.FieldLevel) bool {
	s := fl.Field().String()
	defer func() { _ = recover() }()
	out := fmt.Sprintf(s, "info")
	return !strings.Contains(out, "%!")
}

func validateTimestamp(fl validator.FieldLevel) bool {
	s := fl.Field().String()
	return strings.Contains(s, "2006")
}

func GetAppConfig() (*AppConfig, error) {
	appConfig := &AppConfig{}

	v, err := initValidator()
	if err != nil {
		return nil, fmt.Errorf("GetConfigApp: %w", err)
	}

	data, err := os.ReadFile("./config/config.yaml")
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}
	if err := yaml.Unmarshal(data, appConfig); err != nil {
		return nil, fmt.Errorf("yaml unmarshal: %w", err)
	}

	if err := kit.ValidateStruct(v, appConfig); err != nil {
		return nil, fmt.Errorf("validate struct: %w", err)
	}

	if err := makeAbs(&appConfig.Env); err != nil {
		return nil, fmt.Errorf("abs envConfig: %w", err)
	}

	if err := ensureRuntimeDirs(&appConfig.Env); err != nil {
		return nil, fmt.Errorf("ensure runtime dirs: %w", err)
	}

	if err := validatePathLayout(&appConfig.Env); err != nil {
		return nil, fmt.Errorf("validatePathLayout: %w", err)
	}

	return appConfig, nil
}

func makeAbs(env *EnvConfig) error {
	var err error

	env.DownloadDir, err = absPath(env.DownloadDir, "download dir")
	if err != nil {
		return err
	}

	env.ModelDir, err = absPath(env.ModelDir, "model dir")
	if err != nil {
		return err
	}

	env.ImportDir, err = absPath(env.ImportDir, "import dir")
	if err != nil {
		return err
	}

	env.ScriptsDir, err = absPath(env.ScriptsDir, "scripts dir")
	if err != nil {
		return err
	}

	env.TrainingDir, err = absPath(env.TrainingDir, "training dir")
	if err != nil {
		return err
	}

	env.PythonVenvDir, err = absPath(env.PythonVenvDir, "python venv dir")
	if err != nil {
		return err
	}

	return nil
}

func absPath(p string, name string) (string, error) {
	if p == "" {
		return "", fmt.Errorf("%s is empty", name)
	}

	abs, err := filepath.Abs(p)
	if err != nil {
		return "", fmt.Errorf("abs %s: %w", name, err)
	}

	return filepath.Clean(abs), nil
}

func validatePathLayout(env *EnvConfig) error {
	if err := pathsMustNotOverlap("download dir", env.DownloadDir, "import dir", env.ImportDir); err != nil {
		return err
	}

	return nil
}

func pathsMustNotOverlap(aName, aPath, bName, bPath string) error {
	a := filepath.Clean(aPath)
	b := filepath.Clean(bPath)

	if !filepath.IsAbs(a) {
		return fmt.Errorf("%s must be absolute: %q", aName, aPath)
	}

	if !filepath.IsAbs(b) {
		return fmt.Errorf("%s must be absolute: %q", bName, bPath)
	}

	bInsideA, err := pathInsideOrSame(a, b)
	if err != nil {
		return fmt.Errorf("check %s/%s overlap: %w", aName, bName, err)
	}

	aInsideB, err := pathInsideOrSame(b, a)
	if err != nil {
		return fmt.Errorf("check %s/%s overlap: %w", bName, aName, err)
	}

	switch {
	case aInsideB && bInsideA:
		return fmt.Errorf("%s and %s must be different: %q", aName, bName, a)

	case bInsideA:
		return fmt.Errorf("%s must not be inside %s: %q inside %q", bName, aName, b, a)

	case aInsideB:
		return fmt.Errorf("%s must not be inside %s: %q inside %q", aName, bName, a, b)
	}

	return nil
}

func pathInsideOrSame(parent, child string) (bool, error) {
	parent = filepath.Clean(parent)
	child = filepath.Clean(child)

	rel, err := filepath.Rel(parent, child)
	if err != nil {
		return false, err
	}

	if rel == "." {
		return true, nil
	}

	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return false, nil
	}

	if filepath.IsAbs(rel) {
		return false, nil
	}

	return true, nil
}

func ensureRuntimeDirs(env *EnvConfig) error {
	dirs := []string{
		env.DownloadDir,
		env.ImportDir,
		env.TrainingDir,
		env.ModelDir,
	}

	for _, dir := range dirs {
		if strings.TrimSpace(dir) == "" {
			return fmt.Errorf("runtime dir is empty")
		}

		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("mkdir %s: %w", dir, err)
		}
	}

	return nil
}
