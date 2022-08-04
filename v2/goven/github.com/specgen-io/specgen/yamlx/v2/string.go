package yamlx

import (
	"github.com/specgen-io/specgen-go/v2/github.com/specgen-io/specgen-go/v2/goven/gopkg.in/specgen-io/yaml.v3"
)

func String(value string) yaml.Node {
	return yaml.Node{Kind: yaml.ScalarNode, Value: value}
}
