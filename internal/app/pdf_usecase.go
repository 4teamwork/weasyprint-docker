package app

import (
	"context"
	"errors"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const defaultPageCSS = "@page { size: A4; margin: 2cm 2.5cm; }"

var errPartTooLarge = errors.New("uploaded file too large")

type PDFArtifact struct {
	Path     string
	Filename string
	cleanup  func()
}

func (a PDFArtifact) Cleanup() {
	if a.cleanup != nil {
		a.cleanup()
	}
}

func (a PDFArtifact) Open() (*os.File, error) {
	file, err := os.Open(a.Path)
	if err != nil {
		return nil, NewInternalError("PDF generation failed.", err)
	}
	return file, nil
}

func (s *Service) generatePDFArtifact(ctx context.Context, reader *multipart.Reader) (PDFArtifact, error) {
	workDir, err := os.MkdirTemp("", "pdf-service-*")
	if err != nil {
		return PDFArtifact{}, NewInternalError("Failed to process request.", err)
	}

	cleanup := func() {
		_ = os.RemoveAll(workDir)
	}

	htmlFilename, cssFilename, attachmentFilenames, err := s.persistParts(reader, workDir)
	if err != nil {
		cleanup()
		return PDFArtifact{}, err
	}

	if htmlFilename == "" {
		cleanup()
		return PDFArtifact{}, NewBadRequestError("No html file provided.", nil)
	}

	if cssFilename == "" {
		cssFilename = "default.css"
		defaultCSSPath := filepath.Join(workDir, cssFilename)
		if writeErr := os.WriteFile(defaultCSSPath, []byte(defaultPageCSS), 0o600); writeErr != nil {
			cleanup()
			return PDFArtifact{}, NewInternalError("Failed to process request.", writeErr)
		}
	}

	outputFilename := "output.pdf"
	renderCtx, cancel := context.WithTimeout(ctx, s.config.RequestTimeout)
	defer cancel()

	if err := s.runner.GeneratePDF(renderCtx, workDir, htmlFilename, cssFilename, attachmentFilenames, outputFilename); err != nil {
		cleanup()
		return PDFArtifact{}, NewInternalError("PDF generation failed.", err)
	}

	outputPath := filepath.Join(workDir, outputFilename)
	if _, err := os.Stat(outputPath); err != nil {
		cleanup()
		return PDFArtifact{}, NewInternalError("PDF generation failed.", err)
	}

	return PDFArtifact{
		Path:     outputPath,
		Filename: outputFilename,
		cleanup:  cleanup,
	}, nil
}

func (s *Service) persistParts(reader *multipart.Reader, workDir string) (string, string, []string, error) {
	htmlFilename := ""
	cssFilename := ""
	attachmentFilenames := make([]string, 0)

	for {
		part, err := reader.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return "", "", nil, NewBadRequestError("Invalid multipart data.", err)
		}

		fieldName := part.FormName()
		if fieldName == "" {
			_ = part.Close()
			continue
		}

		if !isAllowedField(fieldName) {
			_, _ = io.Copy(io.Discard, part)
			_ = part.Close()
			continue
		}

		filename := safeFilename(part.FileName(), fieldName)
		if filename == "" {
			_ = part.Close()
			continue
		}

		savedFilename, saveErr := savePart(workDir, filename, part, s.config.MaxPartBytes)
		if saveErr != nil {
			if errors.Is(saveErr, errPartTooLarge) {
				return "", "", nil, NewRequestTooLargeError("Uploaded file too large.", saveErr)
			}
			return "", "", nil, NewInternalError("Failed to process request.", saveErr)
		}

		switch {
		case fieldName == "html":
			htmlFilename = savedFilename
		case fieldName == "css":
			cssFilename = savedFilename
		case strings.HasPrefix(fieldName, "attachment."), strings.HasPrefix(fieldName, "file."):
			attachmentFilenames = append(attachmentFilenames, savedFilename)
		}
	}

	return htmlFilename, cssFilename, attachmentFilenames, nil
}

func isAllowedField(field string) bool {
	return field == "html" || field == "css" || strings.HasPrefix(field, "attachment.") || strings.HasPrefix(field, "asset.") || strings.HasPrefix(field, "file.")
}

func safeFilename(originalFilename string, fieldName string) string {
	trimmed := strings.TrimSpace(originalFilename)
	if trimmed == "" {
		trimmed = fieldName
	}
	base := filepath.Base(trimmed)
	base = strings.ReplaceAll(base, string(filepath.Separator), "_")
	base = strings.ReplaceAll(base, "/", "_")
	base = strings.ReplaceAll(base, "\\", "_")
	base = strings.TrimSpace(base)
	if base == "" || base == "." || base == ".." {
		return ""
	}
	return base
}

func savePart(workDir string, filename string, part *multipart.Part, maxPartBytes int64) (string, error) {
	resolvedFilename := resolveUniqueFilename(workDir, filename)
	fullPath := filepath.Join(workDir, resolvedFilename)
	file, err := os.OpenFile(fullPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return "", err
	}
	defer file.Close()

	read := &io.LimitedReader{R: part, N: maxPartBytes + 1}
	written, err := io.Copy(file, read)
	_ = part.Close()
	if err != nil {
		return "", err
	}
	if written > maxPartBytes {
		return "", errPartTooLarge
	}

	return resolvedFilename, nil
}

func resolveUniqueFilename(workDir string, candidate string) string {
	if _, err := os.Stat(filepath.Join(workDir, candidate)); errors.Is(err, os.ErrNotExist) {
		return candidate
	}

	ext := filepath.Ext(candidate)
	base := strings.TrimSuffix(candidate, ext)
	for index := 2; index < 10000; index++ {
		resolved := base + "_" + strconv.Itoa(index) + ext
		if _, err := os.Stat(filepath.Join(workDir, resolved)); errors.Is(err, os.ErrNotExist) {
			return resolved
		}
	}

	return strconv.FormatInt(time.Now().UnixNano(), 10) + "_" + candidate
}
