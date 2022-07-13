package interpreter

import (
	"fmt"
	"time"
)

type inbuiltfunction func(*Interpreter) value

func (i *Interpreter) callInbuilt(name string) value {
	return inbuiltFunctions[name](i)
}

var inbuiltFunctions = map[string]inbuiltfunction{
	"§Schreibe_Zahl":           schreibeZahl,
	"§Schreibe_Kommazahl":      schreibeKommazahl,
	"§Schreibe_Boolean":        schreibeBoolean,
	"§Schreibe_Buchstabe":      schreibeBuchstabe,
	"§Schreibe_Text":           schreibeText,
	"§Zeit_Seit_Programmstart": zeitSeitProgrammstart,
}

func schreibeZahl(i *Interpreter) value {
	val, _ := i.currentEnvironment.lookupVar("p1")
	fmt.Fprint(i.Stdout, val.(ddpint))
	return nil
}

func schreibeKommazahl(i *Interpreter) value {
	val, _ := i.currentEnvironment.lookupVar("p1")
	fmt.Fprint(i.Stdout, val.(ddpfloat))
	return nil
}

func schreibeBoolean(i *Interpreter) value {
	val, _ := i.currentEnvironment.lookupVar("p1")
	if val.(ddpbool) {
		fmt.Fprint(i.Stdout, "wahr")
	} else {
		fmt.Fprint(i.Stdout, "falsch")
	}
	return nil
}

func schreibeBuchstabe(i *Interpreter) value {
	val, _ := i.currentEnvironment.lookupVar("p1")
	fmt.Fprint(i.Stdout, string(val.(ddpchar)))
	return nil
}

func schreibeText(i *Interpreter) value {
	val, _ := i.currentEnvironment.lookupVar("p1")
	fmt.Fprint(i.Stdout, val.(ddpstring))
	return nil
}

var startTime = time.Now()

func zeitSeitProgrammstart(i *Interpreter) value {
	return ddpint(time.Since(startTime).Milliseconds())
}
