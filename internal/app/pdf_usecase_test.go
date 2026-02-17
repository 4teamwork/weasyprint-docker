package app

import (
	"bytes"
	"context"
	"errors"
	"mime"
	"mime/multipart"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGeneratePDFToWriterRequiresHTMLPart(t *testing.T) {
	runner := &fakeRunner{}
	svc := newTestService(fakeValidator{}, runner)

	body, contentType := newMultipartBody(t, map[string]string{"css": "body{color:red;}"}, nil)
	reader := newMultipartReaderFromBody(t, body, contentType)

	err := svc.generatePDFToWriter(context.Background(), reader, &bytes.Buffer{})

	var appErr *AppError
	assert.Error(t, err)
	assert.ErrorAs(t, err, &appErr)
	assert.Equal(t, 400, appErr.StatusCode)
	assert.Equal(t, "No html file provided.", appErr.Message)
}

func TestGeneratePDFToWriterUsesDefaultStylesheetWhenCssMissing(t *testing.T) {
	runner := &fakeRunner{}
	svc := newTestService(fakeValidator{}, runner)

	body, contentType := newMultipartBody(t, map[string]string{"html": "<html><body>ok</body></html>"}, nil)
	reader := newMultipartReaderFromBody(t, body, contentType)

	err := svc.generatePDFToWriter(context.Background(), reader, &bytes.Buffer{})

	assert.NoError(t, err)
	assert.Equal(t, defaultStylesheetPath, runner.lastCSS)
	assert.NotEmpty(t, runner.lastHTML)
}

func TestGeneratePDFToWriterForwardsAttachmentAndFileParts(t *testing.T) {
	runner := &fakeRunner{}
	svc := newTestService(fakeValidator{}, runner)

	body, contentType := newMultipartBody(
		t,
		map[string]string{"html": "<html><body>ok</body></html>"},
		map[string]string{
			"attachment.invoice": "invoice.pdf",
			"file.terms":         "terms.txt",
		},
	)
	reader := newMultipartReaderFromBody(t, body, contentType)

	err := svc.generatePDFToWriter(context.Background(), reader, &bytes.Buffer{})

	assert.NoError(t, err)
	assert.Len(t, runner.lastAttachments, 2)
	assert.ElementsMatch(t, []string{"invoice.pdf", "terms.txt"}, runner.lastAttachments)
}

func TestGeneratePDFToWriterMapsRunnerErrorToInternalError(t *testing.T) {
	runner := &fakeRunner{runErr: errors.New("boom")}
	svc := newTestService(fakeValidator{}, runner)

	body, contentType := newMultipartBody(t, map[string]string{"html": "<html><body>ok</body></html>"}, nil)
	reader := newMultipartReaderFromBody(t, body, contentType)

	err := svc.generatePDFToWriter(context.Background(), reader, &bytes.Buffer{})

	var appErr *AppError
	assert.Error(t, err)
	assert.ErrorAs(t, err, &appErr)
	assert.Equal(t, 500, appErr.StatusCode)
	assert.Equal(t, "PDF generation failed.", appErr.Message)
	assert.EqualError(t, appErr.Cause, "boom")
}

func newMultipartReaderFromBody(t *testing.T, body *bytes.Buffer, contentType string) *multipart.Reader {
	t.Helper()
	_, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		t.Fatalf("failed to parse media type: %v", err)
	}
	return multipart.NewReader(body, params["boundary"])
}
