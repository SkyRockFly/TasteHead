package htmlparser

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"scraper/internal/pkg/apperror"
	"scraper/internal/repository/scraper"
	scrapestate "scraper/internal/scrapeState"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/blake2b"
)

type Row map[string]string

var validExts = []string{".jpg", ".png", ".jpeg", ".webp", ".gif"}

type ScrapeConfig struct {
	UserAgent          string
	DownloadImagePause time.Duration
	DownloadPagePause  time.Duration
}

type EnvPaths struct {
	PythonVenvDir string
	ScriptsDir    string
}

type Repository struct {
	client *http.Client
	paths  EnvPaths
	cfg    ScrapeConfig
}

type HeaderRow map[string]string

func NewRepository(paths EnvPaths, cfg ScrapeConfig) *Repository {
	return &Repository{
		paths: paths,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
		cfg: cfg,
	}
}

func (r *Repository) ParseHTML(req scraper.ParseReq) (scraper.ParseResp, error) {
	httpResp, err := r.sendGetRequest(req.URL)
	if err != nil {
		return scraper.ParseResp{}, fmt.Errorf("send request: %w", err)
	}
	defer httpResp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(httpResp.Body)
	if err != nil {
		return scraper.ParseResp{}, fmt.Errorf("new document from reader: %w", err)
	}
	var urls []string
	doc.Find(req.PostSelector).Each(func(i int, s *goquery.Selection) {
		url, ok := s.Attr(req.ImageAttr)
		if !ok {
			log.Error().Msg("no url")
		}
		urls = append(urls, url)
	})
	next := doc.Find(req.NextPageSelector).First()

	nextURL, ok := next.Attr(req.NextPageAttr)
	if !ok {
		log.Error().Msg("no next url")
	}

	resp := scraper.ParseResp{
		URL:     urls,
		NextURL: nextURL,
	}

	time.Sleep(r.cfg.DownloadPagePause)
	return resp, nil
}

func (r *Repository) DownloadPic(downloadPath string, url string) *scraper.FailedDownload {
	filename, err := getFilename(url)
	if err != nil {
		issue := unwrapDownloadError(err)
		fail := handleDownloadError(url, "getFilename", err, issue, false)
		return &fail
	}

	outPath := filepath.Join(downloadPath, filename)

	if _, err := os.Stat(outPath); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		fail := handleDownloadError(url, "statOutputFile", err, apperror.IssueFileStatError, true)
		return &fail
	}

	resp, err := r.sendGetRequest(url)
	if err != nil {
		fail := handleDownloadError(url, "sendGetRequest", err, apperror.IssueGetRequestError, true)
		return &fail
	}
	defer resp.Body.Close()

	if err := createPic(outPath, resp); err != nil {
		fail := handleDownloadError(url, "sendGetRequest", err, apperror.IssueCreatePic, true)
		return &fail
	}

	time.Sleep(r.cfg.DownloadImagePause)

	return nil
}

// Dest directories must exist
func (r *Repository) CopyFile(src, dest string) error {
	fileInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("stat: %w", err)
	}

	if fileInfo.IsDir() {
		return fmt.Errorf("not a file")
	}

	bytes, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("ReadFile: %w", err)
	}

	file, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer file.Close()

	if _, err := file.Write(bytes); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}

func (r *Repository) MoveDir(src, dest string) error {
	dirInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("stat: %w", err)
	}

	if !dirInfo.IsDir() {
		return fmt.Errorf("not a dir")
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

		in, err := os.Open(src)
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
		defer out.Close()

		return nil
	}); err != nil {
		return fmt.Errorf("walkDir: %w", err)
	}

	return nil
}

