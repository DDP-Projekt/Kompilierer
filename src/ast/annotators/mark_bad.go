// This file contains an example annotator that attaches metadata to Bad nodes
// The metadata is a simple counter that is attached to each Bad node
//
// All the functionality in this file is for demonstration purposes only
package annotators

import (
	"fmt"

	"github.com/DDP-Projekt/Kompilierer/src/ast"
)

// example metadata
type BadMeta struct {
	n int
}

var _ ast.MetadataAttachment = (*BadMeta)(nil)

func (m *BadMeta) String() string {
	return fmt.Sprintf("BadMeta(%d)", m.n)
}

func (m *BadMeta) Kind() ast.MetadataKind {
	return "Bad"
}

// example annotator
type BadAnnotator struct {
	ast.BaseVisitor // provides useful utilities, like getting the current module
	n               int
}

// validate that the annotator implements the Annotator interface
// because of the ast.BaseVisitor embedding, the interface has to be implemented by the pointer type
var (
	_ ast.Annotator      = (*BadAnnotator)(nil)
	_ ast.BadDeclVisitor = (*BadAnnotator)(nil)
	_ ast.BadExprVisitor = (*BadAnnotator)(nil)
	_ ast.BadStmtVisitor = (*BadAnnotator)(nil)
)

func (a *BadAnnotator) VisitBadDecl(b *ast.BadDecl) ast.VisitResult {
	a.CurrentModule.Ast.AddAttachement(b, &BadMeta{n: a.n})
	a.n++
	return ast.VisitRecurse
}

func (a *BadAnnotator) VisitBadExpr(b *ast.BadExpr) ast.VisitResult {
	a.CurrentModule.Ast.AddAttachement(b, &BadMeta{n: a.n})
	a.n++
	return ast.VisitRecurse
}

func (a *BadAnnotator) VisitBadStmt(b *ast.BadStmt) ast.VisitResult {
	a.CurrentModule.Ast.AddAttachement(b, &BadMeta{n: a.n})
	a.n++
	return ast.VisitRecurse
}
