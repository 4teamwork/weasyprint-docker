package app

import (
	"context"
	"errors"
	"io"
	"mime"
	"net/http"
	"strings"
	"time"
)

type Config struct {
	MaxRequestBytes int64
	MaxPartBytes    int64
	RequestTimeout  time.Duration
}

type TokenValidator interface {
	Validate(ctx context.Context, token string) error
}

type PDFRunner interface {
	GeneratePDF(ctx context.Context, workDir string, htmlFilename string, cssFilename string, attachmentFilenames []string, outputFilename string) error
}

type Service struct {
	validator TokenValidator
	runner    PDFRunner
	config    Config
}

func NewService(validator TokenValidator, runner PDFRunner, config Config) *Service {
	return &Service{
		validator: validator,
		runner:    runner,
		config:    config,
	}
}

func (s *Service) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthcheck", s.healthcheck)
	mux.Handle("/", s.requireAuth(http.HandlerFunc(s.renderPDF)))
	return mux
}

func (s *Service) healthcheck(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}

func (s *Service) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, err := parseBearerToken(r.Header.Get("Authorization"))
		if err != nil {
			writeHTTPError(w, r, NewUnauthorizedError("Unauthorized", err))
			return
		}

		err = s.validator.Validate(r.Context(), token)
		if err != nil {
			if errors.Is(err, ErrForbidden) {
				writeHTTPError(w, r, NewForbiddenError("Forbidden", err))
				return
			}
			writeHTTPError(w, r, NewUnauthorizedError("Unauthorized", err))
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Service) renderPDF(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeHTTPError(w, r, NewMethodNotAllowedError("Method not allowed", nil))
		return
	}

	contentType := r.Header.Get("Content-Type")
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil || mediaType != "multipart/form-data" {
		writeHTTPError(w, r, NewBadRequestError("Multipart request required.", err))
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, s.config.MaxRequestBytes)
	reader, err := r.MultipartReader()
	if err != nil {
		writeHTTPError(w, r, NewBadRequestError("Multipart request required.", err))
		return
	}

	artifact, err := s.generatePDFArtifact(r.Context(), reader)
	if err != nil {
		writeHTTPError(w, r, err)
		return
	}
	defer artifact.Cleanup()

	outputFile, err := artifact.Open()
	if err != nil {
		writeHTTPError(w, r, err)
		return
	}
	defer outputFile.Close()

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", `attachment; filename="`+artifact.Filename+`"`)
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, outputFile)
}

func parseBearerToken(headerValue string) (string, error) {
	if headerValue == "" {
		return "", errors.New("missing authorization")
	}
	parts := strings.SplitN(headerValue, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", errors.New("invalid authorization scheme")
	}
	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", errors.New("missing token")
	}
	return token, nil
}

type WeasyprintRunner struct {
	BwrapPath      string
	WeasyprintPath string
}