func (r *Repository) TrainModel(ctx context.Context, req scraper.TrainModelReq) error {
	venv, err := makeVenvPath(r.paths.PythonVenvDir)
	if err != nil {
		return fmt.Errorf("venv path: %w", err)
	}
	scriptName := filepath.Join(r.paths.ScriptsDir, "ingest_clip.py")

	cmdArgs := []string{
		scriptName,
		"--csvPath", req.CsvPath,
		"--dataPath", req.DownloadPath,
		"--train_mode",
	}
	if err := executeCommand(ctx, venv, cmdArgs...); err != nil {
		return fmt.Errorf("ingest_clip: %w", err)
	}

	scriptName = filepath.Join(r.paths.ScriptsDir, "train_network.py")
	csvDir := filepath.Dir(req.CsvPath)
	vectPath := filepath.Join(csvDir, "output", "clip_training_vecs.npy")
	scorePath := filepath.Join(csvDir, "output", "vect_training_info.npy")
	cmdArgs = []string{
		scriptName,
		"--learn_mode",
		"--vect_path", vectPath,
		"--score_path", scorePath,
		"--output", req.OutputPath,
	}
	if err := executeCommand(ctx, venv, cmdArgs...); err != nil {
		return fmt.Errorf("train_network: %w", err)
	}

	return nil
}

func (r *Repository) ProcessFiles(ctx context.Context, req scraper.ProcessFilesReq) error {
	venv, err := makeVenvPath(r.paths.PythonVenvDir)
	if err != nil {
		return fmt.Errorf("venv path: %w", err)
	}
	scriptName := filepath.Join(r.paths.ScriptsDir, "fileManager.py")

	cmdArgs := []string{
		scriptName,
		"--dir", req.DownloadPath,
	}
	if err := executeCommand(ctx, venv, cmdArgs...); err != nil {
		return fmt.Errorf("file manager: %w", err)
	}

	scriptName = filepath.Join(r.paths.ScriptsDir, "ingest_clip.py")
	csvPath := filepath.Join(req.DownloadPath, "output.csv")
	cmdArgs = []string{
		scriptName,
		"--csvPath", csvPath,
	}

	if err := executeCommand(ctx, venv, cmdArgs...); err != nil {
		return fmt.Errorf("ingest_clip.py: %w", err)
	}

	scriptName = filepath.Join(r.paths.ScriptsDir, "train_network.py")
	vectPath := filepath.Join(req.DownloadPath, "output", "clip_eval_vecs.npy")

	cmdArgs = []string{
		scriptName,
		"--vect_path", vectPath,
		"--model_path", req.ModelPath,
	}
	if err := executeCommand(ctx, venv, cmdArgs...); err != nil {
		return fmt.Errorf("eval vects: %w", err)
	}

	return nil
}

func (r *Repository) ListUnfinishedBatches(dirPath string) ([]string, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("read download dir: %w", err)
	}

	dirs := make([]string, 0)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		statePath := filepath.Join(dirPath, entry.Name())
		state, err := scrapestate.ReadStateManifest(statePath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, fmt.Errorf("read state dir: %w", err)
		}

		if state.PagesRemaining != 0 || !state.IsProcessed {
			dirs = append(dirs, entry.Name())
		}
	}

	if len(dirs) == 0 {
		return nil, fmt.Errorf("dirs: %w", apperror.ErrNotFound)
	}

	return dirs, nil
}

func (r *Repository) ListBatchesWithFails(dirPath string) ([]string, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("read download dir: %w", err)
	}

	dirs := make([]string, 0)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		statePath := filepath.Join(dirPath, entry.Name())
		state, err := scrapestate.ReadStateManifest(statePath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, fmt.Errorf("read state manifest: %w", err)
		}

		if !state.IsProcessed || state.PagesRemaining != 0 {
			continue
		}

		fails, err := scrapestate.ReadFails(statePath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, fmt.Errorf("read fails: %w", err)
		}

		repeatableFails := make([]scrapestate.JSONFailReport, 0)
		for _, reports := range fails {
			for _, report := range reports.Fails {
				if report.Repeatable {
					repeatableFails = append(repeatableFails, reports)
					break
				}
			}
		}

		if len(repeatableFails) != 0 {
			dirs = append(dirs, entry.Name())
		}
	}

	if len(dirs) == 0 {
		return nil, fmt.Errorf("dirs: %w", apperror.ErrNotFound)
	}

	return dirs, nil
}

