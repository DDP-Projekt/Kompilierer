package ast

import (
	"fmt"
	"strings"

	"github.com/DDP-Projekt/Kompilierer/src/token"
)

const (
	commentCutset = "[ \r\n]"
	commentFmt    = " [\n%s%.*s\n]"
)

// simple visitor to print an AST
type printer struct {
	ast          *Ast
	currentIdent int
	returned     string
}

func (pr *printer) printIndent() {
	for i := 0; i < pr.currentIdent; i++ {
		pr.print("   ")
	}
}

func (pr *printer) print(str string) {
	pr.returned += str
}

func (pr *printer) parenthesizeNode(name string, nodes ...Node) string {
	pr.print("(" + name)
	pr.currentIdent++

	for _, node := range nodes {
		pr.print("\n")
		pr.printIndent()
		node.Accept(pr)

		md, ok := pr.ast.GetMetadata(node)
		if !ok || len(md.Attachments) == 0 {
			continue
		}

		pr.print("\n")
		pr.printIndent()
		pr.print("Meta[")
		pr.currentIdent++

		for kind, attachement := range md.Attachments {
			pr.print("\n")
			pr.printIndent()
			pr.print(fmt.Sprintf("\"%s\": %s\n", kind, attachement))
		}
		pr.currentIdent--
		pr.printIndent()
		pr.print("]\n")
	}

	pr.currentIdent--
	if len(nodes) != 0 {
		pr.printIndent()
	}

	pr.print(")\n")
	return pr.returned
}

func (*printer) Visitor() {}

func (pr *printer) VisitBadDecl(decl *BadDecl) VisitResult {
	pr.parenthesizeNode(fmt.Sprintf("BadDecl[%s]", &decl.Tok))
	return VisitRecurse
}

func (pr *printer) VisitVarDecl(decl *VarDecl) VisitResult {
	msg := fmt.Sprintf("VarDecl[%s: %s]", decl.Name(), decl.Type)
	if decl.CommentTok != nil {
		msg += fmt.Sprintf(commentFmt, strings.Trim(decl.CommentTok.Literal, commentCutset), pr.currentIdent, " ")
	}
	pr.parenthesizeNode(msg, decl.InitVal)
	return VisitRecurse
}

func (pr *printer) VisitFuncDecl(decl *FuncDecl) VisitResult {
	msg := fmt.Sprintf("FuncDecl[%s: %v, %s]", decl.Name(), decl.Parameters, decl.ReturnType)
	if IsExternFunc(decl) {
		msg += " [Extern]"
	}
	if decl.IsExternVisible {
		msg += " [Extern Visible]"
	}
	if decl.CommentTok != nil {
		msg += fmt.Sprintf(commentFmt, strings.Trim(decl.CommentTok.Literal, commentCutset), pr.currentIdent, " ")
	}
	if IsExternFunc(decl) {
		pr.parenthesizeNode(msg)
	} else {
		pr.parenthesizeNode(msg, decl.Body)
	}
	return VisitRecurse
}

func (pr *printer) VisitStructDecl(decl *StructDecl) VisitResult {
	msg := fmt.Sprintf("StructDecl[%s: Public(%v)]", decl.Name(), decl.IsPublic)

	nodes := make([]Node, 0, len(decl.Fields))
	for _, v := range decl.Fields {
		nodes = append(nodes, v)
	}
	pr.parenthesizeNode(msg, nodes...)
	return VisitRecurse
}

func (pr *printer) VisitTypeAliasDecl(decl *TypeAliasDecl) VisitResult {
	pr.parenthesizeNode(fmt.Sprintf("TypeAliasDecl[%s: Public(%v)] = %s", decl.Name(), decl.IsPublic, decl.Underlying))
	return VisitRecurse
}

func (pr *printer) VisitTypeDefDecl(decl *TypeDefDecl) VisitResult {
	pr.parenthesizeNode(fmt.Sprintf("TypeDefDecl[%s: Public(%v)] = %s", decl.Name(), decl.IsPublic, decl.Underlying))
	return VisitRecurse
}

