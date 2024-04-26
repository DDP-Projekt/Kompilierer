package tests

import (
	"io/fs"
	"path/filepath"
	"testing"

	"github.com/DDP-Projekt/Kompilierer/src/ast"
	"github.com/DDP-Projekt/Kompilierer/src/ddperror"
	"github.com/DDP-Projekt/Kompilierer/src/ddppath"
	"github.com/DDP-Projekt/Kompilierer/src/parser"
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
	return ast.VisitSkipChildren
}

var (
	duden_modules = make(map[string]*ast.Module, 30)
	duden_funcs   = make(map[*ast.FuncDecl]struct{}, 100)
)

func init() {
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

func TestStdlibCoverage(t *testing.T) {
	t.Logf("Duden modules: %d", len(duden_modules))
	t.Logf("Duden functions: %d", len(duden_funcs))

	const stdlib_testdata = "testdata/stdlib"
	called_functions := make(map[*ast.FuncDecl]struct{}, len(duden_funcs))
	err := filepath.WalkDir(stdlib_testdata, func(path string, d fs.DirEntry, err error) error {
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
			if _, ok := duden_funcs[f.Func]; ok {
				called_functions[f.Func] = struct{}{}
			}
		}))

		return nil
	})
	if err != nil {
		t.Fatalf("Error walking the test directory: %s", err)
	}

	t.Logf("Called functions: %d", len(called_functions))
	t.Logf("Uncovered functions: %d", len(duden_funcs)-len(called_functions))
	t.Logf("Coverage: %.2f%%", float64(len(called_functions))/float64(len(duden_funcs))*100)
}
