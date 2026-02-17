package app

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
)

const defaultPageCSS = "@page { size: A4; margin: 2cm 2.5cm; }"

type Config struct {
	RequiredScope   string
	MaxRequestBytes int64
	MaxPartBytes    int64
	RequestTimeout  time.Duration
	BwrapPath       string
	WeasyprintPath  string
}

type TokenClaims struct {
	Scopes []string
}

type TokenValidator interface {
	Validate(ctx context.Context, token string) (TokenClaims, error)
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
			http.Error(w, "Unauthorized1", http.StatusUnauthorized)
			return
		}

		claims, err := s.validator.Validate(r.Context(), token)
		if err != nil {
			http.Error(w, "Unauthorized2: "+err.Error(), http.StatusUnauthorized)
			return
		}

		if !hasScope(claims.Scopes, s.config.RequiredScope) {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Service) renderPDF(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	contentType := r.Header.Get("Content-Type")
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil || mediaType != "multipart/form-data" {
		http.Error(w, "Multipart request required.", http.StatusBadRequest)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, s.config.MaxRequestBytes)
	reader, err := r.MultipartReader()
	if err != nil {
		http.Error(w, "Multipart request required.", http.StatusBadRequest)
		return
	}

	workDir, err := os.MkdirTemp("", "pdf-service-*")
	if err != nil {
		http.Error(w, "Failed to process request.", http.StatusInternalServerError)
		return
	}
	defer os.RemoveAll(workDir)

	htmlFilename, cssFilename, attachmentFilenames, err := s.persistParts(reader, workDir)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, errPartTooLarge) {
			status = http.StatusRequestEntityTooLarge
		}
		http.Error(w, err.Error(), status)
		return
	}

	if htmlFilename == "" {
		http.Error(w, "No html file provided.", http.StatusBadRequest)
		return
	}

	if cssFilename == "" {
		cssFilename = "default.css"
		defaultCSSPath := filepath.Join(workDir, cssFilename)
		if writeErr := os.WriteFile(defaultCSSPath, []byte(defaultPageCSS), 0o600); writeErr != nil {
			http.Error(w, "Failed to process request.", http.StatusInternalServerError)
			return
		}
	}

	outputFilename := "output.pdf"
	ctx, cancel := context.WithTimeout(r.Context(), s.config.RequestTimeout)
	defer cancel()

	if err := s.runner.GeneratePDF(ctx, workDir, htmlFilename, cssFilename, attachmentFilenames, outputFilename); err != nil {
		fmt.Println("PDF generation error:", err)
		http.Error(w, "PDF generation failed.", http.StatusInternalServerError)
		return
	}

	outputPath := filepath.Join(workDir, outputFilename)
	outputFile, err := os.Open(outputPath)
	if err != nil {
		fmt.Println("PDF generation error:", err)
		http.Error(w, "PDF generation failed.", http.StatusInternalServerError)
		return
	}
	defer outputFile.Close()

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", `attachment; filename="output.pdf"`)
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, outputFile)
}

var errPartTooLarge = errors.New("Uploaded file too large.")

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
			return "", "", nil, fmt.Errorf("Invalid multipart data.")
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
				return "", "", nil, errPartTooLarge
			}
			return "", "", nil, fmt.Errorf("Failed to process request.")
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

func hasScope(scopes []string, requiredScope string) bool {
	if requiredScope == "" {
		return true
	}
	return slices.Contains(scopes, requiredScope)
}

type WeasyprintRunner struct {
	BwrapPath      string
	WeasyprintPath string
}

func (r WeasyprintRunner) GeneratePDF(ctx context.Context, workDir string, htmlFilename string, cssFilename string, attachmentFilenames []string, outputFilename string) error {
	args := []string{
		"--unshare-all",
		"--unshare-net",
		"--die-with-parent",
		"--new-session",
		"--clearenv",
		"--ro-bind", "/usr", "/usr",
		"--ro-bind", "/bin", "/bin",
		"--ro-bind", "/lib", "/lib",
		"--ro-bind", "/lib64", "/lib64",
		"--ro-bind", "/sbin", "/sbin",
		"--ro-bind", "/etc", "/etc",
		"--tmpfs", "/tmp",
		"--tmpfs", "/run",
		"--bind", workDir, "/workspace",
		"--chdir", "/workspace",
		"--setenv", "HOME", "/tmp",
		"--setenv", "TMPDIR", "/tmp",
		"--setenv", "PATH", "/usr/sbin:/usr/bin:/sbin:/bin",
		"--",
		r.WeasyprintPath,
		htmlFilename,
		outputFilename,
		"--stylesheet",
		cssFilename,
	}

	for _, attachment := range attachmentFilenames {
		args = append(args, "--attachment", attachment)
	}

	cmd := exec.CommandContext(ctx, r.BwrapPath, args...)
	cmd.Env = []string{"HOME=/tmp", "TMPDIR=/tmp", "PATH=/usr/sbin:/usr/bin:/sbin:/bin"}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("weasyprint failed: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return nil
}
