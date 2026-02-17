package app

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

func (r WeasyprintRunner) GeneratePDF(ctx context.Context, workDir string, htmlFilename string, cssFilename string, attachmentFilenames []string, output io.Writer) error {
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
		"--setenv", "PATH", "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
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

	cmd := exec.CommandContext(ctx, r.BwrapPath, args...)
	cmd.Env = []string{"HOME=/tmp", "TMPDIR=/tmp", "PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"}
	cmd.Stdout = output
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("weasyprint failed: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return nil
}
