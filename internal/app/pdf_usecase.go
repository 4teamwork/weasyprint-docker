package app

import (
	"context"
	"io"
	"mime/multipart"
	"os"
	"strings"
)

const defaultStylesheetPath = "/defaults/default.css"

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

	pp := &PartProcessor{
		reader: reader,
		root:   root,
	}
	if err := pp.ProcessParts(); err != nil {
		return err
	}

	renderCtx, cancel := context.WithTimeout(ctx, s.config.RequestTimeout)
	defer cancel()

	if err := s.runner.GeneratePDF(renderCtx, workDir, pp.htmlFilename, pp.cssFilename, pp.attachmentFilenames, writer); err != nil {
		return NewInternalError("PDF generation failed.", err)
	}

	return nil
}

type PartProcessor struct {
	reader              *multipart.Reader
	root                *os.Root
	htmlFilename        string
	cssFilename         string
	attachmentFilenames []string
}

func (p *PartProcessor) ProcessParts() error {
	for {
		err := p.ProcessPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}

	if p.htmlFilename == "" {
		return NewBadRequestError("No html file provided.", nil)
	}

	if p.cssFilename == "" {
		p.cssFilename = defaultStylesheetPath
	}

	return nil
}

func (p *PartProcessor) ProcessPart() error {
	part, err := p.reader.NextPart()
	if err != nil {
		return err
	}
	defer part.Close()

	saveErr := p.savePart(part)
	if saveErr != nil {
		return NewInternalError("Failed to process request.", saveErr)
	}

	switch {
	case part.FormName() == "html":
		p.htmlFilename = part.FileName()
	case part.FormName() == "css":
		p.cssFilename = part.FileName()
	case strings.HasPrefix(part.FormName(), "attachment."), strings.HasPrefix(part.FormName(), "file."):
		p.attachmentFilenames = append(p.attachmentFilenames, part.FileName())
	}
	return nil
}

func (p *PartProcessor) savePart(part *multipart.Part) error {
	file, err := p.root.OpenFile(part.FileName(), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, part)
	return err
}
