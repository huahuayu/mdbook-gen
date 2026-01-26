# mdbook-gen

A simple, opinionated Markdown-to-HTML eBook generator implemented in Go.

## Features

- **Standard Structure**: Organized by chapters (`00.00-frontmatter.md`, `01-introduction.md`, etc.).
- **Embedded Assets**: Zero-dependency binary (default CSS is embedded).
- **Automated TOC**: Automatically generates Table of Contents.
- **Syntax Highlighting**: Built-in support for code blocks.
- **Mermaid Support**: Built-in support for mermaid.js diagrams.

## Design

The CSS styling is inspired by [Let's Go](https://lets-go.alexedwards.net/) by Alex Edwards, a beautifully designed technical book. The clean, readable layout makes it perfect for technical documentation and programming books.

## Installation

```bash
git clone https://github.com/yourusername/mdbook-gen.git
cd mdbook-gen
go install
```

## Usage

### 1. Initialize a new book

Create a new directory with a sample structure:

```bash
mdbook-gen init my-new-book
```

This will create:
- `book.yaml` (Configuration)
- `book/` (Markdown source files)
- `assets/` (Custom static assets)

### 2. Build the book

Go into your book directory and run build:

```bash
cd my-new-book
mdbook-gen build
```

This generates the static site in `output.html/` (or whatever `output_dir` is set to in `book.yaml`).

## Directory Structure

Files should follow this naming convention:

- `book/00.00-frontmatter.md` -> Becomes `index.html` (or front matter)
- `book/00.01-contents.md` -> Becomes TOC page (optional)
- `book/01-title.md` -> Becomes `01.00-title.html`
- `book/01.01-subtitle.md` -> Becomes `01.01.subtitle.html`

## Configuration (book.yaml)

```yaml
title: "My Book Title"
author: "Author Name"
copyright: "Copyright 2024"
output_dir: "dist"
categories:
  1: "Part I: Basics"
  2: "Part II: Advanced"
```

## License

MIT
