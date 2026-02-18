package app

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

const sandboxDefaultStylesheetPath = "/defaults/default.css"

func (r WeasyprintRunner) GeneratePDF(ctx context.Context, workDir string, htmlFilename string, cssFilename string, attachmentFilenames []string, output io.Writer) error {
	args := r.buildArgs(workDir, htmlFilename, cssFilename, attachmentFilenames)

	cmd := exec.CommandContext(ctx, r.BwrapPath, args...)
	cmd.Stdout = output
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("weasyprint failed: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

func (r WeasyprintRunner) buildArgs(workDir string, htmlFilename string, cssFilename string, attachmentFilenames []string) []string {
	defaultStylesheetPath := r.DefaultStylesheetPath
	if defaultStylesheetPath == "" {
		defaultStylesheetPath = "assets/default.css"
	}

	args := []string{
		"--unshare-all",
		"--new-session",
		"--clearenv",

		"--ro-bind", "/usr", "/usr",
		"--ro-bind", "/lib", "/lib",
		"--ro-bind", "/lib64", "/lib64",
		"--ro-bind", "/bin", "/bin",
		"--ro-bind", "/etc/fonts", "/etc/fonts",
		"--ro-bind", "/var/cache/fontconfig", "/var/cache/fontconfig",

		"--ro-bind", defaultStylesheetPath, sandboxDefaultStylesheetPath,
		"--ro-bind", workDir, "/workspace",
		"--chdir", "/workspace",
		"--setenv", "PATH", "/usr/local/bin:/usr/bin",
		"--",
		r.WeasyprintPath,
		htmlFilename,
		"-",
		"--stylesheet",
		cssFilename,
	}

	for _, attachment := range attachmentFilenames {
		args = append(args, "--attachment", attachment)
	}

	return args
}
