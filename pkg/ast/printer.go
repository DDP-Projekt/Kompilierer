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

func (pr *printer) VisitBadDecl(decl *BadDecl) Visitor {
	pr.parenthesizeNode(fmt.Sprintf("BadDecl[%s]", decl.Tok))
	return pr
}
func (pr *printer) VisitVarDecl(decl *VarDecl) Visitor {
	pr.parenthesizeNode(fmt.Sprintf("VarDecl[%s]", decl.Name.Literal), decl.InitVal)
	return pr
}
func (pr *printer) VisitFuncDecl(decl *FuncDecl) Visitor {
	if IsExternFunc(decl) {
		pr.parenthesizeNode(fmt.Sprintf("FuncDecl[%s: %v, %v, %s]", decl.Name.Literal, tokenSlice(decl.ParamNames).literals(), decl.ParamTypes, decl.Type), decl.Body)
	} else {
		pr.parenthesizeNode(fmt.Sprintf("FuncDecl[%s: %v, %v, %s] Extern", decl.Name.Literal, tokenSlice(decl.ParamNames).literals(), decl.ParamTypes, decl.Type))
	}
	return pr
}

func (pr *printer) VisitBadExpr(expr *BadExpr) Visitor {
	pr.parenthesizeNode(fmt.Sprintf("BadExpr[%s]", expr.Tok))
	return pr
}
func (pr *printer) VisitIdent(expr *Ident) Visitor {
	pr.parenthesizeNode(fmt.Sprintf("Ident[%s]", expr.Literal.Literal))
	return pr
}
func (pr *printer) VisitIndexing(expr *Indexing) Visitor {
	pr.parenthesizeNode("Indexing", expr.Lhs, expr.Index)
	return pr
}
func (pr *printer) VisitIntLit(expr *IntLit) Visitor {
	pr.parenthesizeNode(fmt.Sprintf("IntLit(%d)", expr.Value))
	return pr
}
func (pr *printer) VisitFLoatLit(expr *FloatLit) Visitor {
	pr.parenthesizeNode(fmt.Sprintf("FloatLit(%f)", expr.Value))
	return pr
}
func (pr *printer) VisitBoolLit(expr *BoolLit) Visitor {
	pr.parenthesizeNode(fmt.Sprintf("BoolLit(%v)", expr.Value))
	return pr
}
func (pr *printer) VisitCharLit(expr *CharLit) Visitor {
	pr.parenthesizeNode(fmt.Sprintf("CharLit(%c)", expr.Value))
	return pr
}
func (pr *printer) VisitStringLit(expr *StringLit) Visitor {
	pr.parenthesizeNode(fmt.Sprintf("StringLit[%s]", expr.Token().Literal))
	return pr
}
func (pr *printer) VisitListLit(expr *ListLit) Visitor {
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
func (pr *printer) VisitUnaryExpr(expr *UnaryExpr) Visitor {
	pr.parenthesizeNode(fmt.Sprintf("UnaryExpr[%s]", expr.Operator), expr.Rhs)
	return pr
}
func (pr *printer) VisitBinaryExpr(expr *BinaryExpr) Visitor {
	pr.parenthesizeNode(fmt.Sprintf("BinaryExpr[%s]", expr.Operator), expr.Lhs, expr.Rhs)
	return pr
}
func (pr *printer) VisitTernaryExpr(expr *TernaryExpr) Visitor {
	pr.parenthesizeNode(fmt.Sprintf("TernaryExpr[%s]", expr.Operator), expr.Lhs, expr.Mid, expr.Rhs)
	return pr
}
func (pr *printer) VisitCastExpr(expr *CastExpr) Visitor {
	pr.parenthesizeNode(fmt.Sprintf("CastExpr[%s]", expr.Type), expr.Lhs)
	return pr
}
func (pr *printer) VisitGrouping(expr *Grouping) Visitor {
	pr.parenthesizeNode("Grouping", expr.Expr)
	return pr
}
func (pr *printer) VisitFuncCall(expr *FuncCall) Visitor {
	args := make([]Node, 0)
	for _, v := range expr.Args {
		args = append(args, v)
	}
	pr.parenthesizeNode(fmt.Sprintf("FuncCall(%s)", expr.Name), args...)
	return pr
}

func (pr *printer) VisitBadStmt(stmt *BadStmt) Visitor {
	pr.parenthesizeNode(fmt.Sprintf("BadStmt[%s]", stmt.Tok))
	return pr
}
func (pr *printer) VisitDeclStmt(stmt *DeclStmt) Visitor {
	pr.parenthesizeNode("DeclStmt", stmt.Decl)
	return pr
}
func (pr *printer) VisitExprStmt(stmt *ExprStmt) Visitor {
	pr.parenthesizeNode("ExprStmt", stmt.Expr)
	return pr
}
func (pr *printer) VisitAssignStmt(stmt *AssignStmt) Visitor {
	pr.parenthesizeNode("AssignStmt", stmt.Var, stmt.Rhs)
	return pr
}
func (pr *printer) VisitBlockStmt(stmt *BlockStmt) Visitor {
	args := make([]Node, len(stmt.Statements))
	for i, v := range stmt.Statements {
		args[i] = v
	}
	pr.parenthesizeNode("BlockStmt", args...)
	return pr
}
func (pr *printer) VisitIfStmt(stmt *IfStmt) Visitor {
	if stmt.Else != nil {
		pr.parenthesizeNode("IfStmt", stmt.Condition, stmt.Then, stmt.Else)
	} else {
		pr.parenthesizeNode("IfStmt", stmt.Condition, stmt.Then)
	}
	return pr
}
func (pr *printer) VisitWhileStmt(stmt *WhileStmt) Visitor {
	pr.parenthesizeNode("WhileStmt", stmt.Condition, stmt.Body)
	return pr
}
func (pr *printer) VisitForStmt(stmt *ForStmt) Visitor {
	pr.parenthesizeNode("ForStmt", stmt.Initializer, stmt.To, stmt.StepSize, stmt.Body)
	return pr
}
func (pr *printer) VisitForRangeStmt(stmt *ForRangeStmt) Visitor {
	pr.parenthesizeNode("ForRangeStmt", stmt.Initializer, stmt.In, stmt.Body)
	return pr
}
func (pr *printer) VisitFuncCallStmt(stmt *FuncCallStmt) Visitor {
	pr.parenthesizeNode("FuncCallStmt", stmt.Call)
	return pr
}
func (pr *printer) VisitReturnStmt(stmt *ReturnStmt) Visitor {
	pr.parenthesizeNode("ReturnStmt", stmt.Value)
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
