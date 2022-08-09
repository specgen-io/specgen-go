package cobra

import (
	"fmt"
	"os"
	"strings"
)

const (
	activeHelpMarker	= "_activeHelp_ "

	activeHelpEnvVarSuffix	= "_ACTIVE_HELP"
	activeHelpGlobalEnvVar	= "COBRA_ACTIVE_HELP"
	activeHelpGlobalDisable	= "0"
)

func AppendActiveHelp(compArray []string, activeHelpStr string) []string {
	return append(compArray, fmt.Sprintf("%s%s", activeHelpMarker, activeHelpStr))
}

func GetActiveHelpConfig(cmd *Command) string {
	activeHelpCfg := os.Getenv(activeHelpGlobalEnvVar)
	if activeHelpCfg != activeHelpGlobalDisable {
		activeHelpCfg = os.Getenv(activeHelpEnvVar(cmd.Root().Name()))
	}
	return activeHelpCfg
}

func activeHelpEnvVar(name string) string {

	activeHelpEnvVar := strings.ToUpper(fmt.Sprintf("%s%s", name, activeHelpEnvVarSuffix))
	return strings.ReplaceAll(activeHelpEnvVar, "-", "_")
}
