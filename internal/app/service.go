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
	RequestTimeout  time.Duration
}

type TokenValidator interface {
	Validate(ctx context.Context, token string) error
}

type PDFRunner interface {
	GeneratePDF(ctx context.Context, workDir string, htmlFilename string, cssFilename string, attachmentFilenames []string, output io.Writer) error
}

type Service struct {
	validator TokenValidator
	runner    PDFRunner
	config    Config
	obs       Observability
}

func NewService(validator TokenValidator, runner PDFRunner, config Config, obs Observability) *Service {
	return &Service{
		validator: validator,
		runner:    runner,
		config:    config,
		obs:       obs,
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
			writeHTTPError(s.obs.Logger(), w, r, NewUnauthorizedError("Unauthorized", err))
			return
		}

		err = s.validator.Validate(r.Context(), token)
		if err != nil {
			if errors.Is(err, ErrForbidden) {
				writeHTTPError(s.obs.Logger(), w, r, NewForbiddenError("Forbidden", err))
				return
			}
			writeHTTPError(s.obs.Logger(), w, r, NewUnauthorizedError("Unauthorized", err))
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Service) renderPDF(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeHTTPError(s.obs.Logger(), w, r, NewMethodNotAllowedError("Method not allowed", nil))
		return
	}

	contentType := r.Header.Get("Content-Type")
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil || mediaType != "multipart/form-data" {
		writeHTTPError(s.obs.Logger(), w, r, NewBadRequestError("Multipart request required.", err))
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, s.config.MaxRequestBytes)
	reader, err := r.MultipartReader()
	if err != nil {
		writeHTTPError(s.obs.Logger(), w, r, NewBadRequestError("Multipart request required.", err))
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", `attachment; filename="output.pdf"`)

	err = s.generatePDFToWriter(r.Context(), reader, w)
	if err != nil {
		writeHTTPError(s.obs.Logger(), w, r, err)
		return
	}
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
	BwrapPath             string
	WeasyprintPath        string
	DefaultStylesheetPath string
}
