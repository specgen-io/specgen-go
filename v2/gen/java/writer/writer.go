package writer

import (
	"github.com/specgen-io/specgen-go/v2/generator"
)

var JavaConfig = generator.Config{"\t", 2, nil}

func NewJavaWriter() *generator.Writer {
	return generator.NewWriter(JavaConfig)
}
