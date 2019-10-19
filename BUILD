load("@io_bazel_rules_go//go:def.bzl", "go_binary", "go_library", "go_test")
load("@bazel_gazelle//:def.bzl", "gazelle")

# gazelle:prefix github.com/zegl/bazel_dependency_tools
gazelle(name = "gazelle")

go_library(
    name = "go_default_library",
    srcs = ["app.go"],
    importpath = "github.com/zegl/bazel_dependency_tools",
    visibility = ["//visibility:private"],
    deps = [
        "//internal/github:go_default_library",
        "//maven_jar:go_default_library",
        "//parse:go_default_library",
        "@com_github_google_go_github_v28//github:go_default_library",
        "@org_golang_x_oauth2//:go_default_library",
    ],
)

go_binary(
    name = "bazel_dependency_tools",
    data = ["WORKSPACE"] + glob(["testdata/**"]),
    embed = [":go_default_library"],
    visibility = ["//visibility:public"],
)

go_test(
    name = "go_default_test",
    srcs = ["parser_test.go"],
    data = glob(["testdata/**"]),
    embed = [":go_default_library"],
    deps = [
        "//http_archive:go_default_library",
        "//internal:go_default_library",
        "//internal/github:go_default_library",
        "//parse:go_default_library",
        "@com_github_blang_semver//:go_default_library",
        "@com_github_stretchr_testify//assert:go_default_library",
    ],
)
