package format

import (
	"bytes"

	"github.com/DDP-Projekt/Kompilierer/src/ast"
)

type formatter struct {
	result bytes.Buffer
	err error
}

var (
	_ ast.FullVisitor = (*formatter)(nil)
)

func (*formatter) Visitor() {}

func (f *formatter) VisitBadDecl(d *ast.BadDecl) ast.VisitResult {
	return ast.VisitRecurse
}
func (f *formatter) VisitVarDecl(d *ast.VarDecl) ast.VisitResult {
	return ast.VisitRecurse
}
func (f *formatter) VisitFuncDecl(d *ast.FuncDecl) ast.VisitResult {
	return ast.VisitRecurse
}
func (f *formatter) VisitStructDecl(d *ast.StructDecl) ast.VisitResult {
	return ast.VisitRecurse
}
func (f *formatter) VisitAliasDecl(d *ast.TypeAliasDecl) ast.VisitResult {
	return ast.VisitRecurse
}
func (f *formatter) VisitDefDecl(d *ast.TypeDefDecl) ast.VisitResult {
	return ast.VisitRecurse
}

func (f *formatter) VisitBadExpr(e *ast.BadExpr) ast.VisitResult {
	return ast.VisitRecurse
}
func (f *formatter) VisitIdent(e *ast.Ident) ast.VisitResult {
	return ast.VisitRecurse
}
func (f *formatter) VisitBadExpr(e *ast.Indexing) ast.VisitResult {
	return ast.VisitRecurse
}
func (f *formatter) VisitBadExpr(e *ast.FieldAccess) ast.VisitResult {
	return ast.VisitRecurse
}
func (f *formatter) VisitBadExpr(e *ast.IntLit) ast.VisitResult {
	return ast.VisitRecurse
}
func (f *formatter) VisitBadExpr(e *ast.FloatLit) ast.VisitResult {
	return ast.VisitRecurse
}
func (f *formatter) VisitBadExpr(e *ast.BoolLit) ast.VisitResult {
	return ast.VisitRecurse
}
func (f *formatter) VisitBadExpr(e *ast.CharLit) ast.VisitResult {
	return ast.VisitRecurse
}
func (f *formatter) VisitBadExpr(e *ast.StringLit) ast.VisitResult {
	return ast.VisitRecurse
}
func (f *formatter) VisitBadExpr(e *ast.ListLit) ast.VisitResult {
	return ast.VisitRecurse
}
func (f *formatter) VisitBadExpr(e *ast.UnaryExpr) ast.VisitResult {
	return ast.VisitRecurse
}
func (f *formatter) VisitBadExpr(e *ast.BinaryExpr) ast.VisitResult {
	return ast.VisitRecurse
}
func (f *formatter) VisitBadExpr(e *ast.TernaryExpr) ast.VisitResult {
	return ast.VisitRecurse
}
func (f *formatter) VisitBadExpr(e *ast.CastExpr) ast.VisitResult {
	return ast.VisitRecurse
}
func (f *formatter) VisitBadExpr(e *ast.TypeOpExpr) ast.VisitResult {
	return ast.VisitRecurse
}
func (f *formatter) VisitBadExpr(e *ast.TypeCheck) ast.VisitResult {
	return ast.VisitRecurse
}
func (f *formatter) VisitBadExpr(e *ast.Grouping) ast.VisitResult {
	return ast.VisitRecurse
}
func (f *formatter) VisitBadExpr(e *ast.FuncCall) ast.VisitResult {
	return ast.VisitRecurse
}
func (f *formatter) VisitBadExpr(e *ast.StructLiteral) ast.VisitResult {
	return ast.VisitRecurse
}

func (f *formatter) 
