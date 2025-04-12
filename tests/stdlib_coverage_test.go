package tests

import (
	"cmp"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/DDP-Projekt/Kompilierer/src/ast"
	"github.com/DDP-Projekt/Kompilierer/src/ddperror"
	"github.com/DDP-Projekt/Kompilierer/src/ddppath"
	"github.com/DDP-Projekt/Kompilierer/src/parser"
	"golang.org/x/exp/maps"
)

type moduleVisitor func(*ast.Module)

var (
	_ ast.Visitor      = moduleVisitor(nil)
	_ ast.ModuleSetter = moduleVisitor(nil)
)

func (moduleVisitor) Visitor() {}
func (v moduleVisitor) SetModule(m *ast.Module) {
	v(m)
}

type funcDeclVisitor func(*ast.FuncDecl)

var (
	_ ast.Visitor         = funcDeclVisitor(nil)
	_ ast.FuncDeclVisitor = funcDeclVisitor(nil)
)

func (funcDeclVisitor) Visitor() {}
func (v funcDeclVisitor) VisitFuncDecl(f *ast.FuncDecl) ast.VisitResult {
	v(f)
	return ast.VisitSkipChildren
}

type funcCallVisitor func(*ast.FuncCall)

var (
	_ ast.Visitor         = funcCallVisitor(nil)
	_ ast.FuncCallVisitor = funcCallVisitor(nil)
)

func (funcCallVisitor) Visitor() {}
func (v funcCallVisitor) VisitFuncCall(f *ast.FuncCall) ast.VisitResult {
	v(f)
	return ast.VisitRecurse
}

type unaryExprVisitor func(*ast.UnaryExpr)

func (unaryExprVisitor) Visitor() {}
func (v unaryExprVisitor) VisitUnaryExpr(c *ast.UnaryExpr) ast.VisitResult {
	v(c)
	return ast.VisitRecurse
}

type binaryExprVisitor func(*ast.BinaryExpr)

func (binaryExprVisitor) Visitor() {}
func (v binaryExprVisitor) VisitBinaryExpr(c *ast.BinaryExpr) ast.VisitResult {
	v(c)
	return ast.VisitRecurse
}

type ternaryExprVisitor func(*ast.TernaryExpr)

func (ternaryExprVisitor) Visitor() {}
func (v ternaryExprVisitor) VisitTernaryExpr(c *ast.TernaryExpr) ast.VisitResult {
	v(c)
	return ast.VisitRecurse
}

type castExprVisitor func(*ast.CastExpr)

func (castExprVisitor) Visitor() {}
func (v castExprVisitor) VisitCastExpr(c *ast.CastExpr) ast.VisitResult {
	v(c)
	return ast.VisitRecurse
}

var (
	duden_modules = make(map[string]*ast.Module, 30)
	duden_funcs   = make(map[*ast.FuncDecl]struct{}, 100)
	wd, _         = os.Getwd()
)

func init_duden() {
	err := filepath.WalkDir(ddppath.Duden, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || filepath.Ext(path) != ".ddp" {
			return nil
		}

		if path, err = filepath.Abs(path); err != nil {
			return err
		}

		module, err := parser.Parse(parser.Options{
			FileName:     path,
			Modules:      duden_modules,
			ErrorHandler: ddperror.MakePanicHandler(),
		})
		if err != nil {
			return err
		}

		ast.VisitModuleRec(module, moduleVisitor(func(m *ast.Module) {
			duden_modules[m.FileName] = m
		}))

		return nil
	})
	if err != nil {
		panic(err)
	}

	for _, module := range duden_modules {
		ast.VisitModule(module, funcDeclVisitor(func(f *ast.FuncDecl) {
			if f.IsPublic {
				duden_funcs[f] = struct{}{}
			}
		}))
	}
}

type moduleFuncInfo struct {
	called int
	total  int
}

