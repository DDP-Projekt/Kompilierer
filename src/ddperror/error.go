package ddperror

import (
	"fmt"

	"github.com/DDP-Projekt/Kompilierer/src/ddptypes"
	"github.com/DDP-Projekt/Kompilierer/src/token"
)

// create a new Error from the given parameters
func NewError(code ErrorCode, Range token.Range, file string, a ...any) Message {
	return Message{
		Code:  int(code),
		Range: Range,
		Msg:   code.ErrorMessage(a...),
		File:  file,
		Level: LEVEL_ERROR,
	}
}

// generates a error message in the format
//
//	Es wurde ..., ... oder ... erwartet aber ... gefunden
func NewUnexpectedTokenError(file string, got *token.Token, expected ...any) Message {
	var msg string
	switch len(expected) {
	case 0:
		panic("called MsgGotExpected without expected arg")
	case 1:
		msg = fmt.Sprintf("Es wurde %v erwartet aber %s gefunden", expected[0], got)
	default:
		msg = fmt.Sprintf("Es wurde %v", expected[0])
		for _, v := range expected[1 : len(expected)-1] {
			msg += fmt.Sprintf(", %v", v)
		}
		msg += fmt.Sprintf(" oder %v erwartet aber %s gefunden", expected[len(expected)-1], got)
	}

	return Message{
		Code:  int(UNEXPECTED_TOKEN),
		Range: got.Range,
		Msg:   msg,
		File:  file,
		Level: LEVEL_ERROR,
	}
}

func NewUnexpectedType(file string, operator string, r token.Range, got ddptypes.Type, expected ...ddptypes.Type) Message {
	msg := fmt.Sprintf("Der %s Operator erwartet einen Ausdruck vom Typ ", operator)
	for i, v := range expected {
		if i >= len(expected)-1 {
			break
		}
		msg += fmt.Sprintf("'%s', ", v)
	}
	msg += fmt.Sprintf("oder '%s' aber hat '%s' bekommen", expected[len(expected)-1], got)

	return Message{
		Code:  int(UNEXPECTED_TOKEN),
		Range: r,
		Msg:   msg,
		File:  file,
	}
}

type ErrorCode int

