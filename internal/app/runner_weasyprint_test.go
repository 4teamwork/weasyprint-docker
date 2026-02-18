package app

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildArgsUsesCustomStylesheetBindAndAttachments(t *testing.T) {
	runner := WeasyprintRunner{
		BwrapPath:             "bwrap",
		WeasyprintPath:        "weasyprint",
		DefaultStylesheetPath: "/app/assets/custom.css",
	}

	args := runner.buildArgs("/tmp/work", "doc.html", "style.css", []string{"a.txt", "b.txt"})
	joinedArgs := strings.Join(args, " ")
	assert.Contains(t, joinedArgs, "--attachment a.txt")
	assert.Contains(t, joinedArgs, "--attachment b.txt")
	assert.Contains(t, joinedArgs, "--ro-bind /app/assets/custom.css "+sandboxDefaultStylesheetPath)
}

func TestGeneratePDFRunsBwrapWeasyprintCommand(t *testing.T) {
	if _, err := exec.LookPath("bwrap"); err != nil {
		t.Skip("bwrap not available in PATH")
	}
	if _, err := exec.LookPath("weasyprint"); err != nil {
		t.Skip("weasyprint not available in PATH")
	}

	workDir := t.TempDir()

	html := "<html><body><h1>Hello</h1><p>PDF integration test.</p></body></html>"
	css := "body { font-family: sans-serif; }"
	defaultCSS := "@page { size: A4; margin: 2cm; }"

	assert.NoError(t, os.WriteFile(filepath.Join(workDir, "index.html"), []byte(html), 0o600))
	assert.NoError(t, os.WriteFile(filepath.Join(workDir, "style.css"), []byte(css), 0o600))
	assert.NoError(t, os.WriteFile(filepath.Join(workDir, "default.css"), []byte(defaultCSS), 0o600))

	runner := WeasyprintRunner{
		BwrapPath:             "bwrap",
		WeasyprintPath:        "weasyprint",
		DefaultStylesheetPath: filepath.Join(workDir, "default.css"),
	}

	var output bytes.Buffer
	err := runner.GeneratePDF(context.Background(), workDir, "index.html", "style.css", nil, &output)
	assert.NoError(t, err)
	assert.NotZero(t, output.Len())
	assert.True(t, bytes.HasPrefix(output.Bytes(), []byte("%PDF")), "expected generated output to start with %PDF")
}