func (pr *printer) VisitBadExpr(expr *BadExpr) VisitResult {
	pr.parenthesizeNode(fmt.Sprintf("BadExpr[%s]", &expr.Tok))
	return VisitRecurse
}

func (pr *printer) VisitIdent(expr *Ident) VisitResult {
	pr.parenthesizeNode(fmt.Sprintf("Ident[%s]", expr.Literal.Literal))
	return VisitRecurse
}

func (pr *printer) VisitIndexing(expr *Indexing) VisitResult {
	pr.parenthesizeNode("Indexing", expr.Lhs, expr.Index)
	return VisitRecurse
}

func (pr *printer) VisitFieldAccess(expr *FieldAccess) VisitResult {
	pr.parenthesizeNode("FieldAccess", expr.Field, expr.Rhs)
	return VisitRecurse
}

func (pr *printer) VisitIntLit(expr *IntLit) VisitResult {
	pr.parenthesizeNode(fmt.Sprintf("IntLit(%d)", expr.Value))
	return VisitRecurse
}

func (pr *printer) VisitFloatLit(expr *FloatLit) VisitResult {
	pr.parenthesizeNode(fmt.Sprintf("FloatLit(%f)", expr.Value))
	return VisitRecurse
}

func (pr *printer) VisitBoolLit(expr *BoolLit) VisitResult {
	pr.parenthesizeNode(fmt.Sprintf("BoolLit(%v)", expr.Value))
	return VisitRecurse
}

func (pr *printer) VisitCharLit(expr *CharLit) VisitResult {
	pr.parenthesizeNode(fmt.Sprintf("CharLit(%c)", expr.Value))
	return VisitRecurse
}

func (pr *printer) VisitStringLit(expr *StringLit) VisitResult {
	pr.parenthesizeNode(fmt.Sprintf("StringLit[%s]", expr.Token().Literal))
	return VisitRecurse
}

func (pr *printer) VisitListLit(expr *ListLit) VisitResult {
	if expr.Values == nil {
		pr.parenthesizeNode(fmt.Sprintf("ListLit[%s]", expr.Type))
	} else {
		nodes := make([]Node, 0, len(expr.Values))
		for _, v := range expr.Values {
			nodes = append(nodes, v)
		}
		pr.parenthesizeNode("ListLit", nodes...)
	}
	return VisitRecurse
}

func (pr *printer) VisitUnaryExpr(expr *UnaryExpr) VisitResult {
	pr.parenthesizeNode(fmt.Sprintf("UnaryExpr[%s]", expr.Operator), expr.Rhs)
	return VisitRecurse
}

func (pr *printer) VisitBinaryExpr(expr *BinaryExpr) VisitResult {
	pr.parenthesizeNode(fmt.Sprintf("BinaryExpr[%s]", expr.Operator), expr.Lhs, expr.Rhs)
	return VisitRecurse
}

func (pr *printer) VisitTernaryExpr(expr *TernaryExpr) VisitResult {
	pr.parenthesizeNode(fmt.Sprintf("TernaryExpr[%s]", expr.Operator), expr.Lhs, expr.Mid, expr.Rhs)
	return VisitRecurse
}

func (pr *printer) VisitCastExpr(expr *CastExpr) VisitResult {
	pr.parenthesizeNode(fmt.Sprintf("CastExpr[%s]", expr.TargetType), expr.Lhs)
	return VisitRecurse
}

func (pr *printer) VisitTypeOpExpr(expr *TypeOpExpr) VisitResult {
	pr.parenthesizeNode(fmt.Sprintf("TypeOpExpr[%s]: %s", expr.Operator, expr.Rhs))
	return VisitRecurse
}

func (pr *printer) VisitGrouping(expr *Grouping) VisitResult {
	pr.parenthesizeNode("Grouping", expr.Expr)
	return VisitRecurse
}

func (pr *printer) VisitFuncCall(expr *FuncCall) VisitResult {
	args := make([]Node, 0, len(expr.Args))
	for _, v := range expr.Args {
		args = append(args, v)
	}
	pr.parenthesizeNode(fmt.Sprintf("FuncCall[%s]", expr.Name), args...)
	return VisitRecurse
}

