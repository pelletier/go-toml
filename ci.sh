#!/usr/bin/env bash


stderr() {
    echo "$@" 1>&2
}

usage() {
    b=$(basename "$0")
    echo $b: ERROR: "$@" 1>&2

    cat 1>&2 <<EOF

DESCRIPTION

    $(basename "$0") is the script to run continuous integration commands for
    go-toml on unix.

    Requires Go and Git to be available in the PATH. Expects to be ran from the
    root of go-toml's Git repository.

USAGE

    $b COMMAND [OPTIONS...]

COMMANDS

coverage [OPTIONS...] [BRANCH]

    Generates code coverage.

    ARGUMENTS

        BRANCH  Optional. Defines which Git branch to use when reporting
                coverage. Defaults to HEAD.

    OPTIONS

        -d      Compare coverage of HEAD with the one of BRANCH. In this form,
                the BRANCH argument is required. Exit code is non-zero when
                coverage percentage decreased.
EOF
    exit 1
}

cover() {
    branch="${1}"
    dir="$(mktemp -d)"

    stderr "Executing coverage for ${branch} at ${dir}"

    if [ "${branch}" = "HEAD" ]; then
	cp -r . "${dir}/"
    else
	git worktree add "$dir" "$branch"
    fi

    pushd "$dir"
    go test -covermode=atomic -coverprofile=coverage.out ./...
    go tool cover -func=coverage.out
    popd

    if [ "${branch}" != "HEAD" ]; then
	git worktree remove --force "$dir"
    fi
}

coverage() {
    case "$1" in
	-d)
	    shift
	    target="${1?Need to provide a target branch argument}"

	    output_dir="$(mktemp -d)"
	    target_out="${output_dir}/target.txt"
	    head_out="${output_dir}/head.txt"
	    
	    cover "${target}" > "${target_out}"
	    cover "HEAD" > "${head_out}"

	    cat "${target_out}"
	    cat "${head_out}"

	    echo ""

	    target_pct="$(cat ${target_out} |sed -E 's/.*total.*\t([0-9.]+)%/\1/;t;d')"
	    head_pct="$(cat ${head_out} |sed -E 's/.*total.*\t([0-9.]+)%/\1/;t;d')"
	    echo "Results: ${target} ${target_pct}% HEAD ${head_pct}%"

	    delta_pct=$(echo "$head_pct - $target_pct" | bc -l)
	    echo "Delta: ${delta_pct}"

	    if [[ $delta_pct = \-* ]]; then
		echo "Regression!";
		return 1
	    fi
	    return 0
	    ;;
    esac

    cover "${1-HEAD}"
}

case "$1" in
    coverage) shift; coverage $@;;
    *) usage "bad argument $1";;
esac
