package main

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
)

// NoOSExitAnalyzer — это анализатор, который проверяет прямые вызовы os.Exit в пакете main.
var NoOSExitAnalyzer = &analysis.Analyzer{
	Name:     "noosexit",
	Doc:      "проверяет прямые вызовы os.Exit в пакете main",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	// Нас интересует только пакет main.
	if pass.Pkg.Name() != "main" {
		return nil, nil
	}

	for _, file := range pass.Files {
		var mainFunc *ast.FuncDecl
		ast.Inspect(file, func(node ast.Node) bool {
			if fn, ok := node.(*ast.FuncDecl); ok && fn.Name.Name == "main" {
				mainFunc = fn
				return false // Нашли main, дальше не ищем
			}
			return true
		})

		if mainFunc != nil {
			ast.Inspect(mainFunc.Body, func(node ast.Node) bool {
				if call, ok := node.(*ast.CallExpr); ok {
					if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
						if ident, ok := sel.X.(*ast.Ident); ok && ident.Name == "os" && sel.Sel.Name == "Exit" {
							pass.Reportf(call.Pos(), "прямой вызов os.Exit в функции main запрещен")
						}
					}
				}
				return true
			})
		}
	}
	return nil, nil
}
