package testutil

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func NormalizeJSON(t *testing.T, s string) string {
	t.Helper()
	var buf bytes.Buffer
	if err := json.Compact(&buf, []byte(s)); err != nil {
		return s
	}
	return buf.String()
}

func SetupPgxPool(dbURL string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		return nil, fmt.Errorf("pgxpool create: %w", err)
	}
	if err := pool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("pgxpool ping: %w", err)
	}

	return pool, nil
}

func ClearFixtures(pool *pgxpool.Pool, resetFixtures string) error {
	_, err := pool.Exec(context.Background(), resetFixtures)
	if err != nil {
		return fmt.Errorf("exec reset fixtures:%w", err)
	}

	return nil
}

func LoadFixtures(pool *pgxpool.Pool, path string, resetFixtures string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	_, err = pool.Exec(context.Background(), resetFixtures)
	if err != nil {
		return fmt.Errorf("exec reset fixtures:%w", err)
	}

	_, err = pool.Exec(context.Background(), string(data))
	if err != nil {
		return fmt.Errorf("exec fixture: %w", err)
	}

	return nil
}

// Creates a copy of a dir with pictures
func CreateTempDirFromPicFixtureDir(src, dest string) error {
	dirInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("stat: %w", err)
	}

	if !dirInfo.IsDir() {
		return fmt.Errorf("not a dir")
	}

	if err := os.RemoveAll(dest); err != nil {
		return fmt.Errorf("clear dest: %w", err)
	}

	if err := os.MkdirAll(dest, dirInfo.Mode().Perm()); err != nil {
		return fmt.Errorf("mkdirAll: %w", err)
	}

	if err := filepath.WalkDir(src, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("walk: %w", err)
		}

		rel, err := filepath.Rel(src, path)
		if err != nil {
			return fmt.Errorf("rel: %w", err)
		}

		if rel == "." {
			return nil
		}

		target := filepath.Join(dest, rel)

		if d.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink is not supported: %s", path)
		}

		info, err := d.Info()
		if err != nil {
			return fmt.Errorf("dirInfo: %w", err)
		}
		if d.IsDir() {
			if err := os.MkdirAll(target, info.Mode().Perm()); err != nil {
				return fmt.Errorf("mkdirAll: %w", err)
			}
			return nil
		}

		in, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("open file: %w", err)
		}
		defer in.Close()

		out, err := os.Create(target)
		if err != nil {
			return fmt.Errorf("create file: %w", err)
		}

		if _, err := io.Copy(out, in); err != nil {
			return fmt.Errorf("io copy: %w", err)
		}
		out.Close()
		in.Close()

		return nil
	}); err != nil {
		return fmt.Errorf("walkDir: %w", err)
	}

	return nil
}

func ClearDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read dir: %w", err)
	}

	for _, entry := range entries {
		path := filepath.Join(dir, entry.Name())
		if err := os.RemoveAll(path); err != nil {
			return fmt.Errorf("remove %s: %w", path, err)
		}
	}

	return nil
}

func ToJSON(t *testing.T, v any) io.Reader {
	t.Helper()

	switch x := v.(type) {
	case string:
		return strings.NewReader(x)
	case []byte:
		return bytes.NewReader(x)
	default:
		b, err := json.Marshal(v)
		require.NoError(t, err)
		return bytes.NewReader(b)
	}
}
