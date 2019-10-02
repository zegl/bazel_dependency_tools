#!/bin/bash

set -euo pipefail

# Usage:
# ./update-jar-dep.sh dependency_prefix desired_version
#
# All maven_jar WORKSPACE-rules that match [dependency_prefix] will be updated
# to desired_version. The new sha1 will be updated automatically.
#
# Example:
#   ./update-jar-dep.sh io_netty_netty_ 4.1.41.Final
#

DEPENDENCY_PREFIX=$1
NEW_VERSION=$2

trim() {
    local var="$*"
    # remove leading whitespace characters
    var="${var#"${var%%[![:space:]]*}"}"
    # remove trailing whitespace characters
    var="${var%"${var##*[![:space:]]}"}"
    echo -n "$var"
}

DEPS=$(rg -No "${DEPENDENCY_PREFIX}(.*)\"" -r "${DEPENDENCY_PREFIX}\$1" WORKSPACE);


while read -r DEP_NAME; do
    # TODO(zegl): Find a way to do this without invoking rg twice times :sweat_smile:
    COORD_X=$(rg -N -r "\$2" --multiline "name = \"${DEP_NAME}\",\n(.*)artifact = \"(.*):(.*):(.*)\"," WORKSPACE);
    COORD_Y=$(rg -N -r "\$3" --multiline "name = \"${DEP_NAME}\",\n(.*)artifact = \"(.*):(.*):(.*)\"," WORKSPACE);

    COORD_X=$(trim "$COORD_X");
    COORD_Y=$(trim "$COORD_Y");

    # Fetch the sha1 from the maven registry
    NEW_SHA=$(curl --silent "https://repo1.maven.org/maven2/${COORD_X//.//}/${COORD_Y}/${NEW_VERSION}/${COORD_Y}-${NEW_VERSION}.jar.sha1")

    if [ ${#NEW_SHA} -ne 40 ]; then
        echo ${#NEW_SHA}
        echo "Could not find new version for ${DEP_NAME}, skipping."
        continue
    fi

    rg -C999999999 --multiline \
        -r "name = \"${DEP_NAME}\",
    artifact = \"\$2:${NEW_VERSION}\",
    sha1 = \"${NEW_SHA}\"," \
            "name = \"${DEP_NAME}\",\$\n(.*)artifact = \"(.*):([0-9a-zA-Z\\.]*)\",\$\n(.*)sha1 = \"([0-9a-f]*)\"," WORKSPACE > WORKSPACE.tmp

    mv WORKSPACE.tmp WORKSPACE

    echo "Successfully updated ${DEP_NAME} to ${NEW_VERSION} ( sha1 = $NEW_SHA )"
done <<< "$DEPS"
