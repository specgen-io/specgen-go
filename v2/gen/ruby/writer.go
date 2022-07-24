package ruby

import (
	"github.com/specgen-io/specgen/v2/generator"
)

var RubyConfig = generator.Config{"  ", 2, nil}

func NewRubyWriter() *generator.Writer {
	return generator.NewWriter(RubyConfig)
}