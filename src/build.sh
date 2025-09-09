#!/usr/bin/env sh

ensure_permissions() {
    DIR="$1"
    REQ_GROUP="$2"
    REQ_PERMS="$3"

    # Get current group owner
    CUR_GROUP=$(stat -c "%G" "$DIR")
    # Get current permissions in octal format
    CUR_PERMS=$(stat -c "%a" "$DIR")

    # Check and update group ownership if needed
    if [ "$CUR_GROUP" != "$REQ_GROUP" ]; then
        echo "Changing group ownership of $DIR to $REQ_GROUP..."
        sudo chown :"$REQ_GROUP" "$DIR"
    fi

    # Check and update permissions if needed
    if [ "$CUR_PERMS" != "$REQ_PERMS" ]; then
        echo "Updating permissions of $DIR to $REQ_PERMS..."
        sudo chmod "$REQ_PERMS" "$DIR"
    fi
}

BRANCH=`git rev-parse --abbrev-ref HEAD`
BRANCH=$(echo "$BRANCH" | tr '/' '_')
BINARY=dvol
OUTPUT=/opt/bin
CHECK_PERMS=0

# Parse arguments
while [ "$#" -gt 0 ]; do
    case "$1" in
        -c|--checkperms)
            CHECK_PERMS=1
            ;;
        *)
            OUTPUT="$1"
            ;;
    esac
    shift
done

if [ "$BRANCH" = "master" ] || [ "$BRANCH" = "main" ] || [ "$BRANCH" = "develop" ]; then
    FULLNAME="$BINARY"
else
    FULLNAME="$BINARY-$BRANCH"
fi

# Run permission check if the flag was passed
if [ "$CHECK_PERMS" -eq 1 ]; then
    ensure_permissions "$OUTPUT" "devops" "775"
fi

echo "Building ${OUTPUT}/${FULLNAME}"
go build -o ${OUTPUT}/${FULLNAME} .
