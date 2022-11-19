package models

import (
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

func NewGenerator(modules *Modules) Generator {
	types := types.NewTypes()
	return NewEncodingJsonGenerator(types, modules)
}
