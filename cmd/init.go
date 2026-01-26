package cmd

import (
"fmt"
"os"
"path/filepath"
"mdbook-gen/templates"
)

func Init(projectName string) {
	if _, err := os.Stat(projectName); !os.IsNotExist(err) {
		fmt.Printf("❌ 目录 %s 已存在\n", projectName)
		return
	}

	fmt.Printf("Creating new book project: %s\n", projectName)
	os.MkdirAll(projectName, 0755)
	os.MkdirAll(filepath.Join(projectName, "book"), 0755)
	os.MkdirAll(filepath.Join(projectName, "assets", "css"), 0755)

	// Write book.yaml
	yamlData, _ := templates.Assets.ReadFile("book.yaml")
	os.WriteFile(filepath.Join(projectName, "book.yaml"), yamlData, 0644)

	// Write sample chapters
	// Note: embed.FS uses forward slashes
	entries, _ := templates.Assets.ReadDir("sample")
	for _, entry := range entries {
		data, _ := templates.Assets.ReadFile("sample/" + entry.Name())
		os.WriteFile(filepath.Join(projectName, "book", entry.Name()), data, 0644)
	}

	// Write Default CSS (optional, user might want to customize it immediately)
	cssData, _ := templates.Assets.ReadFile("main.css")
	os.WriteFile(filepath.Join(projectName, "assets", "css", "main.css"), cssData, 0644)

	fmt.Println("✅ Initialize success!")
	fmt.Printf("Run: cd %s && go run ../main.go build\n", projectName)
}
