// Package main реализует мультичекер для статического анализа исходного кода Go.
//
// # Использование
//
// Для запуска мультичекера выполните следующую команду из корня вашего проекта:
//
//	go run cmd/staticlint/main.go ./...
//
// # Включенные анализаторы
//
// Этот мультичекер включает в себя различные анализаторы для обеспечения качества, корректности и стиля кода.
//
// ## Стандартные анализаторы Go (golang.org/x/tools/go/analysis/passes)
//
// Включены все стандартные статические анализаторы от команды Go. Эти проверки охватывают широкий спектр потенциальных проблем, включая:
// - `printf`: Проверяет согласованность строк формата Printf и аргументов.
// - `shadow`: Проверяет наличие затененных переменных.
// - `structtag`: Проверяет корректность тегов структур.
// - `unreachable`: Проверяет наличие недостижимого кода.
// - ... и многие другие.
//
// ## Staticcheck (staticcheck.io)
//
// Включен большой набор анализаторов из пакета staticcheck.
//
// ### Класс SA
//
// Включены все анализаторы из класса SA (Static Analysis). Это важные проверки на корректность кода.
// Полный список см. по адресу: https://staticcheck.io/docs/checks/#SA
//
// ### Другие анализаторы Staticcheck
//
// - `ST1000`: Проверяет некорректные комментарии к пакету.
//
// ## Публичные анализаторы
//
// - `errcheck`: Проверяет наличие необработанных ошибок в коде Go. (https://github.com/kisielk/errcheck)
// - `structcheck`: Находит неиспользуемые поля структур. (из golang.org/x/tools/go/analysis/passes)
//
// ## Пользовательские анализаторы
//
//   - `noosexit`: Этот анализатор запрещает прямые вызовы `os.Exit` в пакете `main`.
//     Это поощряет правильную обработку ошибок и кодов завершения, возвращая их из функции `main`.
package main

import (
	"github.com/kisielk/errcheck/errcheck"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
	"golang.org/x/tools/go/analysis/passes/printf"
	"golang.org/x/tools/go/analysis/passes/shadow"
	"golang.org/x/tools/go/analysis/passes/structtag"
	"golang.org/x/tools/go/analysis/passes/unreachable"
	"honnef.co/go/tools/staticcheck"
	"honnef.co/go/tools/stylecheck"
)

func main() {
	var analyzers []*analysis.Analyzer

	// Добавляем стандартные анализаторы
	analyzers = append(analyzers, printf.Analyzer)
	analyzers = append(analyzers, shadow.Analyzer)
	analyzers = append(analyzers, structtag.Analyzer)
	analyzers = append(analyzers, unreachable.Analyzer)

	// Добавляем анализаторы класса SA из staticcheck
	for _, v := range staticcheck.Analyzers {
		if len(v.Analyzer.Name) > 2 && v.Analyzer.Name[:2] == "SA" {
			analyzers = append(analyzers, v.Analyzer)
		}
	}

	// Добавляем один анализатор из другого класса staticcheck (ST1000)
	for _, v := range stylecheck.Analyzers {
		if v.Analyzer.Name == "ST1000" {
			analyzers = append(analyzers, v.Analyzer)
		}
	}

	// Добавляем публичные анализаторы
	analyzers = append(analyzers, errcheck.Analyzer)

	// Добавляем свой анализатор
	analyzers = append(analyzers, NoOSExitAnalyzer)

	multichecker.Main(analyzers...)
}
