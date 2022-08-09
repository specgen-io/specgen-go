package cobra

import (
	"fmt"
	"sort"
	"strings"

	flag "github.com/specgen-io/specgen-golang/v2/goven/github.com/spf13/pflag"
)

const (
	requiredAsGroup		= "cobra_annotation_required_if_others_set"
	mutuallyExclusive	= "cobra_annotation_mutually_exclusive"
)

func (c *Command) MarkFlagsRequiredTogether(flagNames ...string) {
	c.mergePersistentFlags()
	for _, v := range flagNames {
		f := c.Flags().Lookup(v)
		if f == nil {
			panic(fmt.Sprintf("Failed to find flag %q and mark it as being required in a flag group", v))
		}
		if err := c.Flags().SetAnnotation(v, requiredAsGroup, append(f.Annotations[requiredAsGroup], strings.Join(flagNames, " "))); err != nil {

			panic(err)
		}
	}
}

func (c *Command) MarkFlagsMutuallyExclusive(flagNames ...string) {
	c.mergePersistentFlags()
	for _, v := range flagNames {
		f := c.Flags().Lookup(v)
		if f == nil {
			panic(fmt.Sprintf("Failed to find flag %q and mark it as being in a mutually exclusive flag group", v))
		}

		if err := c.Flags().SetAnnotation(v, mutuallyExclusive, append(f.Annotations[mutuallyExclusive], strings.Join(flagNames, " "))); err != nil {
			panic(err)
		}
	}
}

func (c *Command) validateFlagGroups() error {
	if c.DisableFlagParsing {
		return nil
	}

	flags := c.Flags()

	groupStatus := map[string]map[string]bool{}
	mutuallyExclusiveGroupStatus := map[string]map[string]bool{}
	flags.VisitAll(func(pflag *flag.Flag) {
		processFlagForGroupAnnotation(flags, pflag, requiredAsGroup, groupStatus)
		processFlagForGroupAnnotation(flags, pflag, mutuallyExclusive, mutuallyExclusiveGroupStatus)
	})

	if err := validateRequiredFlagGroups(groupStatus); err != nil {
		return err
	}
	if err := validateExclusiveFlagGroups(mutuallyExclusiveGroupStatus); err != nil {
		return err
	}
	return nil
}

func hasAllFlags(fs *flag.FlagSet, flagnames ...string) bool {
	for _, fname := range flagnames {
		f := fs.Lookup(fname)
		if f == nil {
			return false
		}
	}
	return true
}

func processFlagForGroupAnnotation(flags *flag.FlagSet, pflag *flag.Flag, annotation string, groupStatus map[string]map[string]bool) {
	groupInfo, found := pflag.Annotations[annotation]
	if found {
		for _, group := range groupInfo {
			if groupStatus[group] == nil {
				flagnames := strings.Split(group, " ")

				if !hasAllFlags(flags, flagnames...) {
					continue
				}

				groupStatus[group] = map[string]bool{}
				for _, name := range flagnames {
					groupStatus[group][name] = false
				}
			}

			groupStatus[group][pflag.Name] = pflag.Changed
		}
	}
}

func validateRequiredFlagGroups(data map[string]map[string]bool) error {
	keys := sortedKeys(data)
	for _, flagList := range keys {
		flagnameAndStatus := data[flagList]

		unset := []string{}
		for flagname, isSet := range flagnameAndStatus {
			if !isSet {
				unset = append(unset, flagname)
			}
		}
		if len(unset) == len(flagnameAndStatus) || len(unset) == 0 {
			continue
		}

		sort.Strings(unset)
		return fmt.Errorf("if any flags in the group [%v] are set they must all be set; missing %v", flagList, unset)
	}

	return nil
}

func validateExclusiveFlagGroups(data map[string]map[string]bool) error {
	keys := sortedKeys(data)
	for _, flagList := range keys {
		flagnameAndStatus := data[flagList]
		var set []string
		for flagname, isSet := range flagnameAndStatus {
			if isSet {
				set = append(set, flagname)
			}
		}
		if len(set) == 0 || len(set) == 1 {
			continue
		}

		sort.Strings(set)
		return fmt.Errorf("if any flags in the group [%v] are set none of the others can be; %v were all set", flagList, set)
	}
	return nil
}

func sortedKeys(m map[string]map[string]bool) []string {
	keys := make([]string, len(m))
	i := 0
	for k := range m {
		keys[i] = k
		i++
	}
	sort.Strings(keys)
	return keys
}

func (c *Command) enforceFlagGroupsForCompletion() {
	if c.DisableFlagParsing {
		return
	}

	flags := c.Flags()
	groupStatus := map[string]map[string]bool{}
	mutuallyExclusiveGroupStatus := map[string]map[string]bool{}
	c.Flags().VisitAll(func(pflag *flag.Flag) {
		processFlagForGroupAnnotation(flags, pflag, requiredAsGroup, groupStatus)
		processFlagForGroupAnnotation(flags, pflag, mutuallyExclusive, mutuallyExclusiveGroupStatus)
	})

	for flagList, flagnameAndStatus := range groupStatus {
		for _, isSet := range flagnameAndStatus {
			if isSet {

				for _, fName := range strings.Split(flagList, " ") {
					_ = c.MarkFlagRequired(fName)
				}
			}
		}
	}

	for flagList, flagnameAndStatus := range mutuallyExclusiveGroupStatus {
		for flagName, isSet := range flagnameAndStatus {
			if isSet {

				for _, fName := range strings.Split(flagList, " ") {
					if fName != flagName {
						flag := c.Flags().Lookup(fName)
						flag.Hidden = true
					}
				}
			}
		}
	}
}
