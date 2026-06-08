#!/usr/bin/env sh
set -eu

REPO="dbkuaizi/seven-player"
TAG="${CNB_BRANCH:-}"
if [ -z "$TAG" ]; then
  TAG="$(git describe --tags --exact-match 2>/dev/null || true)"
fi
if [ -z "$TAG" ]; then
  echo "Missing release tag" >&2
  exit 1
fi

GITHUB_USER="${GIT_USERNAME:-x-access-token}"
: "${GIT_ACCESS_TOKEN:?GIT_ACCESS_TOKEN is required}"

mkdir -p release

github_remote="https://${GITHUB_USER}:${GIT_ACCESS_TOKEN}@github.com/${REPO}.git"
git push --force "$github_remote" "HEAD:refs/tags/${TAG}"

api="https://api.github.com/repos/${REPO}"
auth_header="Authorization: Bearer ${GIT_ACCESS_TOKEN}"
accept_header="Accept: application/vnd.github+json"
version_header="X-GitHub-Api-Version: 2022-11-28"
name="Seven Player ${TAG}"
body="$(cat release/RELEASE_NOTES.md)"

status="$(
  curl -sS \
    -o release/github-release.json \
    -w "%{http_code}" \
    -H "$auth_header" \
    -H "$accept_header" \
    -H "$version_header" \
    "${api}/releases/tags/${TAG}"
)"

if [ "$status" = "200" ]; then
  release_id="$(jq -r '.id' release/github-release.json)"
  payload="$(jq -n --arg name "$name" --arg body "$body" '{name:$name, body:$body, draft:false, prerelease:false}')"
  curl -sS -f \
    -X PATCH \
    -H "$auth_header" \
    -H "$accept_header" \
    -H "$version_header" \
    -H "Content-Type: application/json" \
    -d "$payload" \
    "${api}/releases/${release_id}" \
    > release/github-release.json
elif [ "$status" = "404" ]; then
  payload="$(jq -n --arg tag "$TAG" --arg name "$name" --arg body "$body" '{tag_name:$tag, target_commitish:"main", name:$name, body:$body, draft:false, prerelease:false}')"
  curl -sS -f \
    -X POST \
    -H "$auth_header" \
    -H "$accept_header" \
    -H "$version_header" \
    -H "Content-Type: application/json" \
    -d "$payload" \
    "${api}/releases" \
    > release/github-release.json
else
  cat release/github-release.json >&2
  echo "Failed to query GitHub release, status: ${status}" >&2
  exit 1
fi

jq -r '"GitHub release ready: \(.html_url)"' release/github-release.json
