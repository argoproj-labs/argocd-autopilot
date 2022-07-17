#!/bin/sh
GIT_BRANCH=$(git rev-parse --symbolic-full-name --verify --quiet --abbrev-ref HEAD)

echo "$GIT_BRANCH" | grep -Eq '^release-v(\d+\.)?(\d+\.)?(\*|\d+)$'

if [ -z "$GIT_REPO" ]; then
    echo "error: git repo not defined"
    exit 1
fi

if [ -z "$GITHUB_TOKEN" ]; then
    echo "error: GITHUB_TOKEN token not defined"
    exit 1
fi

if [ -z "$PRERELEASE" ]; then
    PRERELEASE=false
fi

if [ "$?" == "0" ]; then
    echo "on release branch: $GIT_BRANCH"
    echo ""
    echo "uploading files:"
    ls -1a ./dist/*.tar.gz ./dist/*.sha256
    echo ""

    FILE="./docs/releases/release_notes.md"
    echo "using release notes file: ./docs/releases/release_notes.md"
    cat $FILE | head -n 5 && echo ...
    echo ""

    RELEASE_VER=$(echo ${GIT_BRANCH} | sed -E 's/^release-([^"]+).*/\1/')
    echo "creating release ${RELEASE_VER}"
    echo ""

    if [ "$PRE_RELEASE" ]; then
        echo "using pre-release"
        echo ""
    fi

    echo "running: gh release create --repo $GIT_REPO -t $RELEASE_VER -F $FILE --target $GIT_BRANCH --prerelease=$PRERELEASE $RELEASE_VER ./dist/*.tar.gz ./dist/*.sha256"
    
    if [ "$DRY_RUN" == "1" ]; then
        exit 0
    fi

    gh release create --repo $GIT_REPO -t $RELEASE_VER -F $FILE --target $GIT_BRANCH --prerelease=$PRERELEASE $RELEASE_VER ./dist/*.tar.gz ./dist/*.sha256
else 
    echo "not on release branch: $GIT_BRANCH"
    exit 1
fi
