package ast

import "fmt"

// interface for operator enums
// to use them easier in generic functions
type Operator interface {
	String() string
	Operator() // dummy function for the interface
}

type UnaryOperator int

func (UnaryOperator) Operator() {}

const (
	UN_INVALID   UnaryOperator = iota
	UN_ABS                     // Betrag von
	UN_LEN                     // Länge von
	UN_NEGATE                  // -
	UN_NOT                     // nicht
	UN_LOGIC_NOT               // logisch nicht
)

func (op UnaryOperator) String() string {
	switch op {
	case UN_ABS:
		return "Betrag"
	case UN_LEN:
		return "Länge"
	case UN_NEGATE:
		return "-"
	case UN_NOT:
		return "nicht"
	case UN_LOGIC_NOT:
		return "logisch nicht"
	}
	panic(fmt.Errorf("unbekannter unärer Operator %d", op))
}

type BinaryOperator int

func (BinaryOperator) Operator() {}

const (
	BIN_INVALID      BinaryOperator = iota
	BIN_AND                         // und
	BIN_OR                          // oder
	BIN_CONCAT                      // verkettet
	BIN_PLUS                        // plus
	BIN_MINUS                       // minus
	BIN_MULT                        // mal
	BIN_DIV                         // durch
	BIN_INDEX                       // an der Stelle
	BIN_POW                         // hoch
	BIN_LOG                         // Logarithmus
	BIN_LOGIC_AND                   // logisch und
	BIN_LOGIC_OR                    // logisch oder
	BIN_LOGIC_XOR                   // logisch kontra
	BIN_MOD                         // modulo
	BIN_LEFT_SHIFT                  // links verschoben
	BIN_RIGHT_SHIFT                 // rechts verschoben
	BIN_EQUAL                       // gleich
	BIN_UNEQUAL                     // ungleich
	BIN_LESS                        // kleiner
	BIN_GREATER                     // größer
	BIN_LESS_EQ                     // kleiner als, oder
	BIN_GREATER_EQ                  // größer als, oder
	BIN_FIELD_ACCESS                // von
	BIN_SLICE_TO                    // bis zum
	BIN_SLICE_FROM                  // ab dem
)

func (op BinaryOperator) String() string {
	switch op {
	case BIN_AND:
		return "und"
	case BIN_OR:
		return "oder"
	case BIN_CONCAT:
		return "verkettet"
	case BIN_PLUS:
		return "plus"
	case BIN_MINUS:
		return "minus"
	case BIN_MULT:
		return "mal"
	case BIN_DIV:
		return "durch"
	case BIN_INDEX:
		return "an der Stelle"
	case BIN_POW:
		return "hoch"
	case BIN_LOG:
		return "logarithmus"
	case BIN_LOGIC_AND:
		return "logisch und"
	case BIN_LOGIC_OR:
		return "logisch oder"
	case BIN_LOGIC_XOR:
		return "logisch kontra"
	case BIN_MOD:
		return "modulo"
	case BIN_LEFT_SHIFT:
		return "links"
	case BIN_RIGHT_SHIFT:
		return "rechts"
	case BIN_EQUAL:
		return "gleich"
	case BIN_UNEQUAL:
		return "ungleich"
	case BIN_LESS:
		return "kleiner"
	case BIN_GREATER:
		return "größer"
	case BIN_LESS_EQ:
		return "kleiner oder"
	case BIN_GREATER_EQ:
		return "größer oder"
	case BIN_FIELD_ACCESS:
		return "von"
	case BIN_SLICE_TO:
		return "bis zum"
	case BIN_SLICE_FROM:
		return "ab dem"
	}
	panic(fmt.Errorf("unbekannter binärer Operator %d", op))
}

type TernaryOperator int

func (TernaryOperator) Operator() {}

const (
	TER_INVALID TernaryOperator = iota
	TER_SLICE                   // von bis
	TER_BETWEEN                 // zwischen
)

func (op TernaryOperator) String() string {
	switch op {
	case TER_SLICE:
		return "von_bis"
	case TER_BETWEEN:
		return "zwischen"
	}
	panic(fmt.Errorf("unbekannter ternärer Operator %d", op))
}

type TypeOperator int

func (TypeOperator) Operator() {}

const (
	TYPE_INVALID TypeOperator = iota
	TYPE_SIZE                 // Größe
	TYPE_DEFAULT              // Standardwert
)

func (op TypeOperator) String() string {
	switch op {
	case TYPE_SIZE:
		return "Größe"
	case TYPE_DEFAULT:
		return "Standardwert"
	}
	panic(fmt.Errorf("unbekannter Typ-Operator %d", op))
}
