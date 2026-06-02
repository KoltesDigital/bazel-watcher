package lifecycle_hooks

import (
	"testing"

	"github.com/bazelbuild/bazel-watcher/internal/e2e"
)

const mainFiles = `
-- BUILD.bazel --
sh_binary(
  name = "test",
  srcs = ["test.sh"],
)
sh_binary(
  name = "failure",
  srcs = ["failure.sh"],
)
-- test.sh --
-- failure.sh --
exit 42
`

func TestMain(m *testing.M) {
	e2e.TestMain(m, e2e.Args{
		Main: mainFiles,
	})
}

func TestExitCode(t *testing.T) {
	ibazel := e2e.SetUp(t)
	defer ibazel.Kill()

	ibazel.Run([]string{}, "//:test")
	ibazel.ExpectIBazelError("Exited \\(0\\)")
}

func TestExitCodeFailure(t *testing.T) {
	ibazel := e2e.SetUp(t)
	defer ibazel.Kill()

	ibazel.Run([]string{}, "//:failure")
	ibazel.ExpectIBazelError("Exited \\(42\\)")
}
