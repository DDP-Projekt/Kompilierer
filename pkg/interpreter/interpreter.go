package interpreter

import (
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/DDP-Projekt/Kompilierer/pkg/ast"
	"github.com/DDP-Projekt/Kompilierer/pkg/scanner"
	"github.com/DDP-Projekt/Kompilierer/pkg/token"
)

type Interpreter struct {
	ast                *ast.Ast
	currentEnvironment *environment
	lastReturn         value
	errorHandler       scanner.ErrorHandler

	// standard data streams
	Stdout io.Writer
	Stdin  io.Reader
	Stderr io.Writer
}

// returns a new Interpreter ready to interpret it's Ast
func New(Ast *ast.Ast, errorHandler scanner.ErrorHandler) *Interpreter {
	if errorHandler == nil {
		errorHandler = func(string) {}
	}
	i := &Interpreter{
		ast:                Ast,
		currentEnvironment: newEnvironment(nil),
		lastReturn:         nil,
		errorHandler:       errorHandler,
		Stdout:             os.Stdout,
		Stdin:              os.Stdin,
		Stderr:             os.Stderr,
	}
	return i
}

func err(tok token.Token, msg string) {
	panic(fmt.Errorf("Laufzeit Fehler in %s in Zeile %d, Spalte %d: %s", tok.File, tok.Line, tok.Column, msg))
}

// interpret the Ast
// the interpreter should have been created using the New function
func (i *Interpreter) Interpret() (result error) {
	// recover from runtime errors
	defer func() {
		if err := recover(); err != nil {
			result = err.(error)
		}
	}()

	if i.ast.Faulty {
		return fmt.Errorf("faulty ast")
	}

	for _, stmt := range i.ast.Statements {
		i.execute(stmt)
	}

	return nil
}

func (i *Interpreter) visitNode(node ast.Node) {
	i = node.Accept(i).(*Interpreter)
}

func (i *Interpreter) VisitBadDecl(d *ast.BadDecl) ast.Visitor {
	err(d.Token(), "Es wurde eine invalide Deklaration gefunden")
	return i
}
func (i *Interpreter) VisitVarDecl(d *ast.VarDecl) ast.Visitor {
	i.currentEnvironment.addVar(d.Name.Literal, i.evaluate(d.InitVal))
	return i
}
func (i *Interpreter) VisitFuncDecl(d *ast.FuncDecl) ast.Visitor {
	i.currentEnvironment.addFunc(d.Name.Literal, d)
	return i
}

func (i *Interpreter) evaluate(expr ast.Expression) value {
	i.visitNode(expr)
	return i.lastReturn
}

func (i *Interpreter) execute(stmt ast.Statement) {
	i.visitNode(stmt)
}

func isInbuiltFunc(fun *ast.FuncDecl) bool {
	return strings.HasPrefix(fun.Name.Literal, "§")
}

func (i *Interpreter) exitEnvironment() {
	i.currentEnvironment = i.currentEnvironment.enclosing
}