func (r *Repository) ListModels(dirPath string) ([]string, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("read download dir: %w", err)
	}

	models := make([]string, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if filepath.Ext(entry.Name()) != ".pt" {
			continue
		}
		models = append(models, entry.Name())
	}

	if len(models) == 0 {
		return nil, fmt.Errorf("dirs: %w", apperror.ErrNotFound)
	}

	slices.Sort(models)

	return models, nil
}

// allowed: thumbs dir, all images and state files
func (r *Repository) MoveRetryDirToParent(retryDir string) error {
	parentDir := filepath.Dir(retryDir)
	if err := filepath.Walk(retryDir, func(path string, info fs.FileInfo, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("walkErr: %w", walkErr)
		}

		rel, err := filepath.Rel(retryDir, path)
		if err != nil {
			return fmt.Errorf("rel: %w", err)
		}

		if rel == "." {
			return nil
		}

		if info.IsDir() {
			if rel == "thumbs" {
				return nil
			}
			return filepath.SkipDir
		}

		if !isAllowedRetryFile(rel) {
			return nil
		}

		destPath := filepath.Join(parentDir, rel)

		if err := os.Rename(path, destPath); err != nil {
			return fmt.Errorf("rename: %w", err)
		}

		return nil
	}); err != nil {
		return fmt.Errorf("walkDir: %w", err)
	}

	if err := os.RemoveAll(retryDir); err != nil {
		return fmt.Errorf("remove dir:%w", err)
	}

	return nil
}

func (r *Repository) FindDirs(root string) ([]string, error) {
	dirs := make([]string, 0)
	if err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("walk dir : %w", err)
		}
		if d.IsDir() {
			return nil
		}

		if !strings.HasSuffix(path, ".csv") {
			return nil
		}

		relative, err := filepath.Rel(root, filepath.Dir(path))
		if err != nil {
			return fmt.Errorf("make relative: %w", err)
		}

		if strings.Compare("hashes", relative) == 0 {
			return nil
		}

		if _, err := readCSV(path); err != nil {
			return fmt.Errorf("read csv: %s, for integrity: %w", path, err)
		}

		dirs = append(dirs, relative)
		return nil
	}); err != nil {
		return nil, fmt.Errorf("walk dir: %w", err)
	}

	return dirs, nil
}

func (r *Repository) CSVToRows(csvPath string) ([]scraper.CSVRow, error) {
	records, err := readCSV(csvPath)
	if err != nil {
		return nil, fmt.Errorf("read CSV: %w", err)
	}
	headers := records[0]
	record := records[1:]

	maps := make([]HeaderRow, 0, len(record))

	for _, row := range record {
		if len(row) == 0 {
			continue
		}
		rowMap := HeaderRow{}
		for i, header := range headers {
			var val string

			if i < len(row) {
				val = row[i]
			} else {
				val = ""
			}
			header = strings.TrimSpace(header)
			val = strings.TrimSpace(val)
			rowMap[header] = val
		}
		maps = append(maps, rowMap)
	}

	hashRows := make([]scraper.CSVRow, 0, len(maps))
	for _, row := range maps {
		hash := row["hash"]
		dir := row["dir"]
		path := row["path"]

		modelScore, err := strconv.ParseFloat(row["model_score"], 32)
		if err != nil {
			return nil, fmt.Errorf("parse float model_score: %w", err)
		}

		var userScore *float32
		if row["user_score"] != "" {
			val, err := strconv.ParseFloat(row["user_score"], 32)
			if err != nil {
				return nil, fmt.Errorf("parse float user_score: %w", err)
			}
			vval := float32(val)
			userScore = &vval
		} else {
			userScore = nil
		}

		row := scraper.CSVRow{
			Path:       path,
			Dir:        dir,
			Hash:       hash,
			ModelScore: float32(modelScore),
			UserScore:  userScore,
		}
		hashRows = append(hashRows, row)

	}

	return hashRows, nil
}

