package download

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"scraper/internal/pkg/apperror"
	"scraper/internal/pkg/kit"
	"scraper/internal/repository/scraper"
	imagerowsvc "scraper/internal/service/imageRow"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

type ImportIssue string

const (
	IssueEmptyReq          ImportIssue = "invalid req (empty)"
	IssueInsideDownloadDir ImportIssue = "child of download dir"
	IssueFilepathIsNotAbs  ImportIssue = "Path is not absolute"
	IssueFindUniqueFiles   ImportIssue = "Find unique pics error"
	IssueNoUniqueFiles     ImportIssue = "No unique files in dir"
	IssueInsideRoot        ImportIssue = "Path is inside root"
	IssueEntryExistDirNot  ImportIssue = "Entry exists while dir is not"
	IssueDirExistEntryNot  ImportIssue = "Dir exists while entry is not"
	IssueNotADir           ImportIssue = "Path is not a dir"
	IssueDontExist         ImportIssue = "Path don't exist"
	IssueMakeDirError      ImportIssue = "Make dir error"
	IssueCopyFileError     ImportIssue = "Copy file error"
	IssueMoveFileError     ImportIssue = "Move file error"
	IssueRemoveDirError    ImportIssue = "RemoveDirError"
	IssueProcessFiles      ImportIssue = "Process files error"
	IssueComputeDestError  ImportIssue = "ComputeDestError"
)

type FailedImport struct {
	Warn ImportIssue
	Path string
}

func (s *Service) ImportLocalDirs(ctx context.Context) ([]FailedImport, error) {
	fmt.Println("PATH:", s.paths.ImportDir)
	entries, err := os.ReadDir(s.paths.ImportDir)
	if err != nil {
		return nil, fmt.Errorf("ReadDir: %w", err)
	}

	paths := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		dirPath := filepath.Join(s.paths.ImportDir, entry.Name())
		paths = append(paths, dirPath)
	}

	if len(paths) == 0 {
		return nil, fmt.Errorf("paths: %w", apperror.ErrNotFound)
	}

	req := ParseImportDirs{
		Paths: paths,
	}

	fails := s.parseImportDirs(ctx, req)

	return fails, nil
}

func (s *Service) parseImportDirs(ctx context.Context, req ParseImportDirs) []FailedImport {
	fails := make([]FailedImport, 0)
	if err := kit.ValidateStruct(s.validate, req); err != nil {
		fails = append(fails,
			handleImportIssue(IssueEmptyReq, "validate", "ImportLocalDirs", "", err))
		return fails
	}
	for _, path := range req.Paths {
		info, err := os.Stat(path)
		if err != nil {
			fails = append(fails,
				handleImportIssue(IssueDontExist, "stat", "ImportLocalDirs", path, err))
			continue
		}

		if !info.IsDir() {
			fails = append(fails,
				handleImportIssue(IssueNotADir, "not a dir", "ImportLocalDirs", path, err))
			continue
		}

		files, err := s.findAndSplitFiles(ctx, path)
		if err != nil {
			fails = append(fails,
				handleImportIssue(IssueFindUniqueFiles, "findAndSplitFiles", "ImportLocalDirs", path, err))
			continue
		}

		if len(files.UniqueFiles) == 0 {
			fails = append(fails,
				handleImportIssue(IssueNoUniqueFiles, "no unique files", "ImportLocalDirs", path, err))
			continue
		}

		if err := s.isInsideDownloadDir(path); err != nil {
			fails = append(fails,
				handleImportIssue(IssueInsideRoot, "isInsideDownloadDir", "ImportLocalDirs", path, err))
			continue
		}

		dest, err := s.computeDestDir(ctx, path)
		if err != nil {
			fails = append(fails,
				handleImportIssue(dest.Issue, "computeDestDir", "ImportLocalDirs", path, err))
			continue
		}

		if err := os.Mkdir(dest.Path, 0766); err != nil {
			fails = append(fails,
				handleImportIssue(IssueMakeDirError, "mkdir", "ImportLocalDirs", path, err))
			continue
		}

		if err := s.copyFiles(path, dest.Path, files.UniqueFiles); err != nil {
			fails = append(fails,
				handleImportIssue(IssueCopyFileError, "copyFiles", "ImportLocalDirs", path, err))
			if !dest.IsUpdate {
				if err := os.RemoveAll(dest.Path); err != nil {
					log.Err(fmt.Errorf("RemoveAll: %w", err)).Msg("ImportLocalDir")
				}
			}
			continue
		}

		processReq := scraper.ProcessFilesReq{
			ModelPath:    filepath.Join(s.paths.ModelDir, s.paths.ModelName),
			DownloadPath: dest.Path,
		}
		if err := s.scraperRepo.ProcessFiles(ctx, processReq); err != nil {
			fails = append(fails,
				handleImportIssue(IssueProcessFiles, "processFiles", "ImportLocalDirs", path, err))
			if err := os.RemoveAll(dest.Path); err != nil {
				log.Err(fmt.Errorf("RemoveAll: %w", err)).Msg("ImportLocalDir")
			}

			continue
		}

		csvPath := filepath.Join(dest.Path, "output.csv")
		csvRows, err := s.scraperRepo.CSVToRows(csvPath)
		if err != nil {
			fails = append(fails,
				handleImportIssue(IssueProcessFiles, "CSVToRows", "ImportLocalDirs", path, err))
			if err := os.RemoveAll(dest.Path); err != nil {
				log.Err(fmt.Errorf("RemoveAll: %w", err)).Msg("ImportLocalDir")
			}
			continue
		}

		if dest.IsUpdate {
			if err := s.scraperRepo.MoveRetryDirToParent(dest.Path); err != nil {
				fails = append(fails,
					handleImportIssue(IssueMoveFileError, "moveRetryDirToParent", "ImportLocalDirs", path, err))
				continue
			}
			if err := s.writeToExistingBatch(ctx, dest.BatchID, csvRows); err != nil {
				fails = append(fails,
					handleImportIssue(IssueProcessFiles, "writeToExistingBatch", "ImportLocalDirs", path, err))
				continue
			}

			if err := os.RemoveAll(dest.Path); err != nil {
				log.Err(fmt.Errorf("RemoveAll: %w", err)).Msg("ImportLocalDir")
			}

		}

		if _, err := s.writeToNewBatch(ctx, filepath.Base(dest.Path), csvRows); err != nil {
			fails = append(fails,
				handleImportIssue(IssueProcessFiles, "writeToNewBatch", "ImportLocalDirs", path, err))
			if err := os.RemoveAll(dest.Path); err != nil {
				log.Err(fmt.Errorf("RemoveAll: %w", err)).Msg("ImportLocalDir")
			}
			continue
		}

	}

	if len(fails) != 0 {
		return fails
	}

	return nil
}

