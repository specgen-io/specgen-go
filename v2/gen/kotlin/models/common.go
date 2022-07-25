package models

import (
	"github.com/specgen-io/specgen-go/v2/spec"
)

func oneOfItemClassName(item *spec.NamedDefinition) string {
	return item.Name.PascalCase()
}