func (r *Repository) ComputeHashesInDir(dir string) ([]scraper.File, error) {
	list := make([]scraper.File, 0)
	if err := filepath.Walk(dir, func(path string, info fs.FileInfo, walkErr error) error {
		if walkErr != nil {
			fmt.Printf("dir:%s\n\n", dir)
			return fmt.Errorf("walk err: %w", walkErr)
		}

		if filepath.Base(filepath.Dir(path)) == "thumbs" || filepath.Base(filepath.Dir(path)) == "output" {
			return nil
		}

		isValid := false
		for _, ext := range validExts {
			if !strings.HasSuffix(info.Name(), ext) {
				isValid = true
				break
			}
		}
		if !isValid {
			return nil
		}

		bytes, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		hashBuilder, err := blake2b.New(32, nil)
		if err != nil {
			return fmt.Errorf("create blake2b hash: %w", err)
		}
		if _, err := hashBuilder.Write(bytes); err != nil {
			return fmt.Errorf("write to blake2b: %w", err)
		}
		hashBytes := hashBuilder.Sum(nil)
		hash := base64.RawURLEncoding.EncodeToString(hashBytes)

		relPath, err := filepath.Rel(dir, path)
		if err != nil {
			return fmt.Errorf("make rel, file: %s: %w", path, err)
		}
		img := scraper.File{
			Dir:  filepath.Base(filepath.Dir(path)),
			Hash: hash,
			Path: relPath,
		}
		list = append(list, img)

		return nil
	}); err != nil {
		return nil, fmt.Errorf("walk: %w", err)
	}

	return list, nil
}

func (r *Repository) ComputeFileHash(path string) (string, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("readFile: %w", err)
	}

	hashBuilder, err := blake2b.New(32, nil)
	if err != nil {
		return "", fmt.Errorf("create blake2b hash: %w", err)
	}
	if _, err := hashBuilder.Write(bytes); err != nil {
		return "", fmt.Errorf("write to blake2b: %w", err)
	}
	hashBytes := hashBuilder.Sum(nil)
	hash := base64.RawURLEncoding.EncodeToString(hashBytes)

	return hash, nil
}

func (r *Repository) DeleteImages(file string) error {
	if err := os.Remove(file); err != nil {
		return fmt.Errorf("remove file, path: %s : %w", file, err)
	}

	return nil
}

func makeVenvPath(venvDir string) (string, error) {
	if venvDir == "" {
		return "", fmt.Errorf("no venv")
	}

	if runtime.GOOS == "windows" {
		return filepath.Join(venvDir, "Scripts", "python.exe"), nil
	}

	return filepath.Join(venvDir, "bin", "python"), nil
}

func handleDownloadError(URL, funcName string, err error, issue apperror.DownloadIssue, repeat bool) scraper.FailedDownload {
	log.Info().Err(fmt.Errorf("%s: %w", funcName, err)).Msg("download image")
	fail := scraper.FailedDownload{
		Warn:       issue,
		URL:        URL,
		Repeatable: repeat,
	}

	return fail
}

func unwrapDownloadError(err error) apperror.DownloadIssue {
	var issue apperror.DownloadIssue
	switch {
	case errors.Is(err, apperror.ErrUnsupportedExt):
		issue = apperror.IssueUnsupportedExt
	case errors.Is(err, apperror.ErrBadFilename):
		issue = apperror.IssueBadFilename
	default:
		issue = apperror.IssueUnknown
	}

	return issue
}

func readCSV(fpath string) ([][]string, error) {
	f, err := os.Open(fpath)
	if err != nil {
		return nil, fmt.Errorf("open csvPath: %w", err)
	}
	defer f.Close()
	reader := csv.NewReader(f)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("read all: %w", err)
	}
	return records, nil
}

