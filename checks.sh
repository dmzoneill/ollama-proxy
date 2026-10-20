#!/usr/bin/env bash
# Invoked by the shared CICD lint job before Super-linter.
#
# Super-linter's per-file GO linter runs one command:
#   golangci-lint run -c "${GO_LINTER_RULES}" --fast-only <every changed .go file>
# golangci-lint rejects that when the paths are not all in one directory
# ("named files must all be in one directory") and does not lint anything.
# GNU parallel reads PARALLEL. --max-lines 1 makes that linter invoke
# golangci-lint once per file, which still lints every changed Go file.
# GO_MODULES is unchanged: it lints each changed package with the full
# linter set from .github/linters/.golangci.yml (a link to the repo config).
set -euo pipefail

if [[ -n "${GITHUB_ENV:-}" ]]; then
	printf '%s\n' 'PARALLEL=--max-lines 1' >>"${GITHUB_ENV}"
fi
