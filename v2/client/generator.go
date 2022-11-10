package client

import (
	"github.com/specgen-io/specgen-golang/v2/goven/generator"
	"github.com/specgen-io/specgen-golang/v2/goven/spec"
	"github.com/specgen-io/specgen-golang/v2/models"
	"github.com/specgen-io/specgen-golang/v2/module"
	"github.com/specgen-io/specgen-golang/v2/types"
)

type ClientGenerator interface {
	GenerateClientsImplementations(version *spec.Version, versionModule, convertModule, emptyModule, errorsModule, modelsModule, respondModule module.Module) []generator.CodeFile
}

type Generator struct {
	Types  *types.Types
	Models models.Generator
	Client ClientGenerator
}

func NewGenerator(modules *models.Modules) *Generator {
	types := types.NewTypes()
	return &Generator{
		types,
		models.NewGenerator(modules),
		NewNetHttpGenerator(types),
	}
}
