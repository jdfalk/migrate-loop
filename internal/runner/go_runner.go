package runner

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// GoRunner executes `go test -json` and parses the streaming output into a
// Result. Non-zero exit codes from failing tests are treated as a normal
// outcome; only context cancellation or missing binaries surface as errors.
type GoRunner struct {
	Cmd string // e.g. "go test -race -json ./..."
}

func NewGoRunner(cmd string) *GoRunner {
	if cmd == "" {
		cmd = "go test -race -json ./..."
	}
	return &GoRunner{Cmd: cmd}
}

func (g *GoRunner) Run(ctx context.Context, cwd string) (Result, error) {
	args := strings.Fields(g.Cmd)
	if len(args) < 1 {
		return Result{}, fmt.Errorf("runner: empty command")
	}
	c := exec.CommandContext(ctx, args[0], args[1:]...)
	c.Dir = cwd
	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr
	if err := c.Run(); err != nil {
		if ctx.Err() != nil {
			return Result{}, ctx.Err()
		}
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			return Result{}, fmt.Errorf("runner: %w; stderr: %s", err, stderr.String())
		}
	}
	res, err := ParseTestJSON(stdout.Bytes())
	if err != nil {
		return res, err
	}
	if stderr.Len() > 0 {
		res.Errors = append(res.Errors, "stderr: "+stderr.String())
	}
	return res, nil
}
