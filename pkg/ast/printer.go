package ast

import (
	"fmt"

	"github.com/DDP-Projekt/Kompilierer/pkg/token"
)

// simple visitor to print an AST
type printer struct {
	currentIdent int
	returned     string
}

// print the AST to stdout
func (ast *Ast) Print() {
	printer := &printer{}
	WalkAst(ast, printer)
	fmt.Println(printer.returned)
}

func (ast *Ast) String() string {
	printer := &printer{}
	WalkAst(ast, printer)
	return printer.returned
}

func (pr *printer) printIdent() {
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
		pr.printIdent()
		node.Accept(pr)
	}

	pr.currentIdent--
	if len(nodes) != 0 {
		pr.printIdent()
	}

	pr.print(")\n")
	return pr.returned
}

func (pr *printer) VisitBadDecl(decl *BadDecl) FullVisitor {
	pr.parenthesizeNode(fmt.Sprintf("BadDecl[%s]", decl.Tok))
	return pr
}
func (pr *printer) VisitVarDecl(decl *VarDecl) FullVisitor {
	pr.parenthesizeNode(fmt.Sprintf("VarDecl[%s]", decl.Name.Literal), decl.InitVal)
	return pr
}
func (pr *printer) VisitFuncDecl(decl *FuncDecl) FullVisitor {
	if IsExternFunc(decl) {
		pr.parenthesizeNode(fmt.Sprintf("FuncDecl[%s: %v, %v, %s] Extern", decl.Name.Literal, tokenSlice(decl.ParamNames).literals(), decl.ParamTypes, decl.Type))
	} else {
		pr.parenthesizeNode(fmt.Sprintf("FuncDecl[%s: %v, %v, %s]", decl.Name.Literal, tokenSlice(decl.ParamNames).literals(), decl.ParamTypes, decl.Type), decl.Body)
	}
	return pr
}

func (pr *printer) VisitBadExpr(expr *BadExpr) FullVisitor {
	pr.parenthesizeNode(fmt.Sprintf("BadExpr[%s]", expr.Tok))
	return pr
}
func (pr *printer) VisitIdent(expr *Ident) FullVisitor {
	pr.parenthesizeNode(fmt.Sprintf("Ident[%s]", expr.Literal.Literal))
	return pr
}
func (pr *printer) VisitIndexing(expr *Indexing) FullVisitor {
	pr.parenthesizeNode("Indexing", expr.Lhs, expr.Index)
	return pr
}
func (pr *printer) VisitIntLit(expr *IntLit) FullVisitor {
	pr.parenthesizeNode(fmt.Sprintf("IntLit(%d)", expr.Value))
	return pr
}
func (pr *printer) VisitFloatLit(expr *FloatLit) FullVisitor {
	pr.parenthesizeNode(fmt.Sprintf("FloatLit(%f)", expr.Value))
	return pr
}
func (pr *printer) VisitBoolLit(expr *BoolLit) FullVisitor {
	pr.parenthesizeNode(fmt.Sprintf("BoolLit(%v)", expr.Value))
	return pr
}
func (pr *printer) VisitCharLit(expr *CharLit) FullVisitor {
	pr.parenthesizeNode(fmt.Sprintf("CharLit(%c)", expr.Value))
	return pr
}
func (pr *printer) VisitStringLit(expr *StringLit) FullVisitor {
	pr.parenthesizeNode(fmt.Sprintf("StringLit[%s]", expr.Token().Literal))
	return pr
}
func (pr *printer) VisitListLit(expr *ListLit) FullVisitor {
	if expr.Values == nil {
		pr.parenthesizeNode(fmt.Sprintf("ListLit[%s]", expr.Type))
	} else {
		nodes := make([]Node, 0, len(expr.Values))
		for _, v := range expr.Values {
			nodes = append(nodes, v)
		}
		pr.parenthesizeNode("ListLit", nodes...)
	}
	return pr
}
func (pr *printer) VisitUnaryExpr(expr *UnaryExpr) FullVisitor {
	pr.parenthesizeNode(fmt.Sprintf("UnaryExpr[%s]", expr.Operator), expr.Rhs)
	return pr
}
func (pr *printer) VisitBinaryExpr(expr *BinaryExpr) FullVisitor {
	pr.parenthesizeNode(fmt.Sprintf("BinaryExpr[%s]", expr.Operator), expr.Lhs, expr.Rhs)
	return pr
}
func (pr *printer) VisitTernaryExpr(expr *TernaryExpr) FullVisitor {
	pr.parenthesizeNode(fmt.Sprintf("TernaryExpr[%s]", expr.Operator), expr.Lhs, expr.Mid, expr.Rhs)
	return pr
}
func (pr *printer) VisitCastExpr(expr *CastExpr) FullVisitor {
	pr.parenthesizeNode(fmt.Sprintf("CastExpr[%s]", expr.Type), expr.Lhs)
	return pr
}
func (pr *printer) VisitGrouping(expr *Grouping) FullVisitor {
	pr.parenthesizeNode("Grouping", expr.Expr)
	return pr
}
func (pr *printer) VisitFuncCall(expr *FuncCall) FullVisitor {
	args := make([]Node, 0)
	for _, v := range expr.Args {
		args = append(args, v)
	}
	pr.parenthesizeNode(fmt.Sprintf("FuncCall(%s)", expr.Name), args...)
	return pr
}

