// Package plugin registers bigo as a golangci-lint module plugin.
package plugin

import (
	"github.com/RomanAgaltsev/bigo/analyzer"
	"github.com/golangci/plugin-module-register/register"
	"golang.org/x/tools/go/analysis"
)

func init() {
	register.Plugin("bigo", New)
}

// New constructs the bigo plugin. bigo has no settings.
func New(any) (register.LinterPlugin, error) {
	return &plugin{}, nil
}

type plugin struct{}

func (*plugin) BuildAnalyzers() ([]*analysis.Analyzer, error) {
	return []*analysis.Analyzer{analyzer.Analyzer}, nil
}

func (*plugin) GetLoadMode() string {
	return register.LoadModeTypesInfo
}
