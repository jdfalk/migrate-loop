// Package phases contains the state-machine phases: PLAN, CODE, REDIRECT, COVER, PR.
package phases

import (
	"github.com/jdfalk/migrate-loop/internal/agent"
	"github.com/jdfalk/migrate-loop/internal/runner"
	"github.com/jdfalk/migrate-loop/internal/worktree"
)

type Deps struct {
	Agent    agent.Agent
	Runner   runner.Runner
	Worktree *worktree.Worktree
}