func handleImportIssue(issue ImportIssue, errDesc, msg, path string, err error) FailedImport {
	if err != nil {
		log.Error().Err(fmt.Errorf("%s: %w", errDesc, err)).Msg(msg)
	} else {
		log.Error().Msgf("%s: %s", msg, errDesc)
	}
	fail := FailedImport{
		Warn: issue,
		Path: path,
	}

	return fail
}

func (s *Service) isInsideDownloadDir(path string) error {
	cleanPath := filepath.Clean(path)
	cleanDlnPath := filepath.Clean(s.paths.DownloadDir)
	rel, err := filepath.Rel(cleanDlnPath, cleanPath)
	if err != nil {
		return fmt.Errorf("rel: %w", err)
	}

	if rel == "." || !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("path is inside root: %s", rel)
	}

	return nil
}

func (s *Service) computeDestDir(ctx context.Context, path string) (computeDestPathResp, error) {
	batchName := filepath.Base(path)
	testDir := filepath.Join(s.paths.DownloadDir, batchName)
	getReq := imagerowsvc.GetBatchReq{
		Name: batchName,
	}
	var resp computeDestPathResp
	respBatch, err := s.imageSVC.GetBatch(ctx, getReq)
	switch {
	case err == nil:
		info, err := os.Stat(testDir)
		if err != nil {
			resp.Issue = IssueEntryExistDirNot
			return resp, fmt.Errorf("stat: %w", err)
		}
		if !info.IsDir() {
			resp.Issue = IssueEntryExistDirNot
			return resp, fmt.Errorf("batch path is not dir: %s", testDir)
		}
		dirName := "_retry" + time.Now().Format("2006-01-02_15-04-05")
		resp.Path = filepath.Join(s.paths.DownloadDir, batchName, dirName)
		resp.IsUpdate = true

	case errors.Is(err, apperror.ErrNotFound):
		_, err := os.Stat(testDir)
		if err == nil {
			resp.Issue = IssueDirExistEntryNot
			return resp, fmt.Errorf("dir exists without db entry: %s", testDir)
		}
		if !errors.Is(err, os.ErrNotExist) {
			resp.Issue = IssueComputeDestError
			return resp, fmt.Errorf("stat: %w", err)
		}
		resp.Path = filepath.Join(s.paths.DownloadDir, batchName)

	default:
		resp.Issue = IssueComputeDestError
		return resp, fmt.Errorf("get batch: %w", err)
	}
	resp.BatchID = respBatch.ID
	return resp, nil
}

func (s *Service) copyFiles(srcDir, destDir string, uniqueFiles []scraper.File) error {
	for _, file := range uniqueFiles {
		srcPath := filepath.Join(srcDir, file.Path)
		destPath := filepath.Join(destDir, file.Path)
		if err := s.scraperRepo.CopyFile(srcPath, destPath); err != nil {
			return fmt.Errorf("CopyFile: %w", err)
		}
	}
	return nil
}
