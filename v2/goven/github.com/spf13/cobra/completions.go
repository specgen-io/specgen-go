package cobra

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/specgen-io/specgen-golang/v2/goven/github.com/spf13/pflag"
)

const (
	ShellCompRequestCmd	= "__complete"

	ShellCompNoDescRequestCmd	= "__completeNoDesc"
)

var flagCompletionFunctions = map[*pflag.Flag]func(cmd *Command, args []string, toComplete string) ([]string, ShellCompDirective){}

var flagCompletionMutex = &sync.RWMutex{}

type ShellCompDirective int

type flagCompError struct {
	subCommand	string
	flagName	string
}

func (e *flagCompError) Error() string {
	return "Subcommand '" + e.subCommand + "' does not support flag '" + e.flagName + "'"
}

const (
	ShellCompDirectiveError	ShellCompDirective	= 1 << iota

	ShellCompDirectiveNoSpace

	ShellCompDirectiveNoFileComp

	ShellCompDirectiveFilterFileExt

	ShellCompDirectiveFilterDirs

	shellCompDirectiveMaxValue

	ShellCompDirectiveDefault	ShellCompDirective	= 0
)

const (
	compCmdName			= "completion"
	compCmdNoDescFlagName		= "no-descriptions"
	compCmdNoDescFlagDesc		= "disable completion descriptions"
	compCmdNoDescFlagDefault	= false
)

type CompletionOptions struct {
	DisableDefaultCmd	bool

	DisableNoDescFlag	bool

	DisableDescriptions	bool

	HiddenDefaultCmd	bool
}

func NoFileCompletions(cmd *Command, args []string, toComplete string) ([]string, ShellCompDirective) {
	return nil, ShellCompDirectiveNoFileComp
}

func FixedCompletions(choices []string, directive ShellCompDirective) func(cmd *Command, args []string, toComplete string) ([]string, ShellCompDirective) {
	return func(cmd *Command, args []string, toComplete string) ([]string, ShellCompDirective) {
		return choices, directive
	}
}

func (c *Command) RegisterFlagCompletionFunc(flagName string, f func(cmd *Command, args []string, toComplete string) ([]string, ShellCompDirective)) error {
	flag := c.Flag(flagName)
	if flag == nil {
		return fmt.Errorf("RegisterFlagCompletionFunc: flag '%s' does not exist", flagName)
	}
	flagCompletionMutex.Lock()
	defer flagCompletionMutex.Unlock()

	if _, exists := flagCompletionFunctions[flag]; exists {
		return fmt.Errorf("RegisterFlagCompletionFunc: flag '%s' already registered", flagName)
	}
	flagCompletionFunctions[flag] = f
	return nil
}

func (d ShellCompDirective) string() string {
	var directives []string
	if d&ShellCompDirectiveError != 0 {
		directives = append(directives, "ShellCompDirectiveError")
	}
	if d&ShellCompDirectiveNoSpace != 0 {
		directives = append(directives, "ShellCompDirectiveNoSpace")
	}
	if d&ShellCompDirectiveNoFileComp != 0 {
		directives = append(directives, "ShellCompDirectiveNoFileComp")
	}
	if d&ShellCompDirectiveFilterFileExt != 0 {
		directives = append(directives, "ShellCompDirectiveFilterFileExt")
	}
	if d&ShellCompDirectiveFilterDirs != 0 {
		directives = append(directives, "ShellCompDirectiveFilterDirs")
	}
	if len(directives) == 0 {
		directives = append(directives, "ShellCompDirectiveDefault")
	}

	if d >= shellCompDirectiveMaxValue {
		return fmt.Sprintf("ERROR: unexpected ShellCompDirective value: %d", d)
	}
	return strings.Join(directives, ", ")
}

