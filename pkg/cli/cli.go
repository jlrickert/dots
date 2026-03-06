package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/jlrickert/cli-toolkit/toolkit"
)

func Run(ctx context.Context, rt *toolkit.Runtime, args []string) (int, error) {
	if rt == nil {
		var err error
		rt, err = toolkit.NewRuntime()
		if err != nil {
			return 1, err
		}
	}
	if err := rt.Validate(); err != nil {
		return 1, err
	}

	streams := rt.Stream()
	deps := &Deps{
		Runtime:  rt,
		Shutdown: func() {},
	}
	cmd := NewRootCmd(deps)
	cmd.SetArgs(args)
	cmd.SetIn(streams.In)
	cmd.SetOut(streams.Out)
	cmd.SetErr(streams.Err)

	if err := cmd.ExecuteContext(ctx); err != nil {
		_, _ = fmt.Fprintf(streams.Err, "Error: %s\n", err)

		if errors.Is(err, context.Canceled) ||
			errors.Is(err, context.DeadlineExceeded) {
			return 130, err
		}
		return 1, err
	}
	return 0, nil
}
