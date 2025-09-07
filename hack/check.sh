#!/usr/bin/env bash

readonly reset=$(tput sgr0)
readonly red=$(tput bold; tput setaf 1)
readonly green=$(tput bold; tput setaf 2)

exit_code=0
check_scope=$1
if [[ "${check_scope}" = "all" ]]; then
    echo "all"
    files=($(git ls-files | grep "\.go$" | grep -v -e "^third_party" -e "^vendor"))
else
    files=($(git diff --cached --name-only --diff-filter ACM | grep "\.go$" | grep -v -e "^third_party" -e "^vendor"))
fi

echo -e "${green}1. Formatting code style"
if [[ "${#files[@]}" -ne 0 ]]; then
    goimports -w ${files[@]}
fi

echo -e "${green}2. Linting"
if ! command -v golangci-lint &> /dev/null; then
    echo "${red}golangci-lint command not found. Please install it first."
    exit_code=1
else
    # If golangci-lint was built with an older Go than the module target, hint to upgrade
    lint_go_ver=$(golangci-lint --version 2>/dev/null | sed -n 's/.*built with go\([0-9]\+\.[0-9]\+\).*/\1/p')
    mod_go_ver=$(sed -n 's/^go \([0-9]\+\.[0-9]\+\).*/\1/p' go.mod | head -n1)
    if [[ -n "${lint_go_ver}" && -n "${mod_go_ver}" ]]; then
        lint_minor=$(echo "${lint_go_ver}" | cut -d. -f2)
        mod_minor=$(echo "${mod_go_ver}" | cut -d. -f2)
        if [[ ${lint_minor} -lt ${mod_minor} ]]; then
            echo "${red}can't load config: the Go language version (go${lint_go_ver}) used to build golangci-lint is lower than the targeted Go version (${mod_go_ver})"
            echo "${red}Hint: upgrade golangci-lint (run: make init)"
            exit_code=1
        fi
    fi

    if [[ ${exit_code} -eq 0 ]] && ! golangci-lint run; then
        echo "${red}Linting issues found."
        exit_code=1
    fi
fi

if [[ ${exit_code} -ne 0 ]]; then
    echo "${red}Please fix the errors above :)"
else
    echo "${green}Nice!"
fi
echo "${reset}"

exit ${exit_code}
