package cobra

import (
	"fmt"
	"strings"
)

type PositionalArgs func(cmd *Command, args []string) error

func legacyArgs(cmd *Command, args []string) error {

	if !cmd.HasSubCommands() {
		return nil
	}

	if !cmd.HasParent() && len(args) > 0 {
		return fmt.Errorf("unknown command %q for %q%s", args[0], cmd.CommandPath(), cmd.findSuggestions(args[0]))
	}
	return nil
}

func NoArgs(cmd *Command, args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("unknown command %q for %q", args[0], cmd.CommandPath())
	}
	return nil
}

func OnlyValidArgs(cmd *Command, args []string) error {
	if len(cmd.ValidArgs) > 0 {

		var validArgs []string
		for _, v := range cmd.ValidArgs {
			validArgs = append(validArgs, strings.Split(v, "\t")[0])
		}

		for _, v := range args {
			if !stringInSlice(v, validArgs) {
				return fmt.Errorf("invalid argument %q for %q%s", v, cmd.CommandPath(), cmd.findSuggestions(args[0]))
			}
		}
	}
	return nil
}

func ArbitraryArgs(cmd *Command, args []string) error {
	return nil
}

func MinimumNArgs(n int) PositionalArgs {
	return func(cmd *Command, args []string) error {
		if len(args) < n {
			return fmt.Errorf("requires at least %d arg(s), only received %d", n, len(args))
		}
		return nil
	}
}

func MaximumNArgs(n int) PositionalArgs {
	return func(cmd *Command, args []string) error {
		if len(args) > n {
			return fmt.Errorf("accepts at most %d arg(s), received %d", n, len(args))
		}
		return nil
	}
}

func ExactArgs(n int) PositionalArgs {
	return func(cmd *Command, args []string) error {
		if len(args) != n {
			return fmt.Errorf("accepts %d arg(s), received %d", n, len(args))
		}
		return nil
	}
}

func ExactValidArgs(n int) PositionalArgs {
	return func(cmd *Command, args []string) error {
		if err := ExactArgs(n)(cmd, args); err != nil {
			return err
		}
		return OnlyValidArgs(cmd, args)
	}
}

func RangeArgs(min int, max int) PositionalArgs {
	return func(cmd *Command, args []string) error {
		if len(args) < min || len(args) > max {
			return fmt.Errorf("accepts between %d and %d arg(s), received %d", min, max, len(args))
		}
		return nil
	}
}

func MatchAll(pargs ...PositionalArgs) PositionalArgs {
	return func(cmd *Command, args []string) error {
		for _, parg := range pargs {
			if err := parg(cmd, args); err != nil {
				return err
			}
		}
		return nil
	}
}
