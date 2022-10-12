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

	modelsGenerator := models.NewGenerator()

	rootModule := module.New(moduleName, generatePath)
	sources.AddGenerated(generateSpecRouting(specification, rootModule))

	enumsModule := rootModule.Submodule("enums")
	sources.AddGenerated(modelsGenerator.GenerateEnumsHelperFunctions(enumsModule))

	emptyModule := rootModule.Submodule("empty")
	sources.AddGenerated(types.GenerateEmpty(emptyModule))

	paramsParserModule := rootModule.Submodule("paramsparser")
	sources.AddGenerated(generateParamsParser(paramsParserModule))

	respondModule := rootModule.Submodule("respond")
	sources.AddGenerated(generateRespondFunctions(respondModule))

	errorsModule := rootModule.Submodule("httperrors")
	errorsModelsModule := errorsModule.Submodule("models")
	sources.AddGenerated(modelsGenerator.GenerateVersionModels(specification.HttpErrors.ResolvedModels, errorsModelsModule, enumsModule))
	sources.AddGeneratedAll(httpErrors(errorsModule, errorsModelsModule, paramsParserModule, respondModule, &specification.HttpErrors.Responses))

	contentTypeModule := rootModule.Submodule("contenttype")
	sources.AddGenerated(checkContentType(contentTypeModule, errorsModule, errorsModelsModule))

	for _, version := range specification.Versions {
		versionModule := rootModule.Submodule(version.Name.FlatCase())
		modelsModule := versionModule.Submodule(types.VersionModelsPackage)
		routingModule := versionModule.Submodule("routing")

		sources.AddGeneratedAll(generateRoutings(&version, versionModule, routingModule, contentTypeModule, errorsModule, errorsModelsModule, modelsModule, paramsParserModule, respondModule, modelsGenerator))
		sources.AddGeneratedAll(generateServiceInterfaces(&version, versionModule, modelsModule, errorsModelsModule, emptyModule))
		sources.AddGenerated(modelsGenerator.GenerateVersionModels(version.ResolvedModels, modelsModule, enumsModule))
	}

	if swaggerPath != "" {
		sources.AddGenerated(openapi.GenerateOpenapi(specification, swaggerPath))
	}

	if servicesPath != "" {
		rootServicesModule := module.New(moduleName, servicesPath)
		for _, version := range specification.Versions {
			versionServicesModule := rootServicesModule.Submodule(version.Name.FlatCase())
			versionModule := rootModule.Submodule(version.Name.FlatCase())
			modelsModule := versionModule.Submodule(types.VersionModelsPackage)
			sources.AddScaffoldedAll(generateServiceImplementations(&version, versionModule, modelsModule, versionServicesModule))
		}
	}

	return sources
}
