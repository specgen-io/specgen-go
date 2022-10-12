package models

import (
	"github.com/specgen-io/specgen-golang/v2/goven/generator"
	"github.com/specgen-io/specgen-golang/v2/goven/spec"
	"github.com/specgen-io/specgen-golang/v2/module"
	"github.com/specgen-io/specgen-golang/v2/types"
)

func GenerateModels(specification *spec.Spec, moduleName string, generatePath string) *generator.Sources {
	sources := generator.NewSources()

	generator := NewGenerator()

	rootModule := module.New(moduleName, generatePath)

	enumsModule := rootModule.Submodule("enums")
	sources.AddGenerated(generator.GenerateEnumsHelperFunctions(enumsModule))

	for _, version := range specification.Versions {
		versionModule := rootModule.Submodule(version.Name.FlatCase())
		modelsModule := versionModule.Submodule(types.VersionModelsPackage)
		sources.AddGenerated(generator.GenerateVersionModels(version.ResolvedModels, modelsModule, enumsModule))
	}
	return sources
}
