package service

import (
	"github.com/specgen-io/specgen-golang/v2/goven/generator"
	"github.com/specgen-io/specgen-golang/v2/goven/openapi"
	"github.com/specgen-io/specgen-golang/v2/goven/spec"
	"github.com/specgen-io/specgen-golang/v2/models"
	"github.com/specgen-io/specgen-golang/v2/module"
	"github.com/specgen-io/specgen-golang/v2/types"
)

func GenerateService(specification *spec.Spec, moduleName string, swaggerPath string, generatePath string, servicesPath string) *generator.Sources {
	sources := generator.NewSources()

	modules := models.NewModules(moduleName, generatePath, specification)
	serviceGenerator := NewGenerator(modules)

	rootModule := module.New(moduleName, generatePath)
	sources.AddGenerated(serviceGenerator.Service.GenerateSpecRouting(specification, rootModule))

	sources.AddGenerated(serviceGenerator.Models.EnumsHelperFunctions())

	emptyModule := rootModule.Submodule("empty")
	sources.AddGenerated(types.GenerateEmpty(emptyModule))

	paramsParserModule := rootModule.Submodule("paramsparser")
	sources.AddGenerated(generateParamsParser(paramsParserModule))

	respondModule := rootModule.Submodule("respond")
	sources.AddGenerated(generateRespondFunctions(respondModule))

	errorsModule := rootModule.Submodule("httperrors")
	errorsModelsModule := errorsModule.Submodule(types.ErrorsModelsPackage)
	sources.AddGenerated(serviceGenerator.Models.ErrorModels(specification.HttpErrors))
	sources.AddGeneratedAll(serviceGenerator.Service.HttpErrors(errorsModule, errorsModelsModule, paramsParserModule, respondModule, &specification.HttpErrors.Responses))

	contentTypeModule := rootModule.Submodule("contenttype")
	sources.AddGenerated(serviceGenerator.Service.CheckContentType(contentTypeModule, errorsModule, errorsModelsModule))

	for _, version := range specification.Versions {
		versionModule := rootModule.Submodule(version.Name.FlatCase())
		modelsModule := versionModule.Submodule(types.VersionModelsPackage)
		routingModule := versionModule.Submodule("routing")

		sources.AddGeneratedAll(serviceGenerator.Service.GenerateRoutings(&version, versionModule, routingModule, contentTypeModule, errorsModule, errorsModelsModule, modelsModule, paramsParserModule, respondModule))
		sources.AddGeneratedAll(serviceGenerator.generateServiceInterfaces(&version, versionModule, modelsModule, errorsModelsModule, emptyModule))
		sources.AddGenerated(serviceGenerator.Models.Models(&version))
	}

	if swaggerPath != "" {
		sources.AddGenerated(openapi.GenerateOpenapi(specification, swaggerPath))
	}

	if servicesPath != "" {
		rootServicesModule := module.New(moduleName, servicesPath)
		for _, version := range specification.Versions {
			versionImplementationsModule := rootServicesModule.Submodule(version.Name.FlatCase())
			versionModule := rootModule.Submodule(version.Name.FlatCase())
			modelsModule := versionModule.Submodule(types.VersionModelsPackage)
			sources.AddScaffoldedAll(serviceGenerator.generateServiceImplementations(&version, versionModule, modelsModule, versionImplementationsModule))
		}
	}

	return sources
}
