package app

import (
	"context"
	"errors"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
)

const defaultPageCSS = "@page { size: A4; margin: 2cm 2.5cm; }"

var errPartTooLarge = errors.New("uploaded file too large")

func (s *Service) generatePDFToWriter(ctx context.Context, reader *multipart.Reader, writer io.Writer) error {
	workDir, err := os.MkdirTemp("", "pdf-service-*")
	if err != nil {
		return NewInternalError("Failed to process request.", err)
	}
	defer func() { _ = os.RemoveAll(workDir) }()

	root, err := os.OpenRoot(workDir)
	if err != nil {
		return NewInternalError("Failed to process request.", err)
	}
	defer root.Close()

	htmlFilename, cssFilename, attachmentFilenames, err := s.persistParts(reader, root)
	if err != nil {
		return err
	}

	if htmlFilename == "" {
		return NewBadRequestError("No html file provided.", nil)
	}

	if cssFilename == "" {
		cssFilename = "default.css"
		cssFile, writeErr := root.OpenFile(cssFilename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
		if writeErr != nil {
			return NewInternalError("Failed to process request.", writeErr)
		}
		if _, writeErr = cssFile.Write([]byte(defaultPageCSS)); writeErr != nil {
			cssFile.Close()
			return NewInternalError("Failed to process request.", writeErr)
		}
		if closeErr := cssFile.Close(); closeErr != nil {
			return NewInternalError("Failed to process request.", closeErr)
		}
	}

	renderCtx, cancel := context.WithTimeout(ctx, s.config.RequestTimeout)
	defer cancel()

	if err := s.runner.GeneratePDF(renderCtx, workDir, htmlFilename, cssFilename, attachmentFilenames, writer); err != nil {
		return NewInternalError("PDF generation failed.", err)
	}

	return nil
}

func (s *Service) persistParts(reader *multipart.Reader, root *os.Root) (string, string, []string, error) {
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

		html, css, attachment, processErr := s.processPart(part, root)
		if processErr != nil {
			return "", "", nil, processErr
		}

		if html != "" {
			htmlFilename = html
		}
		if css != "" {
			cssFilename = css
		}
		if attachment != "" {
			attachmentFilenames = append(attachmentFilenames, attachment)
		}
	}

	return htmlFilename, cssFilename, attachmentFilenames, nil
}

func (s *Service) processPart(part *multipart.Part, root *os.Root) (string, string, string, error) {
	defer part.Close()

	fieldName := part.FormName()
	if fieldName == "" {
		return "", "", "", nil
	}

	if !isAllowedField(fieldName) {
		_, _ = io.Copy(io.Discard, part)
		return "", "", "", nil
	}

	filename := safeFilename(part.FileName(), fieldName)
	if filename == "" {
		return "", "", "", nil
	}

	savedFilename, saveErr := savePart(root, filename, part, s.config.MaxPartBytes)
	if saveErr != nil {
		if errors.Is(saveErr, errPartTooLarge) {
			return "", "", "", NewRequestTooLargeError("Uploaded file too large.", saveErr)
		}
		return "", "", "", NewInternalError("Failed to process request.", saveErr)
	}

	if fieldName == "html" {
		return savedFilename, "", "", nil
	}
	if fieldName == "css" {
		return "", savedFilename, "", nil
	}
	if strings.HasPrefix(fieldName, "attachment.") || strings.HasPrefix(fieldName, "file.") {
		return "", "", savedFilename, nil
	}

	return "", "", "", nil
}

func isAllowedField(field string) bool {
	return field == "html" || field == "css" || strings.HasPrefix(field, "attachment.") || strings.HasPrefix(field, "asset.") || strings.HasPrefix(field, "file.")
}

func safeFilename(originalFilename string, fieldName string) string {
	trimmed := strings.TrimSpace(originalFilename)
	if trimmed == "" {
		return fieldName
	}
	base := filepath.Base(trimmed)
	if base == "." {
		return fieldName
	}
	return base
}

func savePart(root *os.Root, filename string, part *multipart.Part, maxPartBytes int64) (string, error) {
	file, err := root.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return "", err
	}
	defer file.Close()

	read := &io.LimitedReader{R: part, N: maxPartBytes + 1}
	written, err := io.Copy(file, read)
	if err != nil {
		return "", err
	}
	if written > maxPartBytes {
		return "", errPartTooLarge
	}

	return filename, nil
}
