package apperror

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrInvalidJSON       = errors.New("invalid json")   // 422
	ErrBadRequest        = errors.New("bad request")    // 400
	ErrNotFound          = errors.New("not found")      // 404
	ErrBackend           = errors.New("gateway error")  // 502
	ErrAlreadyExists     = errors.New("already exists") // 409
	ErrHashInUse         = errors.New("hash is still referenced in DB")
	ErrUnauthorized      = errors.New("unauthorized") // 401
	ErrValuesWithoutHost = errors.New("no host but has values")
	ErrNoImgsAfterCursor = errors.New("no imgs after cursor")
	ErrEmptyCSV          = errors.New("empty csv")
	ErrUniqueEntity      = errors.New("unique entity")
	ErrUnsupportedExt    = errors.New("unsupported extension")
	ErrBadFilename       = errors.New("bad filename")
)

type Warning struct {
	Msg string `json:"msg"`
	Err error  `json:"err"`
}

type RemoveImgsError struct {
	Items []Warning
}

type DownloadWarnings struct {
	Items []Warning
}

func (e *DownloadWarnings) Error() string {
	if len(e.Items) == 0 || e == nil {
		return "batch error empty"
	}
	var b strings.Builder
	for _, it := range e.Items {
		fmt.Fprintf(&b, "- %s : %v \n", it.Msg, it.Err)
	}
	return b.String()
}

func (e *RemoveImgsError) Error() string {
	if len(e.Items) == 0 || e == nil {
		return "batch error empty"
	}
	var b strings.Builder
	for _, it := range e.Items {
		fmt.Fprintf(&b, " - %s : %v \n", it.Msg, it.Err)
	}
	return b.String()
}

func NewDownloadError(items []Warning) error {
	if len(items) == 0 {
		return nil
	}
	return &DownloadWarnings{Items: items}
}

func NewRemoveImgsError(items []Warning) error {
	if len(items) == 0 {
		return nil
	}
	return &RemoveImgsError{Items: items}
}

type DownloadIssue string

const (
	IssueUnsupportedExt  DownloadIssue = "unsupported_ext"
	IssueBadFilename     DownloadIssue = "bad filename"
	IssueUnknown         DownloadIssue = "unknown"
	IssueGetRequestError DownloadIssue = "Get Request Error"
	IssueCloseBody       DownloadIssue = "Close body error"
	IssueCreatePic       DownloadIssue = "Create Pic error"
	IssueFileStatError   DownloadIssue = "File stat error"
)
