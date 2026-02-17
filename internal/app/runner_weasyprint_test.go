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

func TestBuildArgsIncludesSandboxDefaultsStylesheetBind(t *testing.T) {
	runner := WeasyprintRunner{
		BwrapPath:      "bwrap",
		WeasyprintPath: "weasyprint",
	}

	args := runner.buildArgs("/tmp/work", "index.html", "/defaults/default.css", nil)

	assert.True(t, containsSequence(args, []string{"--dir", "/defaults"}), "expected --dir /defaults in args")
	assert.True(t, containsSequence(args, []string{"--ro-bind", "assets/default.css", sandboxDefaultStylesheetPath}), "expected default stylesheet ro-bind in args")
	assert.True(t, containsSequence(args, []string{"--ro-bind", "/tmp/work", "/workspace"}), "expected workspace read-only bind in args")
	assert.False(t, containsSequence(args, []string{"--ro-bind", "/etc", "/etc"}), "expected full /etc mount to be removed")
	assert.False(t, containsSequence(args, []string{"--ro-bind", "/sbin", "/sbin"}), "expected /sbin mount to be removed")
	assert.True(t, containsSequence(args, []string{"--", "weasyprint", "index.html", "-", "--stylesheet", "/defaults/default.css"}), "expected weasyprint invocation segment in args")
}

func TestBuildArgsUsesCustomStylesheetBindAndAttachments(t *testing.T) {
	runner := WeasyprintRunner{
		BwrapPath:             "bwrap",
		WeasyprintPath:        "weasyprint",
		DefaultStylesheetPath: "/app/assets/custom.css",
	}

	args := runner.buildArgs("/tmp/work", "doc.html", "style.css", []string{"a.txt", "b.txt"})

	assert.True(t, containsSequence(args, []string{"--ro-bind", "/app/assets/custom.css", sandboxDefaultStylesheetPath}), "expected custom stylesheet ro-bind in args")
	assert.True(t, containsSequence(args, []string{"--attachment", "a.txt"}), "expected first attachment in args")
	assert.True(t, containsSequence(args, []string{"--attachment", "b.txt"}), "expected second attachment in args")
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
		Stderr:                &bytes.Buffer{},
	}

	var output bytes.Buffer
	stderr := runner.Stderr.(*bytes.Buffer)
	err := runner.GeneratePDF(context.Background(), workDir, "index.html", "style.css", nil, &output)
	assert.NoError(t, err)
	assert.NotZero(t, output.Len())
	assert.True(t, bytes.HasPrefix(output.Bytes(), []byte("%PDF")), "expected generated output to start with %PDF")
	assert.NotContains(t, strings.ToLower(stderr.String()), "warning")
}

func containsSequence(haystack []string, needle []string) bool {
	if len(needle) == 0 {
		return true
	}
	if len(needle) > len(haystack) {
		return false
	}

	for index := 0; index <= len(haystack)-len(needle); index++ {
		match := true
		for offset := range needle {
			if haystack[index+offset] != needle[offset] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}

	return false
}
