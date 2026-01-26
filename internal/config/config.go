package config

type Config struct {
	Title      string         `yaml:"title"`
	Author     string         `yaml:"author"`
	Copyright  string         `yaml:"copyright"`
	OutputDir  string         `yaml:"output_dir"`
	Categories map[int]string `yaml:"categories"`
}
