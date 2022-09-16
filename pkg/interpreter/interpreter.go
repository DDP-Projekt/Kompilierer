package interpreter

import (
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/DDP-Projekt/Kompilierer/pkg/ast"
	"github.com/DDP-Projekt/Kompilierer/pkg/ddperror"
	"github.com/DDP-Projekt/Kompilierer/pkg/token"
)

type Interpreter struct {
	ast                *ast.Ast
	currentEnvironment *environment
	lastReturn         value
	errorHandler       ddperror.Handler

	// standard data streams
	Stdout io.Writer
	Stdin  io.Reader
	Stderr io.Writer
}

// returns a new Interpreter ready to interpret it's Ast
func New(Ast *ast.Ast, errorHandler ddperror.Handler) *Interpreter {
	if errorHandler == nil {
		errorHandler = ddperror.EmptyHandler
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
	panic(fmt.Errorf("Laufzeit Fehler in %s in Zeile %d, Spalte %d: %s", tok.File, tok.Line(), tok.Column(), msg))
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
	node.Accept(i)
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
func (i *Interpreter) VisitIndexing(e *ast.Indexing) ast.Visitor {
	/*if v, exists := i.currentEnvironment.lookupVar(e.Name.Literal.Literal); !exists {
		err(e.Token(), fmt.Sprintf("Die Variable '%s' wurde noch nicht deklariert", e.Name.Literal.Literal))
	} else {
		rhs := i.evaluate(e.Index)
		i.lastReturn = ddpchar(([]rune(v.(ddpstring)))[rhs.(ddpint)-1])
	}*/
	err(e.Token(), "indexing not implemented in the interpreter")
	return i
}
func (i *Interpreter) VisitIntLit(e *ast.IntLit) ast.Visitor {
	i.lastReturn = ddpint(e.Value)
	return i
}
func (i *Interpreter) VisitFloatLit(e *ast.FloatLit) ast.Visitor {
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
func (i *Interpreter) VisitListLit(e *ast.ListLit) ast.Visitor {
	panic("lists not implemented")
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
	case token.NEGIERE:
		switch rhs := rhs.(type) {
		case ddpbool:
			i.lastReturn = !rhs
		case ddpint:
			i.lastReturn = ^rhs
		}
	case token.LOGISCHNICHT:
		v := rhs.(ddpint)
		i.lastReturn = ^v
	case token.LÄNGE:
		v := rhs.(ddpstring)
		i.lastReturn = ddpint(len(v))
	case token.GRÖßE:
		switch rhs := rhs.(type) {
		case ddpint, ddpfloat:
			i.lastReturn = ddpint(8)
		case ddpbool:
			i.lastReturn = ddpint(1)
		case ddpchar:
			i.lastReturn = ddpint(4)
		case ddpstring:
			i.lastReturn = ddpint(len(rhs))
		}
	default:
		err(e.Token(), fmt.Sprintf("Unbekannter Operator '%s'", e.Operator))
	}
	return i
}
func (i *Interpreter) VisitBinaryExpr(e *ast.BinaryExpr) ast.Visitor {
	switch e.Operator.Type {
	case token.UND:
		lhs := i.evaluate(e.Lhs).(ddpbool)
		if !lhs {
			i.lastReturn = lhs
			return i
		}
		i.lastReturn = lhs && i.evaluate(e.Rhs).(ddpbool)
		return i
	case token.ODER:
		lhs := i.evaluate(e.Lhs).(ddpbool)
		if lhs {
			i.lastReturn = lhs
			return i
		}
		i.lastReturn = lhs || i.evaluate(e.Rhs).(ddpbool)
		return i
	}

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
	case token.STELLE:
		i.lastReturn = ddpchar(([]rune(lhs.(ddpstring)))[rhs.(ddpint)-1])
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
	case token.LOGARITHMUS:
		switch left := lhs.(type) {
		case ddpint:
			switch right := rhs.(type) {
			case ddpint:
				i.lastReturn = ddpfloat(math.Log10(float64(left)) / math.Log10(float64(right)))
			case ddpfloat:
				i.lastReturn = ddpfloat(math.Log10(float64(left)) / math.Log10(float64(right)))
			}
		case ddpfloat:
			switch right := rhs.(type) {
			case ddpint:
				i.lastReturn = ddpfloat(math.Log10(float64(left)) / math.Log10(float64(right)))
			case ddpfloat:
				i.lastReturn = ddpfloat(math.Log10(float64(left)) / math.Log10(float64(right)))
			}
		}
	case token.LOGISCHUND:
		i.lastReturn = ddpint(lhs.(ddpint) & rhs.(ddpint))
	case token.LOGISCHODER:
		i.lastReturn = ddpint(lhs.(ddpint) | rhs.(ddpint))
	case token.KONTRA:
		i.lastReturn = ddpint(lhs.(ddpint) ^ rhs.(ddpint))
	case token.MODULO:
		i.lastReturn = ddpint(lhs.(ddpint) % rhs.(ddpint))
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
func (i *Interpreter) VisitTernaryExpr(e *ast.TernaryExpr) ast.Visitor {
	lhs := i.evaluate(e.Lhs)
	mid := i.evaluate(e.Mid)
	rhs := i.evaluate(e.Rhs)

	switch e.Operator.Type {
	case token.VONBIS:
		i.lastReturn = ddpstring(([]rune(lhs.(ddpstring)))[mid.(ddpint)-1 : rhs.(ddpint)])
	}

	return i
}
func (i *Interpreter) VisitCastExpr(e *ast.CastExpr) ast.Visitor {
	lhs := i.evaluate(e.Lhs)
	if e.Type.IsList {
		err(e.Token(), "Lists not implemented")
	} else {
		switch e.Type.PrimitiveType {
		case token.ZAHL:
			switch v := lhs.(type) {
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
			switch v := lhs.(type) {
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
			switch v := lhs.(type) {
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
			switch v := lhs.(type) {
			case ddpint:
				i.lastReturn = ddpchar(v)
			case ddpchar:
				i.lastReturn = v
			}
		case token.TEXT:
			switch v := lhs.(type) {
			case ddpint:
				i.lastReturn = ddpstring(strconv.FormatInt(int64(v), 10))
			case ddpfloat:
				i.lastReturn = ddpstring(strconv.FormatFloat(float64(v), 'f', -1, 64))
			case ddpchar:
				i.lastReturn = ddpstring(v)
			case ddpstring:
				i.lastReturn = v
			}
		default:
			err(e.Token(), "Inalide Typumwandlung")
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

		if ast.IsExternFunc(fun) {
			if strings.Trim(fun.ExternFile.Literal, "\"") == "ddpstdlib.lib" {
				i.callInbuilt(fun.Name.Literal)
			} else {
				err(e.Token(), "Externe Funktionen können nicht interpretiert werden")
			}
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
	switch assign := s.Var.(type) {
	case *ast.Ident:
		i.currentEnvironment.updateVar(assign.Literal.Literal, i.evaluate(s.Rhs))
	case *ast.Indexing:
		/*rhs := i.evaluate(s.Rhs)
		index := i.evaluate(assign.Index).(ddpint)
		varRef, _ := i.currentEnvironment.lookupVar(assign.Name.Literal.Literal)
		switch varRef := varRef.(type) {
		case ddpstring:
			runes := []rune(varRef)
			runes[index-1] = rune(rhs.(ddpchar))
			i.currentEnvironment.updateVar(assign.Name.Literal.Literal, ddpstring(runes))
		}*/
		err(assign.Token(), "indexing not implemented in the interpreter")
	}
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
	switch s.While.Type {
	case token.SOLANGE:
		for i.evaluate(s.Condition).(ddpbool) {
			i.execute(s.Body)
		}
	case token.MACHE:
		i.execute(s.Body)
		for i.evaluate(s.Condition).(ddpbool) {
			i.execute(s.Body)
		}
	case token.MAL:
		count := i.evaluate(s.Condition).(ddpint)
		for c := ddpint(0); c < count; c++ {
			i.execute(s.Body)
		}
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
func (i *Interpreter) VisitForRangeStmt(s *ast.ForRangeStmt) ast.Visitor {
	i.currentEnvironment = newEnvironment(i.currentEnvironment)
	defer i.exitEnvironment() // we have to defer it due to return panic issues

	in := i.evaluate(s.In)
	switch in := in.(type) {
	case ddpstring:
		length := len(in)
		if length == 0 {
			return i
		}
		ch, size := utf8.DecodeRuneInString(string(in))
		i.currentEnvironment.addVar(s.Initializer.Name.Literal, ddpchar(ch))
		i.execute(s.Body)
		for j := size; j < length; j += size {
			ch, size = utf8.DecodeRuneInString(string(in)[j:])
			i.currentEnvironment.updateVar(s.Initializer.Name.Literal, ddpchar(ch))
			i.execute(s.Body)
		}
	default:
		panic("runtime error") // unreachable
	}

	return i
}
func (i *Interpreter) VisitFuncCallStmt(s *ast.FuncCallStmt) ast.Visitor {
	return s.Call.Accept(i)
}
func (i *Interpreter) VisitReturnStmt(s *ast.ReturnStmt) ast.Visitor {
	panic(i.evaluate(s.Value)) // we return a value by panicing
}
