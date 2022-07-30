package scala

import (
	"github.com/specgen-io/specgen-go/v2/generator"
)

var ScalaConfig = generator.Config{"  ", 2, nil}

func NewScalaWriter() *generator.Writer {
	return generator.NewWriter(ScalaConfig)
}
