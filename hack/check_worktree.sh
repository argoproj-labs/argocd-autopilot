#!/bin/sh

echo "checking worktree..."
res=$(git status -s)
if [[ -z "$res" ]]; then
    echo worktree is clean!
else
    echo error: working tree is not clean! make sure you run 'make pre-commit' and commit the changes.
    echo "$res"
    exit 1
fi