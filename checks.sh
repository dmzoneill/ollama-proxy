#!/usr/bin/env bash
# Invoked by the shared CICD lint job before Super-linter.
#
# Super-linter's per-file GO linter runs one command:
#   golangci-lint run -c "${GO_LINTER_RULES}" --fast-only <every changed .go file>
# golangci-lint rejects that when the paths are not all in one directory
# ("named files must all be in one directory") and does not lint anything.
# GNU parallel reads PARALLEL. --max-lines 1 makes that linter invoke
# golangci-lint once per file, which still lints every changed Go file.
#
# /.ignore lists directories whose only Go files are //go:build ignore.
# Those paths are exported as FILTER_REGEX_EXCLUDE so GO_MODULES does not
# run golangci-lint there. The config it does use is the real file
# .github/linters/.golangci.yml, which mirrors /.golangci.yml.
set -euo pipefail

if [[ -z "${GITHUB_ENV:-}" ]]; then
	exit 0
fi

printf '%s\n' 'PARALLEL=--max-lines 1' >>"${GITHUB_ENV}"

ignore_file="${GITHUB_WORKSPACE:-.}/.ignore"
if [[ ! -f "${ignore_file}" ]]; then
	exit 0
fi

regex=""
while IFS= read -r line || [[ -n "${line}" ]]; do
	# Drop comments and surrounding whitespace.
	line="${line%%#*}"
	line="${line#"${line%%[![:space:]]*}"}"
	line="${line%"${line##*[![:space:]]}"}"
	if [[ -z "${line}" ]]; then
		continue
	fi

	directory=false
	if [[ "${line}" == */ ]]; then
		directory=true
		line="${line%/}"
	fi

	# Escape regex metacharacters in the path.
	escaped="$(printf '%s' "${line}" | sed -e 's/[][(){}.+*?^$|\\]/\\&/g')"
	if [[ "${directory}" == true ]]; then
		pattern="(^|/)${escaped}/"
	else
		pattern="(^|/)${escaped}$"
	fi

	if [[ -z "${regex}" ]]; then
		regex="${pattern}"
	else
		regex="${regex}|${pattern}"
	fi
done <"${ignore_file}"

if [[ -z "${regex}" ]]; then
	exit 0
fi

if [[ -n "${FILTER_REGEX_EXCLUDE:-}" ]]; then
	regex="${FILTER_REGEX_EXCLUDE}|${regex}"
fi
printf 'FILTER_REGEX_EXCLUDE=%s\n' "${regex}" >>"${GITHUB_ENV}"