func (pr *printer) VisitBadStmt(stmt *BadStmt) FullVisitor {
	pr.parenthesizeNode(fmt.Sprintf("BadStmt[%s]", stmt.Tok))
	return pr
}
func (pr *printer) VisitDeclStmt(stmt *DeclStmt) FullVisitor {
	pr.parenthesizeNode("DeclStmt", stmt.Decl)
	return pr
}
func (pr *printer) VisitExprStmt(stmt *ExprStmt) FullVisitor {
	pr.parenthesizeNode("ExprStmt", stmt.Expr)
	return pr
}
func (pr *printer) VisitAssignStmt(stmt *AssignStmt) FullVisitor {
	pr.parenthesizeNode("AssignStmt", stmt.Var, stmt.Rhs)
	return pr
}
func (pr *printer) VisitBlockStmt(stmt *BlockStmt) FullVisitor {
	args := make([]Node, len(stmt.Statements))
	for i, v := range stmt.Statements {
		args[i] = v
	}
	pr.parenthesizeNode("BlockStmt", args...)
	return pr
}
func (pr *printer) VisitIfStmt(stmt *IfStmt) FullVisitor {
	if stmt.Else != nil {
		pr.parenthesizeNode("IfStmt", stmt.Condition, stmt.Then, stmt.Else)
	} else {
		pr.parenthesizeNode("IfStmt", stmt.Condition, stmt.Then)
	}
	return pr
}
func (pr *printer) VisitWhileStmt(stmt *WhileStmt) FullVisitor {
	pr.parenthesizeNode("WhileStmt", stmt.Condition, stmt.Body)
	return pr
}
func (pr *printer) VisitForStmt(stmt *ForStmt) FullVisitor {
	pr.parenthesizeNode("ForStmt", stmt.Initializer, stmt.To, stmt.StepSize, stmt.Body)
	return pr
}
func (pr *printer) VisitForRangeStmt(stmt *ForRangeStmt) FullVisitor {
	pr.parenthesizeNode("ForRangeStmt", stmt.Initializer, stmt.In, stmt.Body)
	return pr
}
func (pr *printer) VisitReturnStmt(stmt *ReturnStmt) FullVisitor {
	if stmt.Value == nil {
		pr.parenthesizeNode("ReturnStmt[void]")
	} else {
		pr.parenthesizeNode("ReturnStmt", stmt.Value)
	}
	return pr
}

type tokenSlice []token.Token

func (tokens tokenSlice) literals() []string {
	result := make([]string, 0, len(tokens))
	for _, v := range tokens {
		result = append(result, v.Literal)
	}
	return result
}
