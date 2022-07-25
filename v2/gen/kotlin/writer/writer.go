package writer

import (
	"github.com/specgen-io/specgen-go/v2/generator"
)

var KotlinConfig = generator.Config{"\t", 2, nil}

func NewKotlinWriter() *generator.Writer {
	return generator.NewWriter(KotlinConfig)
}