func (c *Command) initCompleteCmd(args []string) {
	completeCmd := &Command{
		Use:			fmt.Sprintf("%s [command-line]", ShellCompRequestCmd),
		Aliases:		[]string{ShellCompNoDescRequestCmd},
		DisableFlagsInUseLine:	true,
		Hidden:			true,
		DisableFlagParsing:	true,
		Args:			MinimumNArgs(1),
		Short:			"Request shell completion choices for the specified command-line",
		Long: fmt.Sprintf("%[2]s is a special command that is used by the shell completion logic\n%[1]s",
			"to request completion choices for the specified command-line.", ShellCompRequestCmd),
		Run: func(cmd *Command, args []string) {
			finalCmd, completions, directive, err := cmd.getCompletions(args)
			if err != nil {
				CompErrorln(err.Error())

			}

			noDescriptions := (cmd.CalledAs() == ShellCompNoDescRequestCmd)
			for _, comp := range completions {
				if GetActiveHelpConfig(finalCmd) == activeHelpGlobalDisable {

					if strings.HasPrefix(comp, activeHelpMarker) {
						continue
					}
				}
				if noDescriptions {

					comp = strings.Split(comp, "\t")[0]
				}

				comp = strings.Split(comp, "\n")[0]

				comp = strings.TrimSpace(comp)

				fmt.Fprintln(finalCmd.OutOrStdout(), comp)
			}

			fmt.Fprintf(finalCmd.OutOrStdout(), ":%d\n", directive)

			fmt.Fprintf(finalCmd.ErrOrStderr(), "Completion ended with directive: %s\n", directive.string())
		},
	}
	c.AddCommand(completeCmd)
	subCmd, _, err := c.Find(args)
	if err != nil || subCmd.Name() != ShellCompRequestCmd {

		c.RemoveCommand(completeCmd)
	}
}

