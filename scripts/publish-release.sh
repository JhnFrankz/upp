#!/usr/bin/env bash
# publish-release.sh — curated release notes + idempotent GitHub Release
# create-or-upload. Runs in CI on tag pushes (refs/tags/v*).
# NEVER creates or pushes tags and NEVER deletes releases: the tag is
# created only by `make publish`; a published tag is never retracted.
# Usage:
#   scripts/publish-release.sh notes     # tag message -> notes markdown (stdout)
#   scripts/publish-release.sh publish   # create release or upload missing assets
# Requires: GITHUB_REF_NAME=vX.Y.Z and GH_TOKEN with contents:write on
# releases (CI provides both automatically via github.token).

set -euo pipefail

TAG="${GITHUB_REF_NAME:-}"

usage() {
    echo "Usage: $0 {notes|publish}" >&2
    exit 2
}

tag_message() {
    git tag -l --format='%(contents)' "$TAG"
}

# First line of the annotated tag message, minus an optional leading
# "upp vX.Y.Z:"/"upp vX.Y.Z —" prefix (the repo's tag-message habit).
summary() {
    local first normalized
    first="$(tag_message | sed -n '1p')"
    normalized="$(printf '%s\n' "$first" | sed -E 's/^upp v[0-9]+\.[0-9]+\.[0-9]+[[:space:]]*[:—-][[:space:]]*//')"
    if [ -n "$normalized" ]; then
        printf '%s\n' "$normalized"
    else
        printf '%s\n' "$first"
    fi
}

# Curated notes: title line, ## What's new (tag message lines 2+, no blanks),
# ## Assets (dist file names), checksums warning.
notes() {
    local first rest f
    first="$(summary)"
    rest="$(tag_message | sed -n '2,$p' | sed '/^[[:space:]]*$/d')"
    printf 'upp %s — %s\n\n' "$TAG" "$first"
    printf '## What'\''s new\n\n'
    if [ -n "$rest" ]; then
        printf '%s\n' "$rest" | sed -E 's/^[[:space:]]*([-*][[:space:]]+)?/- /'
    fi
    printf '\n## Assets\n\n'
    for f in dist/upp-*.tar.gz dist/upp-*.zip dist/checksums.txt; do
        if [ -f "$f" ]; then
            basename "$f"
        fi
    done | sed 's/^/- /'
    printf '\nchecksums.txt must keep shipping with every release — self-update fails closed without it.\n'
}

# Idempotent: release exists -> upload only missing assets; otherwise
# create it with the curated notes. Never git tag / git push / gh release delete.
publish() {
    local existing f name title
    if gh release view "$TAG" >/dev/null 2>&1; then
        existing="$(gh release view "$TAG" --json assets --jq '.assets[].name')"
        for f in dist/upp-*.tar.gz dist/upp-*.zip dist/checksums.txt; do
            [ -f "$f" ] || continue
            name="$(basename "$f")"
            if printf '%s\n' "$existing" | grep -qxF "$name"; then
                echo "Asset '$name' already uploaded; skipping"
            else
                gh release upload "$TAG" "$f"
            fi
        done
    else
        title="upp $TAG — $(summary)"
        notes > notes.md
        # notes.md is intentionally left behind if gh release create fails
        # (debuggability); it is removed on success.
        gh release create "$TAG" dist/upp-*.tar.gz dist/upp-*.zip dist/checksums.txt \
            --title "$title" --notes-file notes.md
        rm -f notes.md
    fi
}

main() {
    if [ -z "$TAG" ]; then
        echo "GITHUB_REF_NAME is required (publish only runs on tag pushes)" >&2
        exit 1
    fi
    printf '%s\n' "$TAG" | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+$' || {
        echo "TAG '$TAG' must match vX.Y.Z" >&2
        exit 1
    }
    case "${1:-}" in
        notes) notes ;;
        publish) publish ;;
        *) usage ;;
    esac
}

main "$@"
