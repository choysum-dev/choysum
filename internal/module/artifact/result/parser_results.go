package result

import (
	"runtime"
	"sync"

	"github.com/choysum-dev/choysum/internal/parser"
)

var (
	parserResultsByBuildResult sync.Map
	buildResultFinalizers      sync.Map
)

func WithParserResults(result *BuildResult, parserResults []*parser.ParserResult) *BuildResult {
	if result == nil {
		return nil
	}

	if parserResults == nil {
		parserResultsByBuildResult.Delete(result)
	} else {
		parserResultsByBuildResult.Store(result, parserResults)
	}

	if _, loaded := buildResultFinalizers.LoadOrStore(result, struct{}{}); !loaded {
		runtime.SetFinalizer(result, func(v *BuildResult) {
			parserResultsByBuildResult.Delete(v)
			buildResultFinalizers.Delete(v)
		})
	}

	return result
}

func SetParserResults(result *BuildResult, parserResults []*parser.ParserResult) {
	_ = WithParserResults(result, parserResults)
}

func ParserResults(result *BuildResult) []*parser.ParserResult {
	if result == nil {
		return nil
	}
	value, ok := parserResultsByBuildResult.Load(result)
	if !ok {
		return nil
	}
	stored, ok := value.([]*parser.ParserResult)
	if !ok {
		return nil
	}
	return stored
}

func AppendParserResult(result *BuildResult, parserResult *parser.ParserResult) {
	if result == nil || parserResult == nil {
		return
	}
	current := ParserResults(result)
	current = append(current, parserResult)
	SetParserResults(result, current)
}
