#!/bin/sh
set -e

# Treat leading flags and vacuum subcommands as vacuum arguments, while still
# allowing CI runners to start their own shell inside the image.
if [ "${1#-}" != "${1}" ] || [ -z "$(command -v "${1}")" ]; then
    set -- vacuum "$@"
fi

exec "$@"
