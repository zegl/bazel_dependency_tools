# bazel_dependency_tools

This project aims to create tools to help with dependency management in Bazel WORKSPACEs.

* `update-jar-dep.sh` - Update multiple `maven_jar` to the same version, automatically sets `sha1`.
* `maven-jar-sha1-to-sha256.sh` - Update `maven_jar`s that are using `sha1` (deprecated) to use `sha256`.
