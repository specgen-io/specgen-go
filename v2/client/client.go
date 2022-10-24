package client

import (
	"github.com/specgen-io/specgen-golang/v2/goven/generator"
	"github.com/specgen-io/specgen-golang/v2/goven/spec"
)

func GenerateClient(specification *spec.Spec, moduleName string, generatePath string) *generator.Sources {
	sources := generator.NewSources()

	modules := NewModules(moduleName, generatePath, specification)
	generator := NewGenerator(modules)

	sources.AddGeneratedAll(generator.AllStaticFiles())

	sources.AddGenerated(generator.GenerateErrorModels(specification.HttpErrors))
	sources.AddGeneratedAll(generator.Errors(&specification.HttpErrors.Responses))

	for _, version := range specification.Versions {
		sources.AddGenerated(generator.GenerateVersionModels(&version))
		sources.AddGeneratedAll(generator.GenerateClientsImplementations(&version))
	}
	return sources
}