func TestStdlibCoverage(t *testing.T) {
	init_duden()
	file, err := os.OpenFile("coverage.md", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		t.Fatalf("Error opening coverage file: %s", err)
	}
	defer file.Close()

	fmt.Fprint(file, "# Duden Coverage\n\n")
	fmt.Fprintf(file, "Duden Module: %d<br>\n", len(duden_modules))
	fmt.Fprintf(file, "Duden Funktionen: %d<br><br>\n\n", len(duden_funcs))

	const stdlib_testdata = "testdata/stdlib"
	called_functions := make(map[*ast.FuncDecl]int, len(duden_funcs))
	func_called := func(f *ast.FuncDecl) {
		if _, ok := duden_funcs[f]; !ok {
			return
		}
		if n, ok := called_functions[f]; ok {
			called_functions[f] = n + 1
		} else {
			called_functions[f] = 1
		}
	}

	err = filepath.WalkDir(stdlib_testdata, func(path string, d fs.DirEntry, err error) error {
		if err != nil || !d.IsDir() || path == stdlib_testdata {
			return nil
		}

		ddp_path := filepath.Join(path, filepath.Base(path)+".ddp")
		module, err := parser.Parse(parser.Options{
			FileName: ddp_path,
			Modules:  duden_modules,
			ErrorHandler: func(e ddperror.Error) {
				t.Fatalf("Error parsing %s: %s", ddp_path, e)
			},
		})
		if err != nil {
			return err
		}

		ast.VisitModule(module, funcCallVisitor(func(f *ast.FuncCall) {
			func_called(f.Func)
		}))

		ast.VisitModule(module, unaryExprVisitor(func(c *ast.UnaryExpr) {
			if c.OverloadedBy != nil {
				func_called(c.OverloadedBy.Decl)
			}
		}))

		ast.VisitModule(module, binaryExprVisitor(func(c *ast.BinaryExpr) {
			if c.OverloadedBy != nil {
				func_called(c.OverloadedBy.Decl)
			}
		}))

		ast.VisitModule(module, ternaryExprVisitor(func(c *ast.TernaryExpr) {
			if c.OverloadedBy != nil {
				func_called(c.OverloadedBy.Decl)
			}
		}))

		ast.VisitModule(module, castExprVisitor(func(c *ast.CastExpr) {
			if c.OverloadedBy != nil {
				func_called(c.OverloadedBy.Decl)
			}
		}))

		return nil
	})
	if err != nil {
		t.Fatalf("Error walking the test directory: %s", err)
	}

	functions_per_module := make(map[*ast.Module]moduleFuncInfo, len(duden_modules))
	for fun := range duden_funcs {
		info := functions_per_module[fun.Module()]
		info.total++
		if _, ok := called_functions[fun]; ok {
			info.called++
		}
		functions_per_module[fun.Module()] = info
	}

	sorted_modules := maps.Values(duden_modules)
	slices.SortStableFunc(sorted_modules, func(a, b *ast.Module) int {
		return cmp.Compare(float32(functions_per_module[a].called)/float32(functions_per_module[a].total), float32(functions_per_module[b].called)/float32(functions_per_module[b].total))
	})

	fmt.Fprintf(file, "Aufgerufene Funktionen: %d<br>\n", len(called_functions))
	fmt.Fprintf(file, "Nicht aufgerufene Funktionen: %d<br>\n", len(duden_funcs)-len(called_functions))
	fmt.Fprintf(file, "Coverage: %.2f%%\n\n", float64(len(called_functions))/float64(len(duden_funcs))*100)

	fmt.Fprintf(file, "### Index\n\n")
	fmt.Fprintf(file, "| Module | Funktionen | Aufgerufene Funktionen | Nicht Aufgerufene Funktionen | %% Aufgerufen |\n")
	fmt.Fprintf(file, "|--------|------------| ---------------------- | ---------------------------- | -- |\n")

	for _, mod := range sorted_modules {
		modName, err := filepath.Rel(ddppath.Duden, mod.FileName)
		if err != nil {
			modName = mod.FileName
			t.Logf("Error getting relative path for %s: %s", modName, err)
		}
		info := functions_per_module[mod]
		fmt.Fprintf(file, "| [%s](#%s) | %d | %d | %d | %.2f%% |\n",
			modName,
			strings.ToLower(strings.ReplaceAll(filepath.Base(modName), ".", "")),
			info.total,
			info.called,
			info.total-info.called,
			float32(info.called)/float32(info.total)*100,
		)
	}
	fmt.Fprintln(file)

	max_called_len := 0
	for f := range called_functions {
		if len(f.Name()) > max_called_len {
			max_called_len = len(f.Name())
		}
	}

	for _, mod := range sorted_modules {
		modName, err := filepath.Rel(ddppath.Duden, mod.FileName)
		if err != nil {
			modName = mod.FileName
			t.Logf("Error getting relative path for %s: %s", modName, err)
		}

		info := functions_per_module[mod]
		fmt.Fprintf(file, "### %s\n", getFileLink(t, modName, mod.FileName, -1))
		fmt.Fprintf(file, "<sup>[« Zurück](#index)</sup>\n")
		fmt.Fprintf(file, "#### Aufgerufene Funktionen:\n\n")

		if info.called > 0 {
			fmt.Fprintf(file, "| Funktion | Aufrufe |\n")
			fmt.Fprintf(file, "|----------|-------|\n")
			for f, n := range called_functions {
				if f.Module() == mod {
					fmt.Fprintf(file, "| %s | %d |\n", getFileLink(t, f.Name(), f.Mod.FileName, int(f.NameTok.Line())), n)
					delete(duden_funcs, f)
				}
			}
		} else {
			fmt.Fprintf(file, "Keine\n")
		}

		fmt.Fprintf(file, "\n#### Nicht Aufgerufene Funktionen:\n\n")
		if info.total-info.called > 0 {
			fmt.Fprintf(file, "| Funktion |\n")
			fmt.Fprintf(file, "|----------|\n")
			for f := range duden_funcs {
				if f.Module() == mod {
					fmt.Fprintf(file, "| %s |\n", getFileLink(t, f.Name(), f.Mod.FileName, int(f.NameTok.Line())))
				}
			}
			fmt.Fprintln(file)
		} else {
			fmt.Fprintf(file, "Keine\n")
		}
	}
}

func getFileLink(t *testing.T, display, path string, line int) string {
	linkPath, err := filepath.Rel(wd, path)
	if err != nil {
		linkPath = path
		t.Logf("Error getting relative path for %s: %s", linkPath, err)
	}
	if line == -1 {
		return fmt.Sprintf("[%s](%s)", display, filepath.ToSlash(linkPath))
	} else {
		return fmt.Sprintf("[%s](%s#L%d)", display, filepath.ToSlash(linkPath), line)
	}
}
