package app

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestHealthcheckWithoutAuth(t *testing.T) {
	svc := newTestService(fakeValidator{}, &fakeRunner{})
	req := httptest.NewRequest(http.MethodGet, "/healthcheck", nil)
	rec := httptest.NewRecorder()

	svc.Routes().ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "OK", strings.TrimSpace(rec.Body.String()))
}

func TestRenderPDFUnauthorizedWhenMissingToken(t *testing.T) {
	svc := newTestService(fakeValidator{}, &fakeRunner{})
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte("")))
	req.Header.Set("Content-Type", "multipart/form-data")
	rec := httptest.NewRecorder()

	svc.Routes().ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestRenderPDFForbiddenWhenScopeMissing(t *testing.T) {
	svc := newTestService(fakeValidator{err: ErrForbidden}, &fakeRunner{})
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte("")))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "multipart/form-data")
	rec := httptest.NewRecorder()

	svc.Routes().ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestRenderPDFBadContentType(t *testing.T) {
	svc := newTestService(fakeValidator{}, &fakeRunner{})
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("{}"))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	svc.Routes().ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestRenderPDFRequiresHTML(t *testing.T) {
	svc := newTestService(fakeValidator{}, &fakeRunner{})
	body, contentType := newMultipartBody(t, map[string]string{"css": "body{color:red;}"}, nil)
	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()

	svc.Routes().ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "No html file provided")
}

func TestRenderPDFSuccess(t *testing.T) {
	runner := &fakeRunner{output: []byte("%PDF-1.4")}
	svc := newTestService(fakeValidator{}, runner)

	body, contentType := newMultipartBody(t,
		map[string]string{
			"html":       "<html><body><h1>Hello</h1><img src=\"logo.png\"></body></html>",
			"css":        "body{font-size:12pt;}",
			"asset.logo": "binary-image-content",
		},
		map[string]string{"attachment.test": "notes.txt"},
	)

	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()

	svc.Routes().ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code, "body: %q", rec.Body.String())
	assert.Equal(t, "application/pdf", rec.Header().Get("Content-Type"))
	assert.True(t, bytes.Equal(rec.Body.Bytes(), []byte("%PDF-1.4")), "unexpected PDF body: %q", rec.Body.String())
	assert.NotEmpty(t, runner.lastHTML)
	assert.NotEmpty(t, runner.lastCSS)
	assert.Len(t, runner.lastAttachments, 1)
}

func TestRenderPDFSupportsFilePrefixAttachments(t *testing.T) {
	runner := &fakeRunner{output: []byte("%PDF")}
	svc := newTestService(fakeValidator{}, runner)

	body, contentType := newMultipartBody(t,
		map[string]string{"html": "<html><body>ok</body></html>"},
		map[string]string{"file.terms": "terms.txt"},
	)
	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()

	svc.Routes().ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Len(t, runner.lastAttachments, 1)
}

func TestRenderPDFFailedRunnerReturns500(t *testing.T) {
	runner := &fakeRunner{runErr: errors.New("boom")}
	svc := newTestService(fakeValidator{}, runner)
	body, contentType := newMultipartBody(t, map[string]string{"html": "<html></html>"}, nil)
	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()

	svc.Routes().ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func newTestService(validator TokenValidator, runner PDFRunner) *Service {
	return NewService(
		validator,
		runner,
		Config{
			MaxRequestBytes: 5 * 1024 * 1024,
			RequestTimeout:  3 * time.Second,
		},
		NewMockObservabilityProvider(),
	)
}

func newMultipartBody(t *testing.T, fields map[string]string, files map[string]string) (*bytes.Buffer, string) {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	for name, value := range fields {
		fileWriter, err := writer.CreateFormFile(name, name+".txt")
		if err != nil {
			t.Fatalf("failed to create form file %s: %v", name, err)
		}
		if _, err := io.Copy(fileWriter, strings.NewReader(value)); err != nil {
			t.Fatalf("failed to write part %s: %v", name, err)
		}
	}

	for name, filename := range files {
		fileWriter, err := writer.CreateFormFile(name, filename)
		if err != nil {
			t.Fatalf("failed to create file part %s: %v", name, err)
		}
		if _, err := io.Copy(fileWriter, strings.NewReader("test-file")); err != nil {
			t.Fatalf("failed to write file part %s: %v", name, err)
		}
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("failed to close multipart writer: %v", err)
	}

	return body, writer.FormDataContentType()
}

type fakeValidator struct {
	err error
}

func (f fakeValidator) Validate(_ context.Context, _ string) error {
	if f.err != nil {
		return f.err
	}
	return nil
}

type fakeRunner struct {
	output          []byte
	runErr          error
	lastHTML        string
	lastCSS         string
	lastAttachments []string
}

func (f *fakeRunner) GeneratePDF(_ context.Context, workDir string, htmlFilename string, cssFilename string, attachmentFilenames []string, output io.Writer) error {
	f.lastHTML = htmlFilename
	f.lastCSS = cssFilename
	f.lastAttachments = attachmentFilenames
	if f.runErr != nil {
		return f.runErr
	}
	content := f.output
	if len(content) == 0 {
		content = []byte("%PDF")
	}
	_, err := output.Write(content)
	return err
}
