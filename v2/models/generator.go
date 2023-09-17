package models

import (
	"fmt"
	"github.com/specgen-io/specgen-golang/v2/goven/generator"
	"github.com/specgen-io/specgen-golang/v2/goven/spec"
	"github.com/specgen-io/specgen-golang/v2/types"
)

type Generator interface {
	Models(version *spec.Version) []generator.CodeFile
	ErrorModels(httperrors *spec.HttpErrors) []generator.CodeFile
	EnumValuesStrings(model *spec.NamedModel) string
	EnumsHelperFunctions() *generator.CodeFile
}

func NewGenerator(jsonmode string, modules *Modules) Generator {
	types := types.NewTypes()

	if jsonmode == Strict {
		return NewEncodingJsonGenerator(types, modules, true)
	}
	if jsonmode == NonStrict {
		return NewEncodingJsonGenerator(types, modules, false)
	}

	panic(fmt.Sprintf(`Unknown jsonmode: %s`, jsonmode))
}

var Strict = "strict"
var NonStrict = "nonstrict"