func (i *Interpreter) VisitBadExpr(e *ast.BadExpr) ast.Visitor {
	err(e.Token(), "Es wurde ein invalider Ausdruck gefunden")
	return i
}
func (i *Interpreter) VisitIdent(e *ast.Ident) ast.Visitor {
	if v, exists := i.currentEnvironment.lookupVar(e.Literal.Literal); !exists {
		err(e.Token(), fmt.Sprintf("Die Variable '%s' wurde noch nicht deklariert", e.Literal.Literal))
	} else {
		i.lastReturn = v
	}
	return i
}
func (i *Interpreter) VisitIntLit(e *ast.IntLit) ast.Visitor {
	i.lastReturn = ddpint(e.Value)
	return i
}
func (i *Interpreter) VisitFLoatLit(e *ast.FloatLit) ast.Visitor {
	i.lastReturn = ddpfloat(e.Value)
	return i
}
func (i *Interpreter) VisitBoolLit(e *ast.BoolLit) ast.Visitor {
	i.lastReturn = ddpbool(e.Value)
	return i
}
func (i *Interpreter) VisitCharLit(e *ast.CharLit) ast.Visitor {
	i.lastReturn = ddpchar(e.Value)
	return i
}
func (i *Interpreter) VisitStringLit(e *ast.StringLit) ast.Visitor {
	i.lastReturn = ddpstring(e.Value)
	return i
}
func (i *Interpreter) VisitUnaryExpr(e *ast.UnaryExpr) ast.Visitor {
	rhs := i.evaluate(e.Rhs)
	switch e.Operator.Type {
	case token.BETRAG:
		switch v := rhs.(type) {
		case ddpfloat:
			i.lastReturn = ddpfloat(math.Abs(float64(v)))
		case ddpint:
			if v < 0 {
				v = -v
			}
			i.lastReturn = v
		}
	case token.NEGATE:
		switch v := rhs.(type) {
		case ddpfloat:
			i.lastReturn = -v
		case ddpint:
			i.lastReturn = -v
		}
	case token.NICHT:
		v := rhs.(ddpbool)
		i.lastReturn = !v
	case token.LÄNGE:
		v := rhs.(ddpstring)
		i.lastReturn = ddpint(len(v))
	case token.GRÖßE:
		switch rhs.(type) {
		case ddpint, ddpfloat:
			i.lastReturn = ddpint(4)
		case ddpbool:
			i.lastReturn = ddpint(1)
		case ddpchar:
			i.lastReturn = ddpint(2)
		case ddpstring:
			i.lastReturn = ddpint(2 * len(rhs.(ddpstring)))
		}
	case token.ZAHL:
		switch v := rhs.(type) {
		case ddpint:
			i.lastReturn = v
		case ddpfloat:
			i.lastReturn = ddpint(v)
		case ddpbool:
			if v {
				i.lastReturn = ddpint(1)
			} else {
				i.lastReturn = ddpint(0)
			}
		case ddpchar:
			i.lastReturn = ddpint(v)
		case ddpstring:
			if result, Err := strconv.ParseInt(string(v), 10, 64); Err != nil {
				err(e.Token(), fmt.Sprintf("Der Text '%s' kann nicht in eine Zahl umgewandelt werden", string(v)))
			} else {
				i.lastReturn = ddpint(result)
			}
		}
	case token.KOMMAZAHL:
		switch v := rhs.(type) {
		case ddpint:
			i.lastReturn = ddpfloat(v)
		case ddpfloat:
			i.lastReturn = v
		case ddpstring:
			if result, Err := strconv.ParseFloat(string(v), 64); Err != nil {
				err(e.Token(), fmt.Sprintf("Der Text '%s' kann nicht in eine Kommazahl umgewandelt werden", string(v)))
			} else {
				i.lastReturn = ddpfloat(result)
			}
		}
	case token.BOOLEAN:
		switch v := rhs.(type) {
		case ddpint:
			if v == 0 {
				i.lastReturn = ddpbool(false)
			} else {
				i.lastReturn = ddpbool(true)
			}
		case ddpbool:
			i.lastReturn = v
		}
	case token.BUCHSTABE:
		switch v := rhs.(type) {
		case ddpint:
			i.lastReturn = ddpchar(v)
		case ddpchar:
			i.lastReturn = v
		}
	case token.TEXT:
		switch v := rhs.(type) {
		case ddpint:
			i.lastReturn = ddpstring(strconv.FormatInt(int64(v), 64))
		case ddpfloat:
			i.lastReturn = ddpstring(strconv.FormatFloat(float64(v), 'f', -1, 64))
		case ddpchar:
			i.lastReturn = ddpstring(v)
		case ddpstring:
			i.lastReturn = v
		}
	default:
		err(e.Token(), fmt.Sprintf("Unbekannter Operator '%s'", e.Operator.String()))
	}
	return i
}
func (i *Interpreter) VisitBinaryExpr(e *ast.BinaryExpr) ast.Visitor {
	lhs := i.evaluate(e.Lhs)
	rhs := i.evaluate(e.Rhs)
	switch e.Operator.Type {
	case token.VERKETTET:
		switch left := lhs.(type) {
		case ddpstring:
			switch right := rhs.(type) {
			case ddpstring:
				i.lastReturn = left + right
			case ddpchar:
				i.lastReturn = left + ddpstring(rune(right))
			}
		case ddpchar:
			switch right := rhs.(type) {
			case ddpstring:
				i.lastReturn = ddpstring(rune(left)) + right
			case ddpchar:
				i.lastReturn = ddpstring(rune(left)) + ddpstring(rune(right))
			}
		}
	case token.PLUS, token.ADDIERE, token.ERHÖHE:
		switch left := lhs.(type) {
		case ddpint:
			switch right := rhs.(type) {
			case ddpint:
				i.lastReturn = left + right
			case ddpfloat:
				i.lastReturn = ddpfloat(left) + right
			}
		case ddpfloat:
			switch right := rhs.(type) {
			case ddpint:
				i.lastReturn = left + ddpfloat(right)
			case ddpfloat:
				i.lastReturn = left + right
			}
		}
	case token.MINUS, token.SUBTRAHIERE, token.VERRINGERE:
		switch left := lhs.(type) {
		case ddpint:
			switch right := rhs.(type) {
			case ddpint:
				i.lastReturn = left - right
			case ddpfloat:
				i.lastReturn = ddpfloat(left) - right
			}
		case ddpfloat:
			switch right := rhs.(type) {
			case ddpint:
				i.lastReturn = left - ddpfloat(right)
			case ddpfloat:
				i.lastReturn = left - right
			}
		}
	case token.MAL, token.MULTIPLIZIERE, token.VERVIELFACHE:
		switch left := lhs.(type) {
		case ddpint:
			switch right := rhs.(type) {
			case ddpint:
				i.lastReturn = left * right
			case ddpfloat:
				i.lastReturn = ddpfloat(left) * right
			}
		case ddpfloat:
			switch right := rhs.(type) {
			case ddpint:
				i.lastReturn = left * ddpfloat(right)
			case ddpfloat:
				i.lastReturn = left * right
			}
		}
	case token.DURCH, token.DIVIDIERE, token.TEILE:
		switch left := lhs.(type) {
		case ddpint:
			switch right := rhs.(type) {
			case ddpint:
				i.lastReturn = ddpfloat(left) / ddpfloat(right)
			case ddpfloat:
				i.lastReturn = ddpfloat(left) / right
			}
		case ddpfloat:
			switch right := rhs.(type) {
			case ddpint:
				i.lastReturn = left / ddpfloat(right)
			case ddpfloat:
				i.lastReturn = left / right
			}
		}
	case token.HOCH:
		switch left := lhs.(type) {
		case ddpint:
			switch right := rhs.(type) {
			case ddpint:
				i.lastReturn = ddpfloat(math.Pow(float64(left), float64(right)))
			case ddpfloat:
				i.lastReturn = ddpfloat(math.Pow(float64(left), float64(right)))
			}
		case ddpfloat:
			switch right := rhs.(type) {
			case ddpint:
				i.lastReturn = ddpfloat(math.Pow(float64(left), float64(right)))
			case ddpfloat:
				i.lastReturn = ddpfloat(math.Pow(float64(left), float64(right)))
			}
		}
	case token.MODULO:
		i.lastReturn = ddpint(lhs.(ddpint) % rhs.(ddpint))
	case token.UND:
		i.lastReturn = lhs.(ddpbool) && rhs.(ddpbool)
	case token.ODER:
		i.lastReturn = lhs.(ddpbool) || rhs.(ddpbool)
	case token.LINKS:
		i.lastReturn = lhs.(ddpint) << rhs.(ddpint)
	case token.RECHTS:
		i.lastReturn = lhs.(ddpint) >> rhs.(ddpint)
	case token.GLEICH:
		i.lastReturn = ddpbool(lhs == rhs)
	case token.UNGLEICH:
		i.lastReturn = ddpbool(lhs != rhs)
	case token.KLEINER:
		switch left := lhs.(type) {
		case ddpint:
			switch right := rhs.(type) {
			case ddpint:
				i.lastReturn = ddpbool(left < right)
			case ddpfloat:
				i.lastReturn = ddpbool(ddpfloat(left) < right)
			}
		case ddpfloat:
			switch right := rhs.(type) {
			case ddpint:
				i.lastReturn = ddpbool(left < ddpfloat(right))
			case ddpfloat:
				i.lastReturn = ddpbool(left < right)
			}
		}
	case token.KLEINERODER:
		switch left := lhs.(type) {
		case ddpint:
			switch right := rhs.(type) {
			case ddpint:
				i.lastReturn = ddpbool(left <= right)
			case ddpfloat:
				i.lastReturn = ddpbool(ddpfloat(left) <= right)
			}
		case ddpfloat:
			switch right := rhs.(type) {
			case ddpint:
				i.lastReturn = ddpbool(left <= ddpfloat(right))
			case ddpfloat:
				i.lastReturn = ddpbool(left <= right)
			}
		}
	case token.GRÖßER:
		switch left := lhs.(type) {
		case ddpint:
			switch right := rhs.(type) {
			case ddpint:
				i.lastReturn = ddpbool(left > right)
			case ddpfloat:
				i.lastReturn = ddpbool(ddpfloat(left) > right)
			}
		case ddpfloat:
			switch right := rhs.(type) {
			case ddpint:
				i.lastReturn = ddpbool(left > ddpfloat(right))
			case ddpfloat:
				i.lastReturn = ddpbool(left > right)
			}
		}
	case token.GRÖßERODER:
		switch left := lhs.(type) {
		case ddpint:
			switch right := rhs.(type) {
			case ddpint:
				i.lastReturn = ddpbool(left >= right)
			case ddpfloat:
				i.lastReturn = ddpbool(ddpfloat(left) >= right)
			}
		case ddpfloat:
			switch right := rhs.(type) {
			case ddpint:
				i.lastReturn = ddpbool(left >= ddpfloat(right))
			case ddpfloat:
				i.lastReturn = ddpbool(left >= right)
			}
		}
	}
	return i
}
func (i *Interpreter) VisitGrouping(e *ast.Grouping) ast.Visitor {
	return e.Expr.Accept(i)
}
func (i *Interpreter) VisitFuncCall(e *ast.FuncCall) ast.Visitor {
	if fun, exists := i.currentEnvironment.lookupFunc(e.Name); exists {
		i.currentEnvironment = newEnvironment(i.currentEnvironment)
		defer i.exitEnvironment() // we have to defer it due to return panic issues

		for k, v := range e.Args {
			i.currentEnvironment.addVar(k, i.evaluate(v))
		}

		if isInbuiltFunc(fun) {
			i.lastReturn = i.callInbuilt(e.Name)
		} else {
			// execute the function body, a panic may return a value
			var returnValue value = nil
			func() {
				defer func() {
					if v, ok := recover().(value); ok || v == nil {
						returnValue = v
					} else {
						panic(v) // repanic if there is no value
					}
				}() // defer this to catch return statements
				i.execute(fun.Body)
			}()
			i.lastReturn = returnValue
		}
	} else {
		err(e.Token(), fmt.Sprintf("Die Funktion '%s' wurde noch nicht deklariert", e.Name))
	}
	return i
}

