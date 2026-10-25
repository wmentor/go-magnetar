#!/bin/sh -e

NO_COMMIT=0

for arg in "$@"; do
    case "$arg" in
        --no-commit|-n) NO_COMMIT=1 ;;
        *) echo "Unknown argument: $arg"; exit 1 ;;
    esac
done

echo "check required commands"

command -v git >/dev/null 2>&1 || { echo "Error: git is required."; exit 1; }
command -v git-cliff >/dev/null 2>&1 || { echo "Error: git-cliff is required."; exit 1; }
command -v curl >/dev/null 2>&1 || { echo "Error: curl is required."; exit 1; }
command -v jq >/dev/null 2>&1 || { echo "Error: jq is required."; exit 1; }

echo "work directory: $(pwd)"

if [ "$NO_COMMIT" -eq 0 ]; then
    echo "check uncommited changes"
    git diff --quiet HEAD || { echo "found local changes" ; exit 1; }
fi

REPO_OWNER=$(git remote get-url origin | sed 's/.*github\.com[:/]\([^/]*\)\/.*/\1/')
REPO_NAME=$(git remote get-url origin | sed 's/.*github\.com[:/][^/]*\/\(.*\)\.git/\1/')

echo "github repo: $REPO_OWNER/$REPO_NAME"

# URL-encode a string. Uses python3 if available, falls back to a basic sed approach.
url_encode() {
    if command -v python3 >/dev/null 2>&1; then
        python3 -c "import urllib.parse,sys; print(urllib.parse.quote(sys.argv[1]))" "$1"
    else
        # Basic encoding for the most common special characters in email addresses
        printf '%s' "$1" | sed 's/@/%40/g; s/+/%2B/g; s/ /%20/g'
    fi
}

LAST_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "")

if [ -z "$LAST_TAG" ]; then
    COMMIT_RANGE="HEAD"
else
    COMMIT_RANGE="${LAST_TAG}..HEAD"
fi

echo "collecting contributors from commits: $COMMIT_RANGE"

# Resolve GitHub login for each commit author email.
#
# Resolution order:
#   1. ID+login@users.noreply.github.com  — extract login directly from email
#   2. ID@users.noreply.github.com        — call GET /user/{id} (public, no token needed)
#   3. Regular email + GITHUB_TOKEN       — GET /search/users?q=email+in:email
#   4. Regular email + GITHUB_TOKEN       — GET /repos/.../commits?author=email
#
# Set GITHUB_TOKEN to enable API-based lookup (cases 3 and 4).
# Without a token, only noreply GitHub emails are resolved.
#
# After resolution, commit counts are aggregated per login (handles the case
# where one person commits with multiple email addresses) and sorted descending.

# Step 1: collect commit counts per email, sorted by count descending
# Format: "   N email"
TMP_EMAIL_COUNTS=/tmp/go_magnetar_email_counts_$$
git log --format="%ae" $COMMIT_RANGE 2>/dev/null \
    | sort | uniq -c | sort -rn \
    > "$TMP_EMAIL_COUNTS"

# Step 2: resolve each email to a GitHub login and accumulate counts per login.
# We use a flat file as an associative map: "LOGIN COUNT" lines, updated in place.
TMP_LOGIN_COUNTS=/tmp/go_magnetar_login_counts_$$
: > "$TMP_LOGIN_COUNTS"

# GITHUB_TOKEN must be exported by the caller for API lookups to work
if [ -z "${GITHUB_TOKEN:-}" ]; then
    echo "Warning: GITHUB_TOKEN is not set. Some email addresses may not be resolved."
    echo "To enable API lookups, set GITHUB_TOKEN: export GITHUB_TOKEN=..."
fi

while read -r COUNT EMAIL; do
    LOGIN=""

    # Case 1: ID+login@users.noreply.github.com
    if echo "$EMAIL" | grep -q 'users\.noreply\.github\.com'; then
        LOGIN=$(echo "$EMAIL" | sed -n 's/^[0-9]*+\([^@]*\)@users\.noreply\.github\.com$/\1/p')

        # Case 2: ID@users.noreply.github.com (legacy format, no token needed)
        if [ -z "$LOGIN" ]; then
            USER_ID=$(echo "$EMAIL" | sed -n 's/^\([0-9]*\)@users\.noreply\.github\.com$/\1/p')
            if [ -n "$USER_ID" ]; then
                API_RESP=$(curl -sf \
                    ${GITHUB_TOKEN:+-H "Authorization: Bearer $GITHUB_TOKEN"} \
                    "https://api.github.com/user/$USER_ID" || echo "")
                LOGIN=$(echo "$API_RESP" | jq -r '.login // empty' 2>/dev/null || echo "")
            fi
        fi
    fi

    # Case 3: search GitHub users by email (requires GITHUB_TOKEN)
    if [ -z "$LOGIN" ] && [ -n "$GITHUB_TOKEN" ]; then
        ENCODED=$(url_encode "$EMAIL")
        API_RESP=$(curl -sf \
            -H "Authorization: Bearer $GITHUB_TOKEN" \
            -H "Accept: application/vnd.github+json" \
            "https://api.github.com/search/users?q=${ENCODED}+in:email" || echo "")
        LOGIN=$(echo "$API_RESP" | jq -r '.items[0].login // empty' 2>/dev/null || echo "")
    fi

    # Case 4: look up author via repo commits API (requires GITHUB_TOKEN)
    if [ -z "$LOGIN" ] && [ -n "$GITHUB_TOKEN" ]; then
        ENCODED=$(url_encode "$EMAIL")
        # Strip control characters before passing to jq to avoid parse errors
        API_RESP=$(curl -sf \
            -H "Authorization: Bearer $GITHUB_TOKEN" \
            -H "Accept: application/vnd.github+json" \
            "https://api.github.com/repos/$REPO_OWNER/$REPO_NAME/commits?author=${ENCODED}&per_page=1" \
            2>/dev/null | python3 -c "import sys,json; data=sys.stdin.read(); print(json.dumps(json.loads(data), separators=(',', ':')))" || echo "")
        LOGIN=$(echo "$API_RESP" | jq -r '.[0].author.login // empty' 2>/dev/null || echo "")
    fi

    if [ -n "$LOGIN" ]; then
        # Accumulate commit count for this login (one person may have multiple emails)
        EXISTING=$(grep "^${LOGIN} " "$TMP_LOGIN_COUNTS" | awk '{print $2}')
        if [ -n "$EXISTING" ]; then
            NEW_COUNT=$((EXISTING + COUNT))
            # Update the existing line in-place using a temp file
            TMP_SWAP=/tmp/go_magnetar_swap_$$
            sed "s/^${LOGIN} .*/${LOGIN} ${NEW_COUNT}/" "$TMP_LOGIN_COUNTS" > "$TMP_SWAP"
            mv "$TMP_SWAP" "$TMP_LOGIN_COUNTS"
            echo "  merged:   $EMAIL ($COUNT commits) -> @$LOGIN (total: $NEW_COUNT)"
        else
            echo "$LOGIN $COUNT" >> "$TMP_LOGIN_COUNTS"
            echo "  resolved: $EMAIL ($COUNT commits) -> @$LOGIN"
        fi
    else
        echo "  warning: could not resolve GitHub login for $EMAIL (set GITHUB_TOKEN to enable API lookups)"
    fi
