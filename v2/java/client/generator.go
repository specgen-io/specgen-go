package client

import (
	"github.com/specgen-io/specgen-go/v2/java/models"
	"github.com/specgen-io/specgen-go/v2/java/types"
)

type Generator struct {
	Jsonlib string
	Types   *types.Types
	Models  models.Generator
}

func NewGenerator(jsonlib string) *Generator {
	return &Generator{
		jsonlib,
		models.NewTypes(jsonlib),
		models.NewGenerator(jsonlib),
	}
}
