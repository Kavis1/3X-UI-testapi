package main

import (
	"embed"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

//go:embed payload/**
var payload embed.FS

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	target := flag.String("target", ".", "path to x-ui project root to patch")
	withCLI := flag.Bool("with-cli", true, "copy cmd/api-guard into target")
	flag.Parse()

	if *target == "" {
		return errors.New("target path is required")
	}

	absTarget, err := filepath.Abs(*target)
	if err != nil {
		return err
	}

	fmt.Printf("Applying hardened API files to %s\n", absTarget)

	if err := copyPayload(absTarget, *withCLI); err != nil {
		return err
	}
	if err := ensureGoMod(absTarget); err != nil {
		return err
	}

	fmt.Println("Done.")
	fmt.Println("Next steps:")
	fmt.Println("  1) В таргет-проекте выполните: go mod tidy")
	fmt.Println("  2) Соберите CLI управления:   go build -o api-guard ./cmd/api-guard")
	fmt.Println("  3) Перезапустите панель с обновленным API")
	return nil
}

func copyPayload(target string, withCLI bool) error {
	return fs.WalkDir(payload, "payload", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel := strings.TrimPrefix(path, "payload/")
		if rel == "" {
			return nil
		}

		if !withCLI && strings.HasPrefix(rel, "cmd/api-guard/") {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}

		dest := filepath.Join(target, filepath.FromSlash(rel))

		if d.IsDir() {
			return os.MkdirAll(dest, 0o755)
		}

		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return err
		}

		srcFile, err := payload.Open(path)
		if err != nil {
			return err
		}
		defer srcFile.Close()

		dstFile, err := os.Create(dest)
		if err != nil {
			return err
		}
		defer dstFile.Close()

		if _, err := io.Copy(dstFile, srcFile); err != nil {
			return err
		}

		return nil
	})
}

func ensureGoMod(root string) error {
	modPath := filepath.Join(root, "go.mod")
	data, err := os.ReadFile(modPath)
	if err != nil {
		return err
	}

	const dep = "\tgolang.org/x/time v0.14.0\n"
	if strings.Contains(string(data), "golang.org/x/time") {
		return nil
	}

	requireIdx := strings.Index(string(data), "require (")
	if requireIdx == -1 {
		return errors.New("go.mod: require block not found; add golang.org/x/time manually")
	}

	closingIdx := strings.Index(string(data)[requireIdx:], ")")
	if closingIdx == -1 {
		return errors.New("go.mod: malformed require block; add golang.org/x/time manually")
	}
	closingIdx += requireIdx

	var b strings.Builder
	b.WriteString(string(data[:closingIdx]))
	if !strings.HasSuffix(b.String(), "\n") {
		b.WriteString("\n")
	}
	b.WriteString(dep)
	b.WriteString(string(data[closingIdx:]))

	return os.WriteFile(modPath, []byte(b.String()), 0o644)
}
