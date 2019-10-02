# bazel_dependency_tools

This project aims to create tools to help with dependency management in Bazel WORKSPACEs.

`bazel_dependency_tools` aims to contain tools to help upgrading dependencies.

## Upgrades

The goal is that bazel_dependency_tools should be able to upgrade dependencies automatically, similar to what dependabot and similar tools can do.

| rule | status |
|------|--------|
| http_archive | ✅ |
| maven_jar | 🙅‍♂️ |
| git_repository | 🙅‍♂️ |
| http_jar | 🙅‍♂️ |
| rules_mvn_external | ❓ |
| go_repository  | ❓ |

* ✅ == implemented, supported
* 🙅‍♂ == not implemented, planned
* ❓ == not implemented, unplanned

## Hacks

These are deprecated, and will hopefully be re-implemented in the Go version.s

* `hack/update-jar-dep.sh` - Update multiple `maven_jar` to the same version, automatically sets `sha1`.
* `hack/maven-jar-sha1-to-sha256.sh` - Update `maven_jar`s that are using `sha1` (deprecated) to use `sha256`.
