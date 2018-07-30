#!/bin/sh

set -e

if [ ! -f "build/env.sh" ]; then
    echo "$0 must be run from the root of the repository."
    exit 2
fi

# Create fake Go workspace if it doesn't exist yet.
workspace="$PWD/build/_workspace"
root="$PWD"
eaidir="$workspace/src/github.com/ethereumai"
if [ ! -L "$eaidir/go-ethereumai" ]; then
    mkdir -p "$eaidir"
    cd "$eaidir"
    ln -s ../../../../../. go-ethereumai
    cd "$root"
fi

# Set up the environment to use the workspace.
GOPATH="$workspace"
export GOPATH

# Run the command inside the workspace.
cd "$eaidir/go-ethereumai"
PWD="$eaidir/go-ethereumai"

# Launch the arguments with the configured environment.
exec "$@"
