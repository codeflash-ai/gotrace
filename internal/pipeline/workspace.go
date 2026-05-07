package pipeline

import (
	"fmt"
	"go/printer"
	"go/token"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/codeflash-ai/gotrace/internal/rewriter"
)

type Workspace struct {
	dir    string
	srcDir string
}

func NewWorkspace(srcDir string) (*Workspace, error) {
	dir, err := os.MkdirTemp("", "gotrace-")
	if err != nil {
		return nil, err
	}

	ws := &Workspace{dir: dir, srcDir: srcDir}
	if err := ws.copySource(); err != nil {
		os.RemoveAll(dir)
		return nil, err
	}

	return ws, nil
}

func (w *Workspace) Dir() string     { return w.dir }
func (w *Workspace) Cleanup()        { os.RemoveAll(w.dir) }

func (w *Workspace) copySource() error {
	return filepath.WalkDir(w.srcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(w.srcDir, path)
		if err != nil {
			return err
		}

		if shouldSkipPath(rel, d) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		dst := filepath.Join(w.dir, rel)

		if d.IsDir() {
			return os.MkdirAll(dst, 0755)
		}

		return copyFile(path, dst)
	})
}

func (w *Workspace) WriteRewrittenFiles(result *rewriter.Result) error {
	for _, rf := range result.Files {
		rel, err := filepath.Rel(w.srcDir, rf.Original)
		if err != nil {
			return fmt.Errorf("rel path for %s: %w", rf.Original, err)
		}

		dst := filepath.Join(w.dir, rel)
		if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
			return err
		}

		f, err := os.Create(dst)
		if err != nil {
			return fmt.Errorf("create %s: %w", dst, err)
		}

		cfg := &printer.Config{Mode: printer.TabIndent | printer.UseSpaces, Tabwidth: 8}
		err = cfg.Fprint(f, rf.Fset, rf.File)
		f.Close()
		if err != nil {
			return fmt.Errorf("print %s: %w", dst, err)
		}
	}
	return nil
}

func (w *Workspace) InjectTracerPackage(tracerSrcDir string) error {
	dstDir := filepath.Join(w.dir, "gotrace_tracer_runtime")
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return err
	}

	entries, err := os.ReadDir(tracerSrcDir)
	if err != nil {
		return fmt.Errorf("read tracer source: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		src := filepath.Join(tracerSrcDir, entry.Name())
		dst := filepath.Join(dstDir, entry.Name())
		if err := copyFile(src, dst); err != nil {
			return err
		}
	}

	goMod := "module gotrace_tracer_runtime\n\ngo 1.26\n"
	return os.WriteFile(filepath.Join(dstDir, "go.mod"), []byte(goMod), 0644)
}

func (w *Workspace) UpdateGoMod() error {
	modPath := filepath.Join(w.dir, "go.mod")
	data, err := os.ReadFile(modPath)
	if err != nil {
		return err
	}

	content := string(data)
	if !strings.Contains(content, "gotrace_tracer_runtime") {
		content += "\nrequire gotrace_tracer_runtime v0.0.0\n"
		content += "replace gotrace_tracer_runtime => ./gotrace_tracer_runtime\n"
	}

	return os.WriteFile(modPath, []byte(content), 0644)
}

func (w *Workspace) WriteGeneratedInit(fset *token.FileSet) error {
	return nil
}

func shouldSkipPath(rel string, d fs.DirEntry) bool {
	parts := strings.Split(rel, string(filepath.Separator))
	for _, p := range parts {
		if p == ".git" || p == ".hg" || p == ".svn" || p == "node_modules" {
			return true
		}
	}
	return false
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
