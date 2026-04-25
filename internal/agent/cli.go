package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// CLIAgent is an Agent implementation that shells out to the `claude` binary
// using `claude -p --output-format json` and parses the JSON envelope.
type CLIAgent struct {
	Binary string // path to `claude` (default: "claude")
}

// NewCLIAgent returns a CLIAgent that uses `claude` from $PATH.
func NewCLIAgent() *CLIAgent { return &CLIAgent{Binary: "claude"} }

// Run invokes the configured claude binary with the request prompt and
// allowed/disallowed tools, then parses the JSON envelope into Response.
func (c *CLIAgent) Run(ctx context.Context, req Request) (Response, error) {
	bin := c.Binary
	if bin == "" {
		bin = "claude"
	}
	args := []string{"-p", "--output-format", "json"}
	if len(req.AllowedTools) > 0 {
		args = append(args, "--allowed-tools", strings.Join(req.AllowedTools, ","))
	}
	if len(req.DisallowedTools) > 0 {
		args = append(args, "--disallowed-tools", strings.Join(req.DisallowedTools, ","))
	}
	args = append(args, req.Prompt)

	timeout := req.Timeout
	if timeout == 0 {
		timeout = 10 * time.Minute
	}
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(cctx, bin, args...)
	cmd.Dir = req.Cwd
	cmd.Env = append(os.Environ(), envSlice(req.Env)...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	start := time.Now()
	err := cmd.Run()
	res := Response{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		Duration: time.Since(start),
	}
	if cctx.Err() == context.DeadlineExceeded {
		return res, fmt.Errorf("agent: claude -p timed out after %s", timeout)
	}
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			res.ExitCode = exitErr.ExitCode()
		} else {
			return res, fmt.Errorf("agent: claude -p: %w", err)
		}
	}
	parseClaudeJSON(stdout.Bytes(), &res)
	return res, nil
}

// parseClaudeJSON best-effort extracts session_id and total_cost_usd from the
// claude CLI JSON envelope. Non-JSON or unrecognized output is ignored so the
// caller still gets stdout/stderr and exit code.
func parseClaudeJSON(out []byte, r *Response) {
	type cj struct {
		SessionID    string  `json:"session_id"`
		TotalCostUSD float64 `json:"total_cost_usd"`
	}
	var v cj
	if err := json.Unmarshal(bytes.TrimSpace(out), &v); err == nil {
		r.SessionID = v.SessionID
		r.Cost = v.TotalCostUSD
	}
}

func envSlice(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k, v := range m {
		out = append(out, k+"="+v)
	}
	return out
}
