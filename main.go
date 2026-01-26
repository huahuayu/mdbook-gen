package main

import (
	"fmt"
	"mdbook-gen/cmd"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		help()
		return
	}

	switch os.Args[1] {
	case "init":
		if len(os.Args) < 3 {
			fmt.Println("Usage: mdbook-gen init <project-name>")
			return
		}
		cmd.Init(os.Args[2])
	case "build":
		outputDir := ""
		// Parse --output flag if provided
		for i := 2; i < len(os.Args); i++ {
			if os.Args[i] == "--output" && i+1 < len(os.Args) {
				outputDir = os.Args[i+1]
				break
			}
		}
		cmd.Build(outputDir)
	default:
		help()
	}
}

func help() {
	fmt.Println("mdbook-gen v0.1")
	fmt.Println("Usage:")
	fmt.Println("  init <name>        Initialize a new book project")
	fmt.Println("  build [--output DIR]  Build the book in current directory")
}
