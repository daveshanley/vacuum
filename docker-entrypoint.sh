#!/bin/sh
set -e

# If supplied command starts with `-`, we assume it is a flag for vacuum,
# otherwise check if command is a valid vacuum subcommand.
# If it is, prepend `vacuum` to script parameters:
# if `$@` is `lint`, it becomes `vacuum lint`
if [ "${1#-}" != "${1}" ] || [ -z "$(command -v "${1}")" ]; then
    set -- vacuum "$@"
fi

exec "$@"