func (c *Command) getCompletions(args []string) (*Command, []string, ShellCompDirective, error) {

	toComplete := args[len(args)-1]
	trimmedArgs := args[:len(args)-1]

	var finalCmd *Command
	var finalArgs []string
	var err error

	if c.Root().TraverseChildren {
		finalCmd, finalArgs, err = c.Root().Traverse(trimmedArgs)
	} else {

		rootCmd := c.Root()
		if len(rootCmd.Commands()) == 1 {
			rootCmd.RemoveCommand(c)
		}

		finalCmd, finalArgs, err = rootCmd.Find(trimmedArgs)
	}
	if err != nil {

		return c, []string{}, ShellCompDirectiveDefault, fmt.Errorf("Unable to find a command for arguments: %v", trimmedArgs)
	}
	finalCmd.ctx = c.ctx

	flag, finalArgs, toComplete, flagErr := checkIfFlagCompletion(finalCmd, finalArgs, toComplete)

	flagCompletion := true
	_ = finalCmd.ParseFlags(append(finalArgs, "--"))
	newArgCount := finalCmd.Flags().NArg()

	if err = finalCmd.ParseFlags(finalArgs); err != nil {
		return finalCmd, []string{}, ShellCompDirectiveDefault, fmt.Errorf("Error while parsing flags from args %v: %s", finalArgs, err.Error())
	}

	realArgCount := finalCmd.Flags().NArg()
	if newArgCount > realArgCount {

		flagCompletion = false
	}

	if flagErr != nil {

		if _, ok := flagErr.(*flagCompError); !(ok && !flagCompletion) {
			return finalCmd, []string{}, ShellCompDirectiveDefault, flagErr
		}
	}

	if !finalCmd.DisableFlagParsing {
		finalArgs = finalCmd.Flags().Args()
	}

	if flag != nil && flagCompletion {

		if validExts, present := flag.Annotations[BashCompFilenameExt]; present {
			if len(validExts) != 0 {

				return finalCmd, validExts, ShellCompDirectiveFilterFileExt, nil
			}

		}

		if subDir, present := flag.Annotations[BashCompSubdirsInDir]; present {
			if len(subDir) == 1 {

				return finalCmd, subDir, ShellCompDirectiveFilterDirs, nil
			}

			return finalCmd, []string{}, ShellCompDirectiveFilterDirs, nil
		}
	}

	var completions []string
	var directive ShellCompDirective

	finalCmd.enforceFlagGroupsForCompletion()

	if flag == nil && len(toComplete) > 0 && toComplete[0] == '-' && !strings.Contains(toComplete, "=") && flagCompletion {

		completions = completeRequireFlags(finalCmd, toComplete)

		if len(completions) == 0 {
			doCompleteFlags := func(flag *pflag.Flag) {
				if !flag.Changed ||
					strings.Contains(flag.Value.Type(), "Slice") ||
					strings.Contains(flag.Value.Type(), "Array") {

					completions = append(completions, getFlagNameCompletions(flag, toComplete)...)
				}
			}

			finalCmd.InheritedFlags().VisitAll(func(flag *pflag.Flag) {
				doCompleteFlags(flag)
			})
			finalCmd.NonInheritedFlags().VisitAll(func(flag *pflag.Flag) {
				doCompleteFlags(flag)
			})
		}

		directive = ShellCompDirectiveNoFileComp
		if len(completions) == 1 && strings.HasSuffix(completions[0], "=") {

			directive = ShellCompDirectiveNoSpace
		}

		if !finalCmd.DisableFlagParsing {

			return finalCmd, completions, directive, nil
		}
	} else {
		directive = ShellCompDirectiveDefault
		if flag == nil {
			foundLocalNonPersistentFlag := false

			if !finalCmd.Root().TraverseChildren {

				localNonPersistentFlags := finalCmd.LocalNonPersistentFlags()
				finalCmd.NonInheritedFlags().VisitAll(func(flag *pflag.Flag) {
					if localNonPersistentFlags.Lookup(flag.Name) != nil && flag.Changed {
						foundLocalNonPersistentFlag = true
					}
				})
			}

			if len(finalArgs) == 0 && !foundLocalNonPersistentFlag {

				for _, subCmd := range finalCmd.Commands() {
					if subCmd.IsAvailableCommand() || subCmd == finalCmd.helpCommand {
						if strings.HasPrefix(subCmd.Name(), toComplete) {
							completions = append(completions, fmt.Sprintf("%s\t%s", subCmd.Name(), subCmd.Short))
						}
						directive = ShellCompDirectiveNoFileComp
					}
				}
			}

			completions = append(completions, completeRequireFlags(finalCmd, toComplete)...)

			if len(finalCmd.ValidArgs) > 0 {
				if len(finalArgs) == 0 {

					for _, validArg := range finalCmd.ValidArgs {
						if strings.HasPrefix(validArg, toComplete) {
							completions = append(completions, validArg)
						}
					}
					directive = ShellCompDirectiveNoFileComp

					if len(completions) == 0 {
						for _, argAlias := range finalCmd.ArgAliases {
							if strings.HasPrefix(argAlias, toComplete) {
								completions = append(completions, argAlias)
							}
						}
					}
				}

				return finalCmd, completions, directive, nil
			}

		}
	}

	var completionFn func(cmd *Command, args []string, toComplete string) ([]string, ShellCompDirective)
	if flag != nil && flagCompletion {
		flagCompletionMutex.RLock()
		completionFn = flagCompletionFunctions[flag]
		flagCompletionMutex.RUnlock()
	} else {
		completionFn = finalCmd.ValidArgsFunction
	}
	if completionFn != nil {

		var comps []string
		comps, directive = completionFn(finalCmd, finalArgs, toComplete)
		completions = append(completions, comps...)
	}

	return finalCmd, completions, directive, nil
}

func getFlagNameCompletions(flag *pflag.Flag, toComplete string) []string {
	if nonCompletableFlag(flag) {
		return []string{}
	}

	var completions []string
	flagName := "--" + flag.Name
	if strings.HasPrefix(flagName, toComplete) {

		completions = append(completions, fmt.Sprintf("%s\t%s", flagName, flag.Usage))

	}

	flagName = "-" + flag.Shorthand
	if len(flag.Shorthand) > 0 && strings.HasPrefix(flagName, toComplete) {
		completions = append(completions, fmt.Sprintf("%s\t%s", flagName, flag.Usage))
	}

	return completions
}

func completeRequireFlags(finalCmd *Command, toComplete string) []string {
	var completions []string

	doCompleteRequiredFlags := func(flag *pflag.Flag) {
		if _, present := flag.Annotations[BashCompOneRequiredFlag]; present {
			if !flag.Changed {

				completions = append(completions, getFlagNameCompletions(flag, toComplete)...)
			}
		}
	}

	finalCmd.InheritedFlags().VisitAll(func(flag *pflag.Flag) {
		doCompleteRequiredFlags(flag)
	})
	finalCmd.NonInheritedFlags().VisitAll(func(flag *pflag.Flag) {
		doCompleteRequiredFlags(flag)
	})

	return completions
}