const (
	EXPECTED_CAPITAL                            ErrorCode = iota // "Nach einem Punkt muss ein Großbuchstabe folgen"
	UNKNOWN_ESCAPE_SEQ                                           // "Unbekannte Escape Sequenz '\\%v'"
	CHAR_TOO_LONG                                                // "Ein Buchstaben Literal darf nur einen Buchstaben enthalten"
	INVALID_FILE_TYPE                                            // "Ungültiger Datei Typ (nicht .ddp)"
	INVALID_PARAMETER_NAME                                       // "Invalider Parameter Name"
	OPEN_PARAMETER                                               // "Offener Parameter"
	EMPTY_PARAMETER                                              // "Ein Parameter in einem Alias muss mindestens einen Buchstaben enthalten"
	ALIAS_EXPECTED_NAME                                          // "Es wurde ein Name als Alias-Parameter erwartet"
	INVALID_UTF8                                                 // "Der Quelltext entspricht nicht dem UTF-8 Standard"
	INCLUDE_MISSING_FILENAME                                     // "Bei Einbindungen muss ein Dateipfad angegeben werden"
	INCLUDE_INVALID_FILE_PATH                                    // "Fehlerhafter Dateipfad '%s': \"%s\""
	INCLUDE_ERROR                                                // "Fehler beim einbinden von '%s': %s"
	INCLUDE_RECURSIVE                                            // "Zwei Module dürfen sich nicht gegenseitig einbinden! Das Modul '%s' versuchte das Modul '%s' einzubinden, während es von diesem Module eingebunden wurde"
	FUNC_NOT_DEFINED                                             // "Die Funktion '%s' wurde nur deklariert aber nie definiert"
	ALIAS_EXISTS_FUNC                                            // "Der Alias %s steht bereits für die Funktion '%s'"
	ALIAS_EXISTS_STRUCT                                          // "Der Alias %s steht bereits für die Struktur '%s'"
	OVERLOAD_ALREADY_DEFINED                                     // "Der Operator '%s' ist für diese Parametertypen bereits überladen"
	UNEXPECTED_TOKEN                                             // "Es wurde %s erwartet aber %v gefunden"
	MAYBE_MISSING_DOT                                            // "Es wurde Mal erwartet aber %v gefunden\nWolltest du vor %s vielleicht einen Punkt setzten?"
	MISSING_DANN                                                 // "In einer Wenn Anweisung, muss ein 'dann' vor dem ':' stehen"
	NO_NEWLINE_AFTER_BLOCK                                       // "Nach einem Doppelpunkt muss eine neue Zeile beginnen"
	EXPECTED_LITERAL                                             // "Es wurde ein Literal erwartet aber ein Ausdruck gefunden"
	WRONG_PRONOUN                                                // "Falsches Pronomen, meintest du %s?"
	WRONG_ARTIKEL                                                // "Falscher Artikel, meintest du %s?"
	EXPECTED_LITERAL_OR_NAME                                     // "Es wurde ein Literal oder ein Name erwartet aber %v gefunden."
	GLOBAL_RETURN_INVALID                                        // "Man kann nur aus Funktionen einen Wert zurückgeben"
	UNEXPECTED_TYPE                                              // "Es wurde %v erwartet aber ein Typname gefunden"
	UNEXPECTED_LIST_TYPE                                         // "Es wurde %v erwartet aber ein Listen-Typname gefunden"
	INVALID_TYPE                                                 // "Invalider Typname %s"
	INVALID_FLOAT_LITERAL                                        // "Das Kommazahlen Literal '%s' kann nicht gelesen werden"
	INVALID_ESCAPE_SEQ_IN_CHAR                                   // "Ungültige Escape Sequenz '\\%s' im Buchstaben Literal"
	INVALID_ESCAPE_SEQ_IN_STR                                    // "Ungültige Escape Sequenz '\\%s' im Text Literal"
	INVALID_INT_LITERAL                                          // "Das Zahlen Literal '%s' kann nicht gelesen werden"
	ALIAS_NAME_EMPTY                                             // "Ein Alias muss mindestens 1 Symbol enthalten"
	ALIAS_MULTIPLE_NEGATION_MARKER                               // "Der Alias enthält mehr als eine Aliasnegationsmarkierung"
	ALIAS_NEGATION_MARKER_IN_NON_BOOL_FUNC                       // "Eine Funktion die kein Wahrheitswert zurück gibt, darf auch keine Negationsmarkierungen haben"
	OPERATOR_UNKNOWN                                             // "'%s' steht nicht für einen Operator"
	OPERATOR_UNARY_WRONG_PARAM_COUNT                             // "Der '%s' Operator erwartet nur einen Parameter, aber hat %d bekommen"
	OPERATOR_BINARY_WRONG_PARAM_COUNT                            // "Der '%s' Operator erwartet zwei Parameter, aber hat %d bekommen"
	OPERATOR_TERNARY_WRONG_PARAM_COUNT                           // "Der '%s' Operator erwartet drei Parameter, aber hat %d bekommen"
	OPERATOR_MUST_RETURN_VALUE                                   // "Ein Operator muss einen Wert zurückgeben"
	NAME_ALREADY_IN_USE                                          // "Der Name %s steht bereits für eine Variable, Konstante, Funktion oder Struktur"
	TYPE_ALREADY_EXISTS                                          // "Ein Typ mit dem Namen '%s' existiert bereits"
	FUNC_MISSING_RETURN                                          // "Am Ende einer Funktion, die etwas zurück gibt, muss eine Rückgabe Anweisung stehen"
	FUNC_MUST_BE_GLOBAL                                          // "Funktionen müssen global definiert werden"
	FUNC_NOT_DECLARED                                            // "Es wurde noch keine Funktion mit dem Namen '%s' deklariert"
	NAME_NOT_DECLARED                                            // "Der Name %s wurde noch nicht deklariert"
	NAME_NOT_A_FUNCTION                                          // "Der Name '%s' steht für eine Variable oder Struktur und nicht für eine Funktion"
	NAME_IS_VARIABLE_NOT_FUNC                                    // "Der Name %s steht für eine Variable und nicht für eine Funktion"
	FUNC_MUST_BE_IN_SAME_MODULE                                  // "Es können nur Funktionen aus demselben Modul definiert werden"
	FUNC_ALREADY_DEFINED                                         // "Die Funktion '%s' wurde bereits definiert"
	TYPE_CANNOT_BE_DEFINED                                       // "Es kann kein neuer Typ als '%s' definiert werden"
	ALIAS_MUST_BE_GLOBAL                                         // "Ein Alias darf nur im globalen Bereich deklariert werden!"
	NAME_ALREADY_USED_IN_MODULE                                  // "Der Name '%s' aus dem Modul '%s' existiert bereits in diesem Modul"
	PUBLIC_CONST_MUST_BE_GLOBAL                                  // "Nur globale Konstante können öffentlich sein"
	PUBLIC_VAR_MUST_BE_GLOBAL                                    // "Nur globale Variablen können öffentlich sein"
	TYPE_MUST_BE_GLOBAL                                          // "Es können nur globale Typen deklariert werden"
	VAR_NOT_DECLARED                                             // "Der Name '%s' wurde noch nicht als Variable deklariert"
	NAME_NOT_A_VAR                                               // "Der Name '%s' steht für eine Funktion oder Struktur und nicht für eine Variable"
	CONST_NOT_ASSIGNABLE                                         // "Der Name '%s' steht für einer Konstante und kann daher nicht zugewiesen werden"
	OPERATOR_VON_EXPECTS_NAME                                    // "Der VON Operator erwartet einen Namen als Linken Operanden, nicht %s"
	NAME_NOT_PUBLIC_IN_MODULE                                    // "Der Name '%s' entspricht keiner öffentlichen Deklaration aus dem Modul '%s'"
	BREAK_OR_CONTINUE_ONLY_IN_LOOP                               // "Break oder Continue darf nur in einer Schleife benutzt werden"
	PUBLIC_CONST_MUST_HAVE_PUBLIC_TYPE                           // "Der Typ einer öffentlichen Konstante muss ebenfalls öffentlich sein"
	PUBLIC_VAR_MUST_HAVE_PUBLIC_TYPE                             // "Der Typ einer öffentlichen Variable muss ebenfalls öffentlich sein"
	PUBLIC_FUNC_RETURN_MUST_BE_PUBLIC                            // "Der Rückgabetyp einer öffentlichen Funktion muss ebenfalls öffentlich sein"
	PUBLIC_FUNC_PARAMS_MUST_BE_PUBLIC                            // "Die Parameter Typen einer öffentlichen Funktion müssen ebenfalls öffentlich sein"
	PUBLIC_STRUCT_FIELDS_MUST_BE_PUBLIC                          // "Wenn eine Struktur öffentlich ist, müssen alle ihre öffentlichen Felder von öffentlichem Typ sein"
	PUBLIC_ALIAS_UNDERLYING_MUST_BE_PUBLIC                       // "Der unterliegende Typ eines öffentlichen Typ-Aliases muss ebenfalls öffentlich sein"
	LOOP_COUNTER_MUST_BE_NUMBER                                  // "Der Zähler in einer zählenden-Schleife muss eine Zahl oder Kommazahl sein"
	EXPECTED_LIST_TYPE_FOR_ITERATOR                              // "Es wurde eine %s erwartet (Listen-Typ des Iterators), aber ein Ausdruck vom Typ %s gefunden"
	EXPECTED_CHAR_TYPE                                           // "Es wurde ein Ausdruck vom Typ Buchstabe erwartet aber %s gefunden"
	CAN_ONLY_ITERATE_OVER_TEXT_OR_LIST                           // "Man kann nur über Texte oder Listen iterieren"
	FUNC_RETURN_TYPE_MISMATCH                                    // "Eine Funktion mit Rückgabetyp %s kann keinen Wert vom Typ %s zurückgeben"
	OPERATOR_EXPECTS_TYPE                                        // "Der %s Operator erwartet einen Ausdruck vom Typ '%s' aber hat '%s' bekommen"
	OPERATOR_EXPECTS_TEXT_OR_LIST                                // "Der %s Operator erwartet einen Text oder eine Liste als Operanden, nicht %s"
	OPERATOR_VERKETTET_TYPE_MISMATCH                             // "Die Typenkombination aus %s und %s passt nicht zum VERKETTET Operator"
	OPERATOR_EXPECTS_TYPE_MISMATCH                               // "Der '%s' Operator erwartet einen Ausdruck vom Typ '%s' aber hat '%s' bekommen"
	OPERATOR_EXPECTS_TWO_SAME_TYPES                              // "Der '%s' Operator erwartet zwei Operanden gleichen Typs aber hat '%s' und '%s' bekommen"
	OPERATOR_FALLS_TYPE_MISMATCH                                 // "Die linke und rechte Seite des 'falls' Ausdrucks müssen den selben Typ haben, aber es wurde %s und %s gefunden"
	EXPRESSION_ALWAYS_TRUE                                       // "Dieser Ausdruck ist immer 'wahr'"
	FUNC_PARAM_TYPE_MISMATCH                                     // "Die Funktion %s erwartet einen Wert vom Typ %s für den Parameter %s, aber hat %s bekommen"
	STRUCT_FIELD_TYPE_MISMATCH                                   // "Die Struktur %s erwartet einen Wert vom Typ %s für das Feld %s, aber hat %s bekommen"
	LOOP_REPEAT_MUST_BE_NUMBER                                   // "Die Anzahl an Wiederholungen einer WIEDERHOLE Anweisung muss vom Typ ZAHL sein, war aber vom Typ %s"
	OPERATOR_FALLS_CONDITION_MUST_BE_BOOL                        // "Die Bedingung des 'falls' Ausdrucks muss vom Typ WAHRHEITSWERT sein, aber es wurde %s gefunden"
	VAR_ASSIGN_TYPE_MISMATCH                                     // "Ein Wert vom Typ %s kann keiner Variable vom Typ %s zugewiesen werden"
	OPERATOR_INDEX_MUST_BE_NUMBER                                // "Der STELLE Operator erwartet eine Zahl als zweiten Operanden, nicht %s"
	OPERATOR_INDEXING_OPERAND_WRONG_TYPE                         // "Der STELLE Operator erwartet einen Text oder eine Liste als ersten Operanden, nicht %s"
	OPERATOR_EXPECTS_TEXT_OR_LIST_FIRST_OPERAND                  // "Der '%s' Operator erwartet einen Text oder eine Liste als ersten Operanden, nicht %s"
	LIST_LITERAL_WRONG_TYPE                                      // "Falscher Typ (%s) in Listen Literal vom Typ %s"
	LIST_SIZE_MUST_BE_NUMBER                                     // "Die Größe einer Liste muss als Zahl angegeben werden, nicht als %s"
	INVALID_CAST                                                 // "Ein Ausdruck vom Typ %s kann nicht in den Typ %s umgewandelt werden"
	EXPECTED_REFERENCE_TYPE                                      // "Es wurde ein Referenz-Typ erwartet aber ein Ausdruck gefunden"
	TEXT_CHAR_CANNOT_BE_REFERENCE                                // "Ein Buchstabe in einem Text kann nicht als Buchstaben Referenz übergeben werden"
	IF_CONDITION_MUST_BE_BOOL                                    // "Die Bedingung einer Wenn-Anweisung muss ein Wahrheitswert sein, war aber vom Typ %s"
	LOOP_CONDITION_MUST_BE_BOOL                                  // "Die Bedingung einer %s muss ein Wahrheitswert sein, war aber vom Typ %s"
	STRUCT_FIELD_NOT_FOUND                                       // "%s %s hat kein Feld mit Name %s"
	OPERATOR_VON_EXPECTS_STRUCT                                  // "Der VON Operator erwartet eine Struktur als rechten Operanden, nicht %s"
	STRUCT_FIELD_NOT_PUBLIC                                      // "Das Feld %s der Struktur %s ist nicht öffentlich"
	ALIAS_ERROR                                                  // "Fehler im Alias '%s': %s"
	EXPECTED_FUNC_NAME                                           // Es wurde ein Funktions Name erwartet
	ARTICLE_MISSING                                              // "Vor '%s' fehlt der Artikel"
	FOR_LOOP_INC_TYPE_MISMATCH                                   // Die Schrittgröße in einer Zählenden-Schleife muss vom selben Typ wie der Zähler (%s) sein, aber war %s
	FOR_LOOP_TYPE_MISMATCH                                       // Der Endwert in einer Zählenden-Schleife muss vom selben Typ wie der Zähler (%s) sein, aber war %s
	FUNC_PARAM_AND_TYPE_COUNT_MISMATCH                           // Die Anzahl von Parametern stimmt nicht mit der Anzahl von Parameter-Typen überein (%d Parameter aber %d Typen)
	FUNC_EXPECTED_LINKABLE_FILEPATH                              // Es wurde ein Pfad zu einer .c, .lib, .a oder .o Datei erwartet aber '%s' gefunden
	FUNC_UNNECESSARY_EXTERN_VISIBLE                              // Es ist unnötig eine externe Funktion auch als extern sichtbar zu deklarieren
	PARAM_ALREADY_EXISTS                                         // Ein Parameter mit dem Namen '%s' ist bereits vorhanden
	EXPECTED_STRUCT_NAME                                         // Es wurde ein Kombinations Name erwartet
	ALIAS_DUPLICATE_PARAMS                                       // Der Alias enthält den Parameter %s mehrmals
	ALIAS_HAS_INVALID_SYMBOLS                                    // Der Alias enthält ungültige Symbole
	ALIAS_PARAM_COUNT_MISMATCH                                   // Der Alias braucht %d Parameter aber hat %d
	FUNC_PARAM_DOESNT_EXIST                                      // Die Funktion hat keinen Parameter mit Namen %s
	ALIAS_TOO_MANY_PARAMS                                        // Der Alias erwartet Maximal %d Parameter aber hat %d
	PARAM_WRONG_GENDER                                           // Es wurde "dem Parameter" oder "den Parametern" erwartet, aber '%s' gefunden
	FUNC_LAST_PARAM_MISSING                                      // der letzte Parameter der Parameterliste ("und <Name>") fehlt.\nMeintest du vorher vielleicht 'dem Parameter' anstatt 'den Parametern'?
	ABS_FILEPATH_NOT_FOUND                                       // Es konnte kein Absoluter Dateipfad für die Datei '%s' gefunden werden: %s
)

