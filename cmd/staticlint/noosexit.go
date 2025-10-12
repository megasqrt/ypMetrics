package main

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
)

// NoOSExitAnalyzer — это анализатор, который проверяет прямые вызовы os.Exit в функции main пакета main.
var NoOSExitAnalyzer = &analysis.Analyzer{
	Name: "noosexit",
	Doc:  "проверяет прямые вызовы os.Exit в main",
	Run:  run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	for _, file := range pass.Files {
		// Нас интересует только пакет main.
		if pass.Pkg.Name() != "main" {
			continue
		}

		ast.Inspect(file, func(node ast.Node) bool {
			// Ищем объявление функции.
			if fn, ok := node.(*ast.FuncDecl); ok && fn.Name.Name == "main" {
				// Мы находимся в функции main. Теперь мы проверяем ее тело на наличие вызовов os.Exit.
				ast.Inspect(fn.Body, func(n ast.Node) bool {
					// Ищем выражение вызова.
					if call, ok := n.(*ast.CallExpr); ok {
						// Проверяем, является ли вызов функции выражением-селектором (например, os.Exit).
						if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
							// Проверяем, является ли объект селектора идентификатором с именем "os".
							if ident, ok := sel.X.(*ast.Ident); ok && ident.Name == "os" && sel.Sel.Name == "Exit" {
								pass.Reportf(call.Pos(), "прямой вызов os.Exit в функции main запрещен")
							}
						}
					}
					return true
				})
			}
			return true
		})
	}
	return nil, nil
}