func checkIfFlagCompletion(finalCmd *Command, args []string, lastArg string) (*pflag.Flag, []string, string, error) {
	if finalCmd.DisableFlagParsing {

		return nil, args, lastArg, nil
	}

	var flagName string
	trimmedArgs := args
	flagWithEqual := false
	orgLastArg := lastArg

	if len(lastArg) > 0 && lastArg[0] == '-' {
		if index := strings.Index(lastArg, "="); index >= 0 {

			if strings.HasPrefix(lastArg[:index], "--") {

				flagName = lastArg[2:index]
			} else {

				flagName = lastArg[index-1 : index]
			}
			lastArg = lastArg[index+1:]
			flagWithEqual = true
		} else {

			return nil, args, lastArg, nil
		}
	}

	if len(flagName) == 0 {
		if len(args) > 0 {
			prevArg := args[len(args)-1]
			if isFlagArg(prevArg) {

				if index := strings.Index(prevArg, "="); index < 0 {
					if strings.HasPrefix(prevArg, "--") {

						flagName = prevArg[2:]
					} else {

						flagName = prevArg[len(prevArg)-1:]
					}

					trimmedArgs = args[:len(args)-1]
				}
			}
		}
	}

	if len(flagName) == 0 {

		return nil, trimmedArgs, lastArg, nil
	}

	flag := findFlag(finalCmd, flagName)
	if flag == nil {

		return nil, args, orgLastArg, &flagCompError{subCommand: finalCmd.Name(), flagName: flagName}
	}

	if !flagWithEqual {
		if len(flag.NoOptDefVal) != 0 {

			trimmedArgs = args
			flag = nil
		}
	}

	return flag, trimmedArgs, lastArg, nil
}

