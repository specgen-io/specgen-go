package empty

import (
	"github.com/specgen-io/specgen-golang/v2/goven/generator"
	"github.com/specgen-io/specgen-golang/v2/module"
	"github.com/specgen-io/specgen-golang/v2/writer"
)

func GenerateEmpty(emptyModule module.Module) *generator.CodeFile {
	w := writer.New(emptyModule, `empty.go`)
	w.Lines(`
type Type struct{}

var Value = Type{}
`)
	return w.ToCodeFile()
}