func (c ErrorCode) ErrorMessage(a ...any) string {
	return fmt.Sprintf([]string{
		"Nach einem Punkt muss ein Großbuchstabe folgen",
		"Unbekannte Escape Sequenz: '\\%s'",
		"Ein Buchstaben Literal darf nur einen Buchstaben enthalten",
		"Ungültiger Datei Typ (nicht .ddp)",
		"Ungültiger Parameter Name",
		"Offener Parameter",
		"Ein Parameter in einem Alias muss mindestens einen Buchstaben enthalten",
		"Es wurde ein Name als Alias-Parameter erwartet",
		"Der Quelltext entspricht nicht dem UTF-8 Standard",
		"Bei Einbindungen muss ein Dateipfad angegeben werden",
		"Fehlerhafter Dateipfad '%s': \"%s\"",
		"Fehler beim einbinden von '%s': %s",
		"Zwei Module dürfen sich nicht gegenseitig einbinden! Das Modul '%s' versuchte das Modul '%s' einzubinden, während es von diesem Module eingebunden wurde",
		"Die Funktion '%s' wurde nur deklariert aber nie definiert",
		"Der Alias %s steht bereits für die Funktion '%s'",
		"Der Alias %s steht bereits für die Struktur '%s'",
		"Der Operator '%s' ist für diese Parametertypen bereits überladen",
		"Es wurde %s erwartet aber %v gefunden",
		"Es wurde Mal erwartet aber %v gefunden\nWolltest du vor %s vielleicht einen Punkt setzten?",
		"In einer Wenn Anweisung, muss ein 'dann' vor dem ':' stehen",
		"Nach einem Doppelpunkt muss eine neue Zeile beginnen",
		"Es wurde ein Literal erwartet aber ein Ausdruck gefunden",
		"Falsches Pronomen, meintest du %s?",
		"Falscher Artikel, meintest du %s?",
		"Es wurde ein Literal oder ein Name erwartet aber %v gefunden.",
		"Man kann nur aus Funktionen einen Wert zurückgeben",
		"Es wurde %v erwartet aber ein Typname gefunden",
		"Es wurde %v erwartet aber ein Listen-Typname gefunden",
		"Ungültiger Typname %s",
		"Das Kommazahlen Literal '%s' kann nicht gelesen werden",
		"Ungültige Escape Sequenz '\\%s' im Buchstaben Literal",
		"Ungültige Escape Sequenz '\\%s' im Text Literal",
		"Das Zahlen Literal '%s' kann nicht gelesen werden",
		"Ein Alias muss mindestens 1 Symbol enthalten",
		"Der Alias enthält mehr als eine Aliasnegationsmarkierung",
		"Eine Funktion die kein Wahrheitswert zurück gibt, darf auch keine Negationsmarkierungen haben",
		"'%s' steht nicht für einen Operator",
		"Der '%s' Operator erwartet nur einen Parameter, aber hat %d bekommen",
		"Der '%s' Operator erwartet zwei Parameter, aber hat %d bekommen",
		"Der '%s' Operator erwartet drei Parameter, aber hat %d bekommen",
		"Ein Operator muss einen Wert zurückgeben",
		"Der Name %s steht bereits für eine Variable, Konstante, Funktion oder Struktur",
		"Ein Typ mit dem Namen '%s' existiert bereits",
		"Am Ende einer Funktion, die etwas zurück gibt, muss eine Rückgabe Anweisung stehen",
		"Funktionen müssen global definiert werden",
		"Es wurde noch keine Funktion mit dem Namen '%s' deklariert",
		"Der Name %s wurde noch nicht deklariert",
		"Der Name '%s' steht für eine Variable oder Struktur und nicht für eine Funktion",
		"Der Name %s steht für eine Variable und nicht für eine Funktion",
		"Es können nur Funktionen aus demselben Modul definiert werden",
		"Die Funktion '%s' wurde bereits definiert",
		"Es kann kein neuer Typ als '%s' definiert werden",
		"Ein Alias darf nur im globalen Bereich deklariert werden!",
		"Der Name '%s' aus dem Modul '%s' existiert bereits in diesem Modul",
		"Nur globale Konstante können öffentlich sein",
		"Nur globale Variablen können öffentlich sein",
		"Es können nur globale Typen deklariert werden",
		"Der Name '%s' wurde noch nicht als Variable deklariert",
		"Der Name '%s' steht für eine Funktion oder Struktur und nicht für eine Variable",
		"Der Name '%s' steht für einer Konstante und kann daher nicht zugewiesen werden",
		"Der VON Operator erwartet einen Namen als Linken Operanden, nicht %s",
		"Der Name '%s' entspricht keiner öffentlichen Deklaration aus dem Modul '%s'",
		"Break oder Continue darf nur in einer Schleife benutzt werden",
		"Der Typ einer öffentlichen Konstante muss ebenfalls öffentlich sein",
		"Der Typ einer öffentlichen Variable muss ebenfalls öffentlich sein",
		"Der Rückgabetyp einer öffentlichen Funktion muss ebenfalls öffentlich sein",
		"Die Parameter Typen einer öffentlichen Funktion müssen ebenfalls öffentlich sein",
		"Wenn eine Struktur öffentlich ist, müssen alle ihre öffentlichen Felder von öffentlichem Typ sein",
		"Der unterliegende Typ eines öffentlichen Typ-Aliases muss ebenfalls öffentlich sein",
		"Der Zähler in einer zählenden-Schleife muss eine Zahl oder Kommazahl sein",
		"Es wurde eine %s erwartet (Listen-Typ des Iterators), aber ein Ausdruck vom Typ %s gefunden",
		"Es wurde ein Ausdruck vom Typ Buchstabe erwartet aber %s gefunden",
		"Man kann nur über Texte oder Listen iterieren",
		"Eine Funktion mit Rückgabetyp %s kann keinen Wert vom Typ %s zurückgeben",
		"Der %s Operator erwartet einen Ausdruck vom Typ '%s' aber hat '%s' bekommen",
		"Der %s Operator erwartet einen Text oder eine Liste als Operanden, nicht %s",
		"Die Typenkombination aus %s und %s passt nicht zum VERKETTET Operator",
		"Der '%s' Operator erwartet einen Ausdruck vom Typ '%s' aber hat '%s' bekommen",
		"Der '%s' Operator erwartet zwei Operanden gleichen Typs aber hat '%s' und '%s' bekommen",
		"Die linke und rechte Seite des 'falls' Ausdrucks müssen den selben Typ haben, aber es wurde %s und %s gefunden",
		"Dieser Ausdruck ist immer 'wahr'",
		"Die Funktion %s erwartet einen Wert vom Typ %s für den Parameter %s, aber hat %s bekommen",
		"Die Struktur %s erwartet einen Wert vom Typ %s für das Feld %s, aber hat %s bekommen",
		"Die Anzahl an Wiederholungen einer WIEDERHOLE Anweisung muss vom Typ ZAHL sein, war aber vom Typ %s",
		"Die Bedingung des 'falls' Ausdrucks muss vom Typ WAHRHEITSWERT sein, aber es wurde %s gefunden",
		"Ein Wert vom Typ %s kann keiner Variable vom Typ %s zugewiesen werden",
		"Der STELLE Operator erwartet eine Zahl als zweiten Operanden, nicht %s",
		"Der STELLE Operator erwartet einen Text oder eine Liste als ersten Operanden, nicht %s",
		"Der '%s' Operator erwartet einen Text oder eine Liste als ersten Operanden, nicht %s",
		"Falscher Typ (%s) in Listen Literal vom Typ %s",
		"Die Größe einer Liste muss als Zahl angegeben werden, nicht als %s",
		"Ein Ausdruck vom Typ %s kann nicht in den Typ %s umgewandelt werden",
		"Es wurde ein Referenz-Typ erwartet aber ein Ausdruck gefunden",
		"Ein Buchstabe in einem Text kann nicht als Buchstaben Referenz übergeben werden",
		"Die Bedingung einer Wenn-Anweisung muss ein Wahrheitswert sein, war aber vom Typ %s",
		"Die Bedingung einer %s muss ein Wahrheitswert sein, war aber vom Typ %s",
		"%s %s hat kein Feld mit Name %s",
		"Der VON Operator erwartet eine Struktur als rechten Operanden, nicht %s",
		"Das Feld %s der Struktur %s ist nicht öffentlich",
		"Fehler im Alias '%s': %s",
		"Es wurde ein Funktions Name erwartet",
		"Vor '%s' fehlt der Artikel",
		"Die Schrittgröße in einer Zählenden-Schleife muss vom selben Typ wie der Zähler (%s) sein, aber war %s",
		"Der Endwert in einer Zählenden-Schleife muss vom selben Typ wie der Zähler (%s) sein, aber war %s",
		"Die Anzahl von Parametern stimmt nicht mit der Anzahl von Parameter-Typen überein (%d Parameter aber %d Typen)",
		"Es wurde ein Pfad zu einer .c, .lib, .a oder .o Datei erwartet aber '%s' gefunden",
		"Es ist unnötig eine externe Funktion auch als extern sichtbar zu deklarieren",
		"Ein Parameter mit dem Namen '%s' ist bereits vorhanden",
		"Es wurde ein Kombinations Name erwartet",
		"Der Alias enthält den Parameter %s mehrmals",
		"Der Alias enthält ungültige Symbole",
		"Der Alias braucht %d Parameter aber hat %d",
		"Die Funktion hat keinen Parameter mit Namen %s",
		"Der Alias erwartet Maximal %d Parameter aber hat %d",
		"Es wurde \"dem Parameter\" oder \"den Parametern\" erwartet, aber '%s' gefunden",
		"der letzte Parameter der Parameterliste (\"und <Name>\") fehlt.\nMeintest du vorher vielleicht 'dem Parameter' anstatt 'den Parametern'?",
		"Es konnte kein Absoluter Dateipfad für die Datei '%s' gefunden werden: %s",
	}[c], a...)
}

func AliasExistsCode(isFunc bool) ErrorCode {
	code := ALIAS_EXISTS_STRUCT
	if isFunc {
		code = ALIAS_EXISTS_FUNC
	}
	return code
}
