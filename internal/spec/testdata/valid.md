---
title: trivial add
slug: trivial-add
target_packages:
  - internal/add
  - internal/util
test_runner: "go test -race -json ./..."
prior_examples: []
success_criteria:
  - all tests pass
---

# trivial add

## Behavior
Add two integers.
