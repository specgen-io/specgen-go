package gen

import (
	"github.com/specgen-io/specgen-go/v2/generator"
	"github.com/specgen-io/specgen-go/v2/golang"
	"github.com/specgen-io/specgen-go/v2/openapi"
	"github.com/specgen-io/specgen-go/v2/ruby"
	"github.com/specgen-io/specgen-go/v2/scala"
	"github.com/specgen-io/specgen-go/v2/typescript"
	"github.com/specgen-io/specgen-go/v2/gen/java"
	"github.com/specgen-io/specgen-go/v2/gen/kotlin"
)

var Generators = []generator.Generator{
	golang.Models,
	golang.Client,
	golang.Service,
	java.Models,
	java.Client,
	java.Service,
	kotlin.Models,
	kotlin.Client,
	kotlin.Service,
	ruby.Models,
	ruby.Client,
	scala.Models,
	scala.Client,
	scala.Service,
	typescript.Models,
	typescript.Client,
	typescript.Service,
	openapi.Openapi,
}
