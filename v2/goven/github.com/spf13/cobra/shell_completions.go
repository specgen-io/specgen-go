package cobra

import (
	"github.com/specgen-io/specgen-golang/v2/goven/github.com/spf13/pflag"
)

func (c *Command) MarkFlagRequired(name string) error {
	return MarkFlagRequired(c.Flags(), name)
}

func (c *Command) MarkPersistentFlagRequired(name string) error {
	return MarkFlagRequired(c.PersistentFlags(), name)
}

func MarkFlagRequired(flags *pflag.FlagSet, name string) error {
	return flags.SetAnnotation(name, BashCompOneRequiredFlag, []string{"true"})
}

func (c *Command) MarkFlagFilename(name string, extensions ...string) error {
	return MarkFlagFilename(c.Flags(), name, extensions...)
}

func (c *Command) MarkFlagCustom(name string, f string) error {
	return MarkFlagCustom(c.Flags(), name, f)
}

func (c *Command) MarkPersistentFlagFilename(name string, extensions ...string) error {
	return MarkFlagFilename(c.PersistentFlags(), name, extensions...)
}

func MarkFlagFilename(flags *pflag.FlagSet, name string, extensions ...string) error {
	return flags.SetAnnotation(name, BashCompFilenameExt, extensions)
}

func MarkFlagCustom(flags *pflag.FlagSet, name string, f string) error {
	return flags.SetAnnotation(name, BashCompCustom, []string{f})
}

func (c *Command) MarkFlagDirname(name string) error {
	return MarkFlagDirname(c.Flags(), name)
}

func (c *Command) MarkPersistentFlagDirname(name string) error {
	return MarkFlagDirname(c.PersistentFlags(), name)
}

func MarkFlagDirname(flags *pflag.FlagSet, name string) error {
	return flags.SetAnnotation(name, BashCompSubdirsInDir, []string{})
}
