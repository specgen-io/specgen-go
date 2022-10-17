package service

import (
	"github.com/specgen-io/specgen-golang/v2/goven/generator"
	"github.com/specgen-io/specgen-golang/v2/goven/spec"
	"github.com/specgen-io/specgen-golang/v2/models"
	"github.com/specgen-io/specgen-golang/v2/module"
	"github.com/specgen-io/specgen-golang/v2/types"
)

type ServiceGenerator interface {
	GenerateSpecRouting(specification *spec.Spec, rootModule module.Module) *generator.CodeFile
	HttpErrors(converterModule, errorsModelsModule, paramsParserModule, respondModule module.Module, responses *spec.Responses) []generator.CodeFile
	CheckContentType(contentTypeModule, errorsModule, errorsModelsModule module.Module) *generator.CodeFile
	GenerateRoutings(version *spec.Version, versionModule, routingModule, contentTypeModule, errorsModule, errorsModelsModule, modelsModule, paramsParserModule, respondModule module.Module) []generator.CodeFile
}

type Generator struct {
	Types   *types.Types
	Models  models.Generator
	Service ServiceGenerator
}

func NewGenerator(modules *models.Modules) *Generator {
	modelsGenerator := models.NewGenerator(modules)
	types := types.NewTypes()
	return &Generator{
		types,
		modelsGenerator,
		NewVestigoGenerator(types, modelsGenerator),
	}
}
