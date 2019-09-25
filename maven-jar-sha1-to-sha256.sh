#!/bin/bash

# This script is a convertion tool for maven_jar rules.
# It finds rules that are using the deprecated SHA1 option, and updates them
# to the correct SHA256 sum.
#
# The script only supports artefacts that are available on maven central.

set -uo pipefail

trim() {
    local var="$*"
    # remove leading whitespace characters
    var="${var#"${var%%[![:space:]]*}"}"
    # remove trailing whitespace characters
    var="${var%"${var##*[![:space:]]}"}"
    echo -n "$var"
}

DEPS=$(rg -r '$2' -N --multiline 'maven_jar\(\n(.*)name = "(.*)",$' WORKSPACE)

while read -r DEP_NAME; do
    echo $DEP_NAME

    # TODO(zegl): Find a way to do this without invoking rg twice times :sweat_smile:
    COORD_X=$(rg -N -r "\$2" --multiline "name = \"${DEP_NAME}\",\n(.*)artifact = \"(.*):(.*):(.*)\"," WORKSPACE);
    COORD_Y=$(rg -N -r "\$3" --multiline "name = \"${DEP_NAME}\",\n(.*)artifact = \"(.*):(.*):(.*)\"," WORKSPACE);
    COORD_Z=$(rg -N -r "\$4" --multiline "name = \"${DEP_NAME}\",\n(.*)artifact = \"(.*):(.*):(.*)\"," WORKSPACE);

    COORD_X=$(trim "$COORD_X");
    COORD_Y=$(trim "$COORD_Y");
    COORD_Z=$(trim "$COORD_Z");

    if [ -x ${COORD_X+x} ]; then
        echo "Unable to update ${DEP_NAME}, unexpected format in WORKSPACE"
        continue
    fi

    # Fetch the sha256 from the maven registry
    URL="https://repo1.maven.org/maven2/${COORD_X//.//}/${COORD_Y}/${COORD_Z}/${COORD_Y}-${COORD_Z}.jar"
    NEW_SHA=$(curl --silent "$URL" | sha256sum - | head -c 64)

    if [ ${#NEW_SHA} -ne 64 ]; then
        echo ${#NEW_SHA}
        echo "Could not find new version for ${DEP_NAME}, skipping."
        continue
    fi

    if rg -C999999999 --multiline \
        -r "name = \"${DEP_NAME}\",
    artifact = \"\$2:${COORD_Z}\",
    sha256 = \"${NEW_SHA}\"," \
            "name = \"${DEP_NAME}\",\$\n(.*)artifact = \"(.*):([0-9a-zA-Z\\.]*)\",\$\n(.*)sha1 = \"([0-9a-f]*)\"," WORKSPACE > WORKSPACE.tmp; then
        mv WORKSPACE.tmp WORKSPACE
        echo "Successfully updated ${DEP_NAME} to ${COORD_Z} ( sha256 = $NEW_SHA )"
    else
        echo "Update of ${DEP_NAME} failed"
        continue
    fi

done <<< "$DEPS"