done < "$TMP_EMAIL_COUNTS"

rm -f "$TMP_EMAIL_COUNTS"

# Step 3: sort logins by accumulated commit count descending
TMP_LOGINS_SORTED=/tmp/go_magnetar_logins_sorted_$$
if [ -s "$TMP_LOGIN_COUNTS" ]; then
    sort -k2 -rn "$TMP_LOGIN_COUNTS" | awk '{print $1}' > "$TMP_LOGINS_SORTED"
else
    : > "$TMP_LOGINS_SORTED"
fi
rm -f "$TMP_LOGIN_COUNTS"

LOGINS_COUNT=$(wc -l < "$TMP_LOGINS_SORTED" | tr -d ' ')
echo "resolved $LOGINS_COUNT contributor(s)"

NEXT_TAG=$(git cliff --bumped-version 2> /dev/null)
export NEXT_TAG

echo "new release tag: $NEXT_TAG"

echo "update changelog"

git cliff -u --tag "$NEXT_TAG" --prepend CHANGELOG.md

# Insert a contributors section at the end of the release entry.
# Find the release entry and append contributors before the next release header
# (or before the "All Changes" header if it exists, or at the end of file).
#
# Result in CHANGELOG.md:
#
#   ## [1.6.0] - 2026-10-17
#
#   ### 🚀 Features
#   ...
#
#   ### 🔧 Bug Fixes
#   ...
#
#   ### 👥 Contributors
#
#   <div align="left">
#   <a href="https://github.com/login"><img src="https://avatars.githubusercontent.com/login?v=4&s=32"
#      width="32" height="32" alt="@login" title="@login" style="border-radius:50%"></a> ...
#   </div>
#
#   ## [1.5.0] - 2026-09-15
#   ...
#
if [ "$LOGINS_COUNT" -gt 0 ]; then
    # Build the avatar row as a single line
    AVATAR_ROW=""
    while IFS= read -r LOGIN; do
        AVATAR_ROW="${AVATAR_ROW}<a href=\"https://github.com/${LOGIN}\" data-hovercard-type=\"user\" data-hovercard-url=\"/users/${LOGIN}/hovercard\" data-octo-click=\"hovercard-link-click\" data-octo-dimensions=\"link_type:self\" aria-keyshortcuts=\"Alt+ArrowUp\"><img src=\"https://avatars.githubusercontent.com/${LOGIN}?v=4&s=64\" width=\"32\" height=\"32\" data-view-component=\"true\" class=\"avatar-user\" alt=\"@${LOGIN}\" title=\"@${LOGIN}\" style=\"border-radius: 50% !important;\"></a> "
    done < "$TMP_LOGINS_SORTED"

    TMP_CHANGELOG=/tmp/go_magnetar_changelog_$$
    
    # Insert contributors before the second release header (i.e., at the end of
    # the newest release entry that was just prepended by git-cliff).
    awk -v avatars="$AVATAR_ROW" '
    BEGIN { inserted = 0; seen_first = 0 }
    /^## \[/ {
        if (seen_first && !inserted) {
            # Second release header found — insert contributors block before it
            print ""
            print "### 👥 Contributors"
            print ""
            print "<div align=\"left\" style=\"padding-left: 20px;\">"
            print avatars
            print "</div><br/><br/>"
            print ""
            inserted = 1
        }
        seen_first = 1
    }
    { print }
    END {
        # Only one release in the file — append contributors at the end
        if (!inserted && seen_first) {
            print ""
            print "### 👥 Contributors"
            print ""
            print "<div align=\"left\" style=\"padding-left: 20px;\">"
            print avatars
            print "</div>"
        }
    }
    ' CHANGELOG.md > "$TMP_CHANGELOG"

    mv "$TMP_CHANGELOG" CHANGELOG.md
    echo "contributors section appended to changelog"
fi

rm -f "$TMP_LOGINS_SORTED"

if [ "$NO_COMMIT" -eq 1 ]; then
    echo "skipping git add / commit (--no-commit)"
else
    git add CHANGELOG.md
    git commit -m "chore(release): prepare for $NEXT_TAG"
fi
