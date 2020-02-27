#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

file="$1"

signedoff_regex='^Signed-off-by: '

if [ "$(grep -c "$signedoff_regex" "$file")" != "1" ]; then
	printf >&2 "Signed-off-by line is missing.\n"
	exit 1
fi
