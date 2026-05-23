// Command gendocs emits shell completions and a man page for dots into
// assets/completions and assets/man. Invoked from .goreleaser.yaml's
// before.hooks so the resulting files can be packaged into the .deb
// produced by nfpm. Output lives outside dist/ because goreleaser's
// "ensure distribution directory" check runs after before.hooks and
// refuses a non-empty dist/.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra/doc"

	"github.com/jlrickert/dots/pkg/cli"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "gendocs: %s\n", err)
		os.Exit(1)
	}
}

func run() error {
	root := cli.NewRootCmd(nil)
	root.DisableAutoGenTag = true

	manDir := filepath.Join("assets", "man")
	compDir := filepath.Join("assets", "completions")
	if err := os.MkdirAll(manDir, 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(compDir, 0o755); err != nil {
		return err
	}

	header := &doc.GenManHeader{Title: "DOTS", Section: "1"}
	if err := doc.GenManTree(root, header, manDir); err != nil {
		return fmt.Errorf("man: %w", err)
	}

	bashPath := filepath.Join(compDir, "dots.bash")
	if err := writeFile(bashPath, func(f *os.File) error {
		return root.GenBashCompletionV2(f, true)
	}); err != nil {
		return fmt.Errorf("bash completion: %w", err)
	}

	zshPath := filepath.Join(compDir, "dots.zsh")
	if err := writeFile(zshPath, func(f *os.File) error {
		return root.GenZshCompletion(f)
	}); err != nil {
		return fmt.Errorf("zsh completion: %w", err)
	}

	fishPath := filepath.Join(compDir, "dots.fish")
	if err := writeFile(fishPath, func(f *os.File) error {
		return root.GenFishCompletion(f, true)
	}); err != nil {
		return fmt.Errorf("fish completion: %w", err)
	}

	return nil
}

func writeFile(path string, gen func(*os.File) error) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return gen(f)
}
