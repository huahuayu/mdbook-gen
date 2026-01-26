package cmd

import (
	"fmt"
	"mdbook-gen/internal/core"
	"os"
)

func Build(outputDir string) {
	cwd, _ := os.Getwd()
	fmt.Println("Building book in:", cwd)
	if err := core.RenderBook(cwd, outputDir); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}
