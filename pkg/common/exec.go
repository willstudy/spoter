package common

import (
	"context"
	"os/exec"
)

func ExecCmd(ctx context.Context, args []string) (string, error) {
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)

	output, err := cmd.CombinedOutput()

	return string(output), err
}
