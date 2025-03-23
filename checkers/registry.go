// AUTOMATICALLY GENERATED: DO NOT EDIT

package checkers

import (
	"globstar.dev/checkers/javascript"
	"globstar.dev/checkers/python"
	goAnalysis "globstar.dev/analysis"
)

type Analyzer struct {
	TestDir   string
	Analyzers []*goAnalysis.Analyzer
}

var AnalyzerRegistry = []Analyzer{
	{
		TestDir:   "checkers/javascript/testdata", // relative to the repository root
		Analyzers: []*goAnalysis.Analyzer{
			javascript.SQLInjection,
			javascript.NoDoubleEq,

		},
	},
	{
		TestDir: "checkers/python/testdata",
		Analyzers: []*goAnalysis.Analyzer{
			python.DjangoSQLInjection,
			python.DjangoSSRFInjection,
			python.DjangoCsvWriterInjection,
			python.DjangoInsecurePickleDeserialize,
			python.DjangoMissingThrottleConfig,
			python.DjangoPasswordEmptyString,
			python.DjangoRequestDataWrite,
			python.DjangoRequestHttpResponse,
			python.InsecureUrllibFtp,
			python.DjangoNanInjection,

		},
	},
}
