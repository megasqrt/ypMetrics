package main

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestNoOSExit(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, NoOSExitAnalyzer, "directexit/a","directexit/b")
}