func (c *Command) initDefaultCompletionCmd() {
	if c.CompletionOptions.DisableDefaultCmd || !c.HasSubCommands() {
		return
	}

	for _, cmd := range c.commands {
		if cmd.Name() == compCmdName || cmd.HasAlias(compCmdName) {

			return
		}
	}

	haveNoDescFlag := !c.CompletionOptions.DisableNoDescFlag && !c.CompletionOptions.DisableDescriptions

	completionCmd := &Command{
		Use:	compCmdName,
		Short:	"Generate the autocompletion script for the specified shell",
		Long: fmt.Sprintf(`Generate the autocompletion script for %[1]s for the specified shell.
See each sub-command's help for details on how to use the generated script.
`, c.Root().Name()),
		Args:			NoArgs,
		ValidArgsFunction:	NoFileCompletions,
		Hidden:			c.CompletionOptions.HiddenDefaultCmd,
	}
	c.AddCommand(completionCmd)

	out := c.OutOrStdout()
	noDesc := c.CompletionOptions.DisableDescriptions
	shortDesc := "Generate the autocompletion script for %s"
	bash := &Command{
		Use:	"bash",
		Short:	fmt.Sprintf(shortDesc, "bash"),
		Long: fmt.Sprintf(`Generate the autocompletion script for the bash shell.

This script depends on the 'bash-completion' package.
If it is not installed already, you can install it via your OS's package manager.

To load completions in your current shell session:

	source <(%[1]s completion bash)

To load completions for every new session, execute once:

#### Linux:

	%[1]s completion bash > /etc/bash_completion.d/%[1]s

#### macOS:

	%[1]s completion bash > $(brew --prefix)/etc/bash_completion.d/%[1]s

You will need to start a new shell for this setup to take effect.
`, c.Root().Name()),
		Args:			NoArgs,
		DisableFlagsInUseLine:	true,
		ValidArgsFunction:	NoFileCompletions,
		RunE: func(cmd *Command, args []string) error {
			return cmd.Root().GenBashCompletionV2(out, !noDesc)
		},
	}
	if haveNoDescFlag {
		bash.Flags().BoolVar(&noDesc, compCmdNoDescFlagName, compCmdNoDescFlagDefault, compCmdNoDescFlagDesc)
	}

	zsh := &Command{
		Use:	"zsh",
		Short:	fmt.Sprintf(shortDesc, "zsh"),
		Long: fmt.Sprintf(`Generate the autocompletion script for the zsh shell.

If shell completion is not already enabled in your environment you will need
to enable it.  You can execute the following once:

	echo "autoload -U compinit; compinit" >> ~/.zshrc

To load completions in your current shell session:

	source <(%[1]s completion zsh); compdef _%[1]s %[1]s

To load completions for every new session, execute once:

#### Linux:

	%[1]s completion zsh > "${fpath[1]}/_%[1]s"

#### macOS:

	%[1]s completion zsh > $(brew --prefix)/share/zsh/site-functions/_%[1]s

You will need to start a new shell for this setup to take effect.
`, c.Root().Name()),
		Args:			NoArgs,
		ValidArgsFunction:	NoFileCompletions,
		RunE: func(cmd *Command, args []string) error {
			if noDesc {
				return cmd.Root().GenZshCompletionNoDesc(out)
			}
			return cmd.Root().GenZshCompletion(out)
		},
	}
	if haveNoDescFlag {
		zsh.Flags().BoolVar(&noDesc, compCmdNoDescFlagName, compCmdNoDescFlagDefault, compCmdNoDescFlagDesc)
	}

	fish := &Command{
		Use:	"fish",
		Short:	fmt.Sprintf(shortDesc, "fish"),
		Long: fmt.Sprintf(`Generate the autocompletion script for the fish shell.

To load completions in your current shell session:

	%[1]s completion fish | source

To load completions for every new session, execute once:

	%[1]s completion fish > ~/.config/fish/completions/%[1]s.fish

You will need to start a new shell for this setup to take effect.
`, c.Root().Name()),
		Args:			NoArgs,
		ValidArgsFunction:	NoFileCompletions,
		RunE: func(cmd *Command, args []string) error {
			return cmd.Root().GenFishCompletion(out, !noDesc)
		},
	}
	if haveNoDescFlag {
		fish.Flags().BoolVar(&noDesc, compCmdNoDescFlagName, compCmdNoDescFlagDefault, compCmdNoDescFlagDesc)
	}

	powershell := &Command{
		Use:	"powershell",
		Short:	fmt.Sprintf(shortDesc, "powershell"),
		Long: fmt.Sprintf(`Generate the autocompletion script for powershell.

To load completions in your current shell session:

	%[1]s completion powershell | Out-String | Invoke-Expression

To load completions for every new session, add the output of the above command
to your powershell profile.
`, c.Root().Name()),
		Args:			NoArgs,
		ValidArgsFunction:	NoFileCompletions,
		RunE: func(cmd *Command, args []string) error {
			if noDesc {
				return cmd.Root().GenPowerShellCompletion(out)
			}
			return cmd.Root().GenPowerShellCompletionWithDesc(out)

		},
	}
	if haveNoDescFlag {
		powershell.Flags().BoolVar(&noDesc, compCmdNoDescFlagName, compCmdNoDescFlagDefault, compCmdNoDescFlagDesc)
	}

	completionCmd.AddCommand(bash, zsh, fish, powershell)
}

func findFlag(cmd *Command, name string) *pflag.Flag {
	flagSet := cmd.Flags()
	if len(name) == 1 {

		if short := flagSet.ShorthandLookup(name); short != nil {
			name = short.Name
		} else {
			set := cmd.InheritedFlags()
			if short = set.ShorthandLookup(name); short != nil {
				name = short.Name
			} else {
				return nil
			}
		}
	}
	return cmd.Flag(name)
}

func CompDebug(msg string, printToStdErr bool) {
	msg = fmt.Sprintf("[Debug] %s", msg)

	if path := os.Getenv("BASH_COMP_DEBUG_FILE"); path != "" {
		f, err := os.OpenFile(path,
			os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err == nil {
			defer f.Close()
			WriteStringAndCheck(f, msg)
		}
	}

	if printToStdErr {

		fmt.Fprint(os.Stderr, msg)
	}
}

func CompDebugln(msg string, printToStdErr bool) {
	CompDebug(fmt.Sprintf("%s\n", msg), printToStdErr)
}

func CompError(msg string) {
	msg = fmt.Sprintf("[Error] %s", msg)
	CompDebug(msg, true)
}

func CompErrorln(msg string) {
	CompError(fmt.Sprintf("%s\n", msg))
}
