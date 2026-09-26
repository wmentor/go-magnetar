#!/bin/sh -e

echo "check required commands"

command -v git >/dev/null 2>&1 || { echo "Error: git is required."; exit 1; }
command -v git-cliff >/dev/null 2>&1 || { echo "Error: git-cliff is required."; exit 1; }

echo "work directory: $(pwd)"
echo "check uncommited changes"

git diff --quiet HEAD || { echo "found local chnages" && exit 1 }

export NEXT_TAG=$(git cliff --bumped-version 2> /dev/null)

echo "new release tag: $NEXT_TAG"

echo "update change log"

export TAG_CONTENT=$(git cliff --bump --unreleased --strip all)

git cliff -u --tag $NEXT_TAG --prepend CHANGELOG.md
git add CHANGELOG.md
git commit -m "chore(release): prepare for $NEXT_TAG"

git tag -a $NEXT_TAG -m "$TAG_CONTENT"

git push origin $NEXT_TAG