func (pr *printer) VisitStructLiteral(expr *StructLiteral) VisitResult {
	args := make([]Node, 0, len(expr.Args))
	for _, v := range expr.Args {
		args = append(args, v)
	}
	pr.parenthesizeNode(fmt.Sprintf("StructLiteral[%s]", expr.Struct.Name()), args...)
	return VisitRecurse
}

func (pr *printer) VisitBadStmt(stmt *BadStmt) VisitResult {
	pr.parenthesizeNode(fmt.Sprintf("BadStmt[%s]", &stmt.Tok))
	return VisitRecurse
}

func (pr *printer) VisitDeclStmt(stmt *DeclStmt) VisitResult {
	pr.parenthesizeNode("DeclStmt", stmt.Decl)
	return VisitRecurse
}

func (pr *printer) VisitExprStmt(stmt *ExprStmt) VisitResult {
	pr.parenthesizeNode("ExprStmt", stmt.Expr)
	return VisitRecurse
}

func (pr *printer) VisitImportStmt(stmt *ImportStmt) VisitResult {
	if stmt.Module == nil {
		return VisitRecurse
	}
	// TODO: pretty print imports
	nodes := make([]Node, 0)

	IterateImportedDecls(stmt, func(_ string, decl Declaration, _ token.Token) bool {
		if decl != nil {
			nodes = append(nodes, decl)
		}
		return true
	})

	pr.parenthesizeNode("ImportStmt", nodes...)
	return VisitRecurse
}

func (pr *printer) VisitAssignStmt(stmt *AssignStmt) VisitResult {
	pr.parenthesizeNode("AssignStmt", stmt.Var, stmt.Rhs)
	return VisitRecurse
}

func (pr *printer) VisitBlockStmt(stmt *BlockStmt) VisitResult {
	args := make([]Node, len(stmt.Statements))
	for i, v := range stmt.Statements {
		args[i] = v
	}
	pr.parenthesizeNode("BlockStmt", args...)
	return VisitRecurse
}

func (pr *printer) VisitIfStmt(stmt *IfStmt) VisitResult {
	if stmt.Else != nil {
		pr.parenthesizeNode("IfStmt", stmt.Condition, stmt.Then, stmt.Else)
	} else {
		pr.parenthesizeNode("IfStmt", stmt.Condition, stmt.Then)
	}
	return VisitRecurse
}

func (pr *printer) VisitWhileStmt(stmt *WhileStmt) VisitResult {
	pr.parenthesizeNode("WhileStmt", stmt.Condition, stmt.Body)
	return VisitRecurse
}

func (pr *printer) VisitForStmt(stmt *ForStmt) VisitResult {
	pr.parenthesizeNode("ForStmt", stmt.Initializer, stmt.To, stmt.StepSize, stmt.Body)
	return VisitRecurse
}

func (pr *printer) VisitForRangeStmt(stmt *ForRangeStmt) VisitResult {
	pr.parenthesizeNode("ForRangeStmt", stmt.Initializer, stmt.In, stmt.Body)
	return VisitRecurse
}

func (pr *printer) VisitBreakContinueStmt(stmt *BreakContinueStmt) VisitResult {
	if stmt.Tok.Type == token.VERLASSE {
		pr.parenthesizeNode("BreakContinueStmt[break]")
	} else {
		pr.parenthesizeNode("BreakContinueStmt[continue]")
	}
	return VisitRecurse
}

func (pr *printer) VisitReturnStmt(stmt *ReturnStmt) VisitResult {
	if stmt.Value == nil {
		pr.parenthesizeNode("ReturnStmt[void]")
	} else {
		pr.parenthesizeNode("ReturnStmt", stmt.Value)
	}
	return VisitRecurse
}

func literals(tokens []token.Token) []string {
	result := make([]string, 0, len(tokens))
	for _, v := range tokens {
		result = append(result, v.Literal)
	}
	return result
}

func commentLiterals(comments []*token.Token) string {
	result := make([]string, 0, len(comments))
	for _, v := range comments {
		if v == nil {
			result = append(result, "nil")
		} else {
			result = append(result, strings.Trim(v.Literal, commentCutset))
		}
	}
	return "[" + strings.Join(result, ", ") + "]"
}