func (i *Interpreter) VisitBadStmt(s *ast.BadStmt) ast.Visitor {
	err(s.Token(), "Es wurde eine invalide Aussage gefunden")
	return i
}
func (i *Interpreter) VisitDeclStmt(s *ast.DeclStmt) ast.Visitor {
	return s.Decl.Accept(i)
}
func (i *Interpreter) VisitExprStmt(s *ast.ExprStmt) ast.Visitor {
	return s.Expr.Accept(i)
}
func (i *Interpreter) VisitAssignStmt(s *ast.AssignStmt) ast.Visitor {
	i.currentEnvironment.updateVar(s.Name.Literal, i.evaluate(s.Rhs))
	return i
}
func (i *Interpreter) VisitBlockStmt(s *ast.BlockStmt) ast.Visitor {
	i.currentEnvironment = newEnvironment(i.currentEnvironment)
	defer i.exitEnvironment() // we have to defer it due to return panic issues
	for _, stmt := range s.Statements {
		i.execute(stmt)
	}
	return i
}
func (i *Interpreter) VisitIfStmt(s *ast.IfStmt) ast.Visitor {
	if i.evaluate(s.Condition).(ddpbool) {
		i.execute(s.Then)
	} else if s.Else != nil {
		i.execute(s.Else)
	}
	return i
}
func (i *Interpreter) VisitWhileStmt(s *ast.WhileStmt) ast.Visitor {
	for i.evaluate(s.Condition).(ddpbool) {
		i.execute(s.Body)
	}
	return i
}
func (i *Interpreter) VisitForStmt(s *ast.ForStmt) ast.Visitor {
	i.currentEnvironment = newEnvironment(i.currentEnvironment)
	defer i.exitEnvironment() // we have to defer it due to return panic issues

	i.visitNode(s.Initializer)
	for {
		val, _ := i.currentEnvironment.lookupVar(s.Initializer.Name.Literal)
		if val == i.evaluate(s.To) {
			break
		}
		i.execute(s.Body)
		if s.StepSize == nil {
			v, _ := i.currentEnvironment.lookupVar(s.Initializer.Name.Literal)
			val := v.(ddpint)
			val += 1
			i.currentEnvironment.updateVar(s.Initializer.Name.Literal, val)
		} else {
			v, _ := i.currentEnvironment.lookupVar(s.Initializer.Name.Literal)
			val := v.(ddpint)
			val += i.evaluate(s.StepSize).(ddpint)
			i.currentEnvironment.updateVar(s.Initializer.Name.Literal, val)
		}
	}
	i.execute(s.Body)

	return i
}
func (i *Interpreter) VisitFuncCallStmt(s *ast.FuncCallStmt) ast.Visitor {
	return s.Call.Accept(i)
}
func (i *Interpreter) VisitReturnStmt(s *ast.ReturnStmt) ast.Visitor {
	panic(i.evaluate(s.Value)) // we return a value by panicing
}