func executeCommand(ctx context.Context, exePath string, args ...string) error {
	cmd := exec.CommandContext(
		ctx,
		exePath,
		args...,
	)

	cmd.Env = append(os.Environ(),
		"HF_HUB_OFFLINE=1",
		"HF_HUB_VERBOSITY=error",
		"PYTORCH_ALLOC_CONF=expandable_segments:True",
		"PYTHONUNBUFFERED=1",
	)

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("stderr pipe: %w", err)
	}

	var stdoutBuf bytes.Buffer
	var stderrBuf bytes.Buffer

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("cmd start: %w", err)
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		streamPipe("PYTHON STDOUT", stdoutPipe, &stdoutBuf)
	}()

	go func() {
		defer wg.Done()
		streamPipe("PYTHON STDERR", stderrPipe, &stderrBuf)
	}()

	waitErr := cmd.Wait()
	wg.Wait()

	if waitErr != nil {
		stdoutStr := strings.TrimSpace(stdoutBuf.String())
		stderrStr := strings.TrimSpace(stderrBuf.String())

		return fmt.Errorf("cmd: %w, stdout: %s, stderr: %s", waitErr, stdoutStr, stderrStr)
	}
	return nil
}

func getFilename(fURL string) (string, error) {
	u, err := url.Parse(fURL)
	if err != nil {
		return "", fmt.Errorf("%w: parse filename URL, %s: %v", apperror.ErrBadFilename, fURL, err)
	}

	filename := filepath.Base(u.Path)
	if filename == "." || filename == "/" {
		return "", fmt.Errorf("%w: filename %s is empty ", apperror.ErrBadFilename, fURL)
	}

	if !slices.Contains(validExts, filepath.Ext(filename)) {
		return "", fmt.Errorf("%w : %s", apperror.ErrBadFilename, filepath.Ext(filename))
	}
	return filename, nil
}

func streamPipe(prefix string, r io.Reader, dst *bytes.Buffer) {
	scanner := bufio.NewScanner(r)

	// 64K
	scanner.Buffer(make([]byte, 1024), 1024*1024*10)

	for scanner.Scan() {
		line := scanner.Text()

		dst.WriteString(line)
		dst.WriteByte('\n')

		fmt.Printf("%s: %s\n", prefix, line)
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("%s read error: %v\n", prefix, err)
	}
}

func (r *Repository) sendGetRequest(picURL string) (*http.Response, error) {
	req, err := http.NewRequest("GET", picURL, nil)
	if err != nil {
		return nil, fmt.Errorf("make request: %w", err)
	}

	req.Header.Set("User-Agent", r.cfg.UserAgent)
	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get request: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("http code resp is not 200: %s", picURL)
	}

	return resp, nil
}

func createPic(outPath string, resp *http.Response) error {
	partPath := outPath + ".part"
	f, err := os.Create(partPath)
	if err != nil {
		_ = os.Remove(partPath)
		return fmt.Errorf("create file: %w", err)
	}

	defer func() {
		if err != nil {
			_ = f.Close()
			_ = os.Remove(partPath)
		}
	}()

	if _, err := io.Copy(f, resp.Body); err != nil {
		_ = os.Remove(partPath)
		return fmt.Errorf("copy body into file: %w", err)
	}

	if err := f.Close(); err != nil {
		_ = os.Remove(partPath)
		return fmt.Errorf("close file: %w", err)
	}

	if err := os.Rename(partPath, outPath); err != nil {
		_ = os.Remove(partPath)
		return fmt.Errorf("rename: %w", err)
	}

	return nil
}

func isAllowedRetryFile(rel string) bool {
	if rel == "failed.jsonl" {
		return true
	}

	ext := strings.ToLower(filepath.Ext(rel))
	if !slices.Contains(validExts, ext) {
		return false
	}

	dir := filepath.Dir(rel)
	return dir == "." || dir == "thumbs"
}
