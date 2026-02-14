package validator

import (
	"fmt"
	"mime"
	"regexp"
	"strings"
)

const (
	minEmailLength    = 3
	maxEmailLength    = 255
	minPasswordLength = 8
	maxPasswordLength = 128
	maxProjectNameLen = 255
	maxFileNameLen    = 255
	maxAPIKeyNameLen  = 255
	maxContentTypeLen = 255
	maxFileSizeGB     = 100
	maxFileSizeBytes  = int64(100 * 1024 * 1024 * 1024)
	asciiControlStart = 32
	asciiDelete       = 127

	errEmailEmptyFmt             = "email cannot be empty"
	errEmailLengthFmt            = "email must be between %d and %d characters"
	errEmailInvalidFmt           = "invalid email format"
	errPasswordMinLengthFmt      = "password must be at least %d characters"
	errPasswordMaxLengthFmt      = "password must not exceed %d characters"
	errProjectNameEmptyFmt       = "project name cannot be empty"
	errProjectNameMaxLengthFmt   = "project name must not exceed %d characters"
	errFileNameEmptyFmt          = "file name cannot be empty"
	errFileNameMaxLengthFmt      = "file name must not exceed %d characters"
	errFileNamePathSepFmt        = "file name cannot contain path separators"
	errFileNameControlCharsFmt   = "file name cannot contain control characters"
	errAPIKeyNameEmptyFmt        = "API key name cannot be empty"
	errAPIKeyNameMaxLengthFmt    = "API key name must not exceed %d characters"
	errAPIKeyNameControlFmt      = "API key name cannot contain control characters"
	errContentTypeMaxLengthFmt   = "content type must not exceed %d characters"
	errContentTypeInvalidFmt     = "invalid content type"
	errFileSizeNegativeFmt       = "file size cannot be negative"
	errFileSizeMaxFmt            = "file size exceeds maximum of %dGB"
	errFolderNameEmptyFmt        = "folder name cannot be empty"
	errFolderNameMaxLengthFmt    = "folder name must not exceed %d characters"
	errFolderNamePathSepFmt      = "folder name cannot contain path separators"
	errFolderNameControlCharsFmt = "folder name cannot contain control characters"
	errFolderPathMaxLengthFmt    = "folder path must not exceed %d characters"
	errFolderPathBackslashFmt    = "folder path cannot contain backslashes"
	errFolderPathEmptySegFmt     = "folder path contains empty segment"
	errFolderPathTraversalFmt    = "folder path cannot contain path traversal"
	errFolderPathControlCharsFmt = "folder path cannot contain control characters"

	maxFolderPathLen = 1024
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

func Email(email string) error {
	if email == "" {
		return fmt.Errorf(errEmailEmptyFmt)
	}

	if len(email) < minEmailLength || len(email) > maxEmailLength {
		return fmt.Errorf(errEmailLengthFmt, minEmailLength, maxEmailLength)
	}

	if !emailRegex.MatchString(email) {
		return fmt.Errorf(errEmailInvalidFmt)
	}

	return nil
}

func Password(password string) error {
	if len(password) < minPasswordLength {
		return fmt.Errorf(errPasswordMinLengthFmt, minPasswordLength)
	}

	if len(password) > maxPasswordLength {
		return fmt.Errorf(errPasswordMaxLengthFmt, maxPasswordLength)
	}

	return nil
}

func ProjectName(name string) error {
	if name == "" {
		return fmt.Errorf(errProjectNameEmptyFmt)
	}

	if len(name) > maxProjectNameLen {
		return fmt.Errorf(errProjectNameMaxLengthFmt, maxProjectNameLen)
	}

	return nil
}

func FileName(name string) error {
	if name == "" {
		return fmt.Errorf(errFileNameEmptyFmt)
	}

	if len(name) > maxFileNameLen {
		return fmt.Errorf(errFileNameMaxLengthFmt, maxFileNameLen)
	}

	if strings.Contains(name, "..") || strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return fmt.Errorf(errFileNamePathSepFmt)
	}

	for _, char := range name {
		if char < asciiControlStart || char == asciiDelete {
			return fmt.Errorf(errFileNameControlCharsFmt)
		}
	}

	return nil
}

func APIKeyName(name string) error {
	if name == "" {
		return fmt.Errorf(errAPIKeyNameEmptyFmt)
	}

	if len(name) > maxAPIKeyNameLen {
		return fmt.Errorf(errAPIKeyNameMaxLengthFmt, maxAPIKeyNameLen)
	}

	for _, char := range name {
		if char < asciiControlStart || char == asciiDelete {
			return fmt.Errorf(errAPIKeyNameControlFmt)
		}
	}

	return nil
}

func FolderName(name string) error {
	if name == "" {
		return fmt.Errorf(errFolderNameEmptyFmt)
	}

	if len(name) > maxFileNameLen {
		return fmt.Errorf(errFolderNameMaxLengthFmt, maxFileNameLen)
	}

	if strings.Contains(name, "..") || strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return fmt.Errorf(errFolderNamePathSepFmt)
	}

	for _, char := range name {
		if char < asciiControlStart || char == asciiDelete {
			return fmt.Errorf(errFolderNameControlCharsFmt)
		}
	}

	return nil
}

func FolderPath(path string) error {
	if path == "" {
		return nil
	}

	if len(path) > maxFolderPathLen {
		return fmt.Errorf(errFolderPathMaxLengthFmt, maxFolderPathLen)
	}

	if strings.Contains(path, "\\") {
		return fmt.Errorf(errFolderPathBackslashFmt)
	}

	segments := strings.Split(strings.Trim(path, "/"), "/")
	for _, seg := range segments {
		if seg == "" {
			return fmt.Errorf(errFolderPathEmptySegFmt)
		}
		if seg == ".." || seg == "." {
			return fmt.Errorf(errFolderPathTraversalFmt)
		}
		for _, char := range seg {
			if char < asciiControlStart || char == asciiDelete {
				return fmt.Errorf(errFolderPathControlCharsFmt)
			}
		}
	}

	return nil
}

func FileSize(size int64) error {
	if size < 0 {
		return fmt.Errorf(errFileSizeNegativeFmt)
	}

	if size > maxFileSizeBytes {
		return fmt.Errorf(errFileSizeMaxFmt, maxFileSizeGB)
	}

	return nil
}

func ContentType(contentType string) error {
	if contentType == "" {
		return nil
	}

	if len(contentType) > maxContentTypeLen {
		return fmt.Errorf(errContentTypeMaxLengthFmt, maxContentTypeLen)
	}

	if _, _, err := mime.ParseMediaType(contentType); err != nil {
		return fmt.Errorf(errContentTypeInvalidFmt)
	}

	return nil
}
