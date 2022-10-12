package models

import (
	"github.com/specgen-io/specgen-golang/v2/goven/generator"
	"github.com/specgen-io/specgen-golang/v2/goven/spec"
	"github.com/specgen-io/specgen-golang/v2/module"
)

type Generator interface {
	GenerateVersionModels(models []*spec.NamedModel, module, enumsModule module.Module) *generator.CodeFile
	EnumValuesStrings(model *spec.NamedModel) string
	GenerateEnumsHelperFunctions(module module.Module) *generator.CodeFile
}

func NewGenerator() Generator {
	return NewEncodingJsonGenerator()
}
