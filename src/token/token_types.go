package token

const (
	ILLEGAL TokenType = iota
	EOF
	IDENTIFIER
	ALIAS_PARAMETER // <x> only found in function aliases
	COMMENT         // [...]
	SYMBOL          // any symbol not matched otherwise (?, !, ~, ...)

	INT    // 1 2
	FLOAT  // 2,2 3,4
	STRING // "hallo" "hi\n"
	CHAR   // 'H' 'ü'
	TRUE   // wahr
	FALSE  // falsch

	PLUS         // plus
	MINUS        // minus
	MAL          // mal
	DURCH        // durch
	MODULO       // modulo
	HOCH         // hoch
	WURZEL       // (n.) Wurzel (von)
	BETRAG       // Betrag (von)
	UND          // und
	ODER         // oder
	ENTWEDER     // entweder
	NICHT        // nicht
	GLEICH       // gleich
	UNGLEICH     // ungleich
	KLEINER      // kleiner (als)
	GRÖßER       // größer (als)(, oder) groesser (als)(, oder)
	ZWISCHEN     // zwischen <a> und <b>
	NEGATE       // -
	IST          // ist
	LINKS        // links
	RECHTS       // rechts
	GRÖßE        // Größe von
	LÄNGE        // Länge von
	KONTRA       // kontra
	VERKETTET    // verkettet mit
	ERHÖHE       // +=
	VERRINGERE   // -=
	VERVIELFACHE // *=
	TEILE        // /=
	VERSCHIEBE   // >>= <<=
	NEGIERE      // x = !x ~=
	LOGARITHMUS
	ZUR
	BASIS
	FALLS     // <a>, falls <b>, ansonsten <c>
	ANSONSTEN // <a>, falls <b>, ansonsten <c>

	DER
	DIE
	VON
	ALS
	WENN
	DANN
	ABER
	SONST
	SOLANGE
	FÜR
	JEDE
	JEDEN
	JEDES
	BIS
	MIT
	SCHRITTGRÖßE
	ZAHL
	ZAHLEN
	KOMMAZAHL
	KOMMAZAHLEN
	WAHRHEITSWERT
	BUCHSTABE
	BUCHSTABEN
	TEXT
	LISTE
	LISTEN
	REFERENZ
	FUNKTION
	BINDE
	EIN
	GIB
	ZURÜCK
	NICHTS
	UM
	BIT
	NACH
	VERSCHOBEN
	LOGISCH
	MACHE
	WIEDERHOLE
	DEM
	PARAMETER
	DEN
	PARAMETERN
	VOM
	TYP
	GIBT
	EINE
	EINEN
	MACHT
	KANN
	SO
	BENUTZT
	WERDEN
	SPEICHERE
	DAS
	ERGEBNIS
	IN
	AN
	STELLE
	VONBIS
	DEFINIERT
	LEERE
	LEEREN
	AUS
	BESTEHT
	EINER
	EINEM
	VERLASSE
	COUNT_MAL
	ALIAS
	STEHT
	OEFFENTLICH
	OEFFENTLICHE
	OEFFENTLICHEN
	WIR
	NENNEN
	DEFINIEREN
	KOMBINATION
	STANDARDWERT
	ERSTELLEN
	SIE
	FAHRE
	SCHLEIFE
	FORT
	IM
	BEREICH
	AB
	ZUM
	ELEMENT
	EXTERN
	SICHTBAR
	SICHTBARE
	AUCH

	DOT    // .
	COMMA  // ,
	COLON  // :
	LPAREN // (
	RPAREN // )
)

var tokenStrings = [...]string{
	ILLEGAL:         "ILLEGAL",
	EOF:             "EOF",
	IDENTIFIER:      "ein Name",
	ALIAS_PARAMETER: "ein Alias Parameter",
	COMMENT:         "ein Kommentar",
	SYMBOL:          "ein Symbol",

	INT:    "eine Zahl",
	FLOAT:  "eine Kommazahl",
	STRING: "ein Text",
	CHAR:   "ein Buchstabe",
	TRUE:   "wahr",
	FALSE:  "falsch",

	PLUS:         "plus",
	MINUS:        "minus",
	MAL:          "mal",
	DURCH:        "durch",
	MODULO:       "modulo",
	HOCH:         "hoch",
	WURZEL:       "Wurzel",
	BETRAG:       "Betrag",
	UND:          "und",
	ODER:         "oder",
	ENTWEDER:     "entweder",
	NICHT:        "nicht",
	GLEICH:       "gleich",
	UNGLEICH:     "ungleich",
	KLEINER:      "kleiner",
	GRÖßER:       "größer",
	ZWISCHEN:     "zwischen",
	NEGATE:       "-",
	IST:          "ist",
	LINKS:        "Links",
	RECHTS:       "Rechts",
	GRÖßE:        "Größe",
	LÄNGE:        "Länge",
	KONTRA:       "kontr",
	VERKETTET:    "verkettet",
	ERHÖHE:       "Erhöhe",
	VERRINGERE:   "Verringere",
	VERVIELFACHE: "Vervielfache",
	TEILE:        "Teile",
	VERSCHIEBE:   "Verschiebe",
	NEGIERE:      "Negiere",
	LOGARITHMUS:  "Logarithmus",
	ZUR:          "zur",
	BASIS:        "Basis",

	DER:           "der",
	DIE:           "die",
	VON:           "von",
	ALS:           "als",
	WENN:          "wenn",
	DANN:          "dann",
	ABER:          "aber",
	SONST:         "sonst",
	SOLANGE:       "solange",
	FÜR:           "für",
	JEDE:          "jede",
	JEDEN:         "jeden",
	JEDES:         "jedes",
	BIS:           "bis",
	MIT:           "mit",
	SCHRITTGRÖßE:  "Schrittgröße",
	ZAHL:          "Zahl",
	ZAHLEN:        "Zahlen",
	KOMMAZAHL:     "Kommazahl",
	KOMMAZAHLEN:   "Kommazahlen",
	WAHRHEITSWERT: "Wahrheitswert",
	BUCHSTABE:     "Buchstabe",
	BUCHSTABEN:    "Buchstaben",
	TEXT:          "Text",
	LISTE:         "Liste",
	LISTEN:        "Listen",
	REFERENZ:      "Referenz",
	FUNKTION:      "Funktion",
	BINDE:         "Binde",
	EIN:           "ein",
	GIB:           "Gib",
	ZURÜCK:        "zurück",
	NICHTS:        "nichts",
	UM:            "um",
	BIT:           "Bit",
	NACH:          "nach",
	VERSCHOBEN:    "verschoben",
	LOGISCH:       "logisch",
	MACHE:         "mache",
	WIEDERHOLE:    "wiederhole",
	DEM:           "dem",
	PARAMETER:     "Parameter",
	DEN:           "den",
	PARAMETERN:    "Parametern",
	VOM:           "vom",
	TYP:           "Typ",
	GIBT:          "gibt",
	EINE:          "eine",
	EINEN:         "einen",
	MACHT:         "macht",
	KANN:          "kann",
	SO:            "so",
	BENUTZT:       "benutzt",
	WERDEN:        "werden",
	SPEICHERE:     "Speichere",
	DAS:           "das",
	ERGEBNIS:      "Ergebnis",
	IN:            "in",
	AN:            "an",
	STELLE:        "Stelle",
	VONBIS:        "von bis", // as operator
	DEFINIERT:     "definiert",
	LEERE:         "leere",
	LEEREN:        "leeren",
	AUS:           "aus",
	BESTEHT:       "besteht",
	EINER:         "einer",
	EINEM:         "einem",
	VERLASSE:      "Verlasse",
	COUNT_MAL:     "Mal",
	ALIAS:         "Alias",
	STEHT:         "Steht",
	OEFFENTLICH:   "öffentlich",
	OEFFENTLICHE:  "öffentliche",
	OEFFENTLICHEN: "öffentlichen",
	WIR:           "Wir",
	NENNEN:        "nennen",
	DEFINIEREN:    "definieren",
	KOMBINATION:   "Kombination",
	STANDARDWERT:  "Standardwert",
	ERSTELLEN:     "erstellen",
	SIE:           "sie",
	FAHRE:         "Fahre",
	SCHLEIFE:      "Schleife",
	FORT:          "fort",
	IM:            "im",
	BEREICH:       "Bereich",
	AB:            "ab",
	ZUM:           "zum",
	ELEMENT:       "Element",
	FALLS:         "falls",
	ANSONSTEN:     "ansonsten",
	EXTERN:        "extern",
	SICHTBAR:      "sichtbar",
	SICHTBARE:     "sichtbare",
	AUCH:          "auch",

	DOT:    ".",
	COMMA:  ",",
	COLON:  ":",
	LPAREN: "(",
	RPAREN: ")",
}

func (t TokenType) String() string {
	return tokenStrings[t]
}

// maps all DDP-keywords to their token-type
// should not be modified!
var KeywordMap = map[string]TokenType{
	"wahr":           TRUE,
	"falsch":         FALSE,
	"plus":           PLUS,
	"minus":          MINUS,
	"mal":            MAL,
	"durch":          DURCH,
	"modulo":         MODULO,
	"hoch":           HOCH,
	"Wurzel":         WURZEL,
	"Betrag":         BETRAG,
	"und":            UND,
	"oder":           ODER,
	"entweder":       ENTWEDER,
	"nicht":          NICHT,
	"gleich":         GLEICH,
	"ungleich":       UNGLEICH,
	"kleiner":        KLEINER,
	"größer":         GRÖßER,
	"groesser":       GRÖßER,
	"zwischen":       ZWISCHEN,
	"ist":            IST,
	"der":            DER,
	"die":            DIE,
	"von":            VON,
	"als":            ALS,
	"wenn":           WENN,
	"dann":           DANN,
	"aber":           ABER,
	"sonst":          SONST,
	"solange":        SOLANGE,
	"für":            FÜR,
	"fuer":           FÜR,
	"jede":           JEDE,
	"jeden":          JEDEN,
	"jedes":          JEDES,
	"bis":            BIS,
	"mit":            MIT,
	"Schrittgröße":   SCHRITTGRÖßE,
	"Schrittgroesse": SCHRITTGRÖßE,
	"Zahl":           ZAHL,
	"Zahlen":         ZAHLEN,
	"Kommazahl":      KOMMAZAHL,
	"Kommazahlen":    KOMMAZAHLEN,
	"Wahrheitswert":  WAHRHEITSWERT,
	"Buchstabe":      BUCHSTABE,
	"Buchstaben":     BUCHSTABEN,
	"Text":           TEXT,
	"Liste":          LISTE,
	"Listen":         LISTEN,
	"Referenz":       REFERENZ,
	"Funktion":       FUNKTION,
	"Binde":          BINDE,
	"ein":            EIN,
	"gib":            GIB,
	"zurück":         ZURÜCK,
	"zurueck":        ZURÜCK,
	"nichts":         NICHTS,
	"um":             UM,
	"Bit":            BIT,
	"nach":           NACH,
	"links":          LINKS,
	"rechts":         RECHTS,
	"verschoben":     VERSCHOBEN,
	"Größe":          GRÖßE,
	"Groesse":        GRÖßE,
	"Länge":          LÄNGE,
	"Laenge":         LÄNGE,
	"kontra":         KONTRA,
	"logisch":        LOGISCH,
	"mache":          MACHE,
	"wiederhole":     WIEDERHOLE,
	"dem":            DEM,
	"Parameter":      PARAMETER,
	"den":            DEN,
	"Parametern":     PARAMETERN,
	"vom":            VOM,
	"Typ":            TYP,
	"gibt":           GIBT,
	"eine":           EINE,
	"einen":          EINEN,
	"macht":          MACHT,
	"kann":           KANN,
	"so":             SO,
	"benutzt":        BENUTZT,
	"werden":         WERDEN,
	"speichere":      SPEICHERE,
	"das":            DAS,
	"Ergebnis":       ERGEBNIS,
	"in":             IN,
	"verkettet":      VERKETTET,
	"erhöhe":         ERHÖHE,
	"erhoehe":        ERHÖHE,
	"verringere":     VERRINGERE,
	"vervielfache":   VERVIELFACHE,
	"teile":          TEILE,
	"verschiebe":     VERSCHIEBE,
	"negiere":        NEGIERE,
	"an":             AN,
	"Stelle":         STELLE,
	"Logarithmus":    LOGARITHMUS,
	"zur":            ZUR,
	"Basis":          BASIS,
	"definiert":      DEFINIERT,
	"leere":          LEERE,
	"leeren":         LEEREN,
	"aus":            AUS,
	"besteht":        BESTEHT,
	"einer":          EINER,
	"einem":          EINEM,
	"verlasse":       VERLASSE,
	"Mal":            COUNT_MAL,
	"Alias":          ALIAS,
	"steht":          STEHT,
	"öffentlich":     OEFFENTLICH,
	"oeffentlich":    OEFFENTLICH,
	"öffentliche":    OEFFENTLICHE,
	"oeffentliche":   OEFFENTLICHE,
	"öffentlichen":   OEFFENTLICHEN,
	"oeffentlichen":  OEFFENTLICHEN,
	"Wir":            WIR,
	"nennen":         NENNEN,
	"definieren":     DEFINIEREN,
	"Kombination":    KOMBINATION,
	"Standardwert":   STANDARDWERT,
	"erstellen":      ERSTELLEN,
	"sie":            SIE,
	"fahre":          FAHRE,
	"Schleife":       SCHLEIFE,
	"fort":           FORT,
	"im":             IM,
	"Bereich":        BEREICH,
	"ab":             AB,
	"zum":            ZUM,
	"Element":        ELEMENT,
	"falls":          FALLS,
	"ansonsten":      ANSONSTEN,
	"extern":         EXTERN,
	"sichtbar":       SICHTBAR,
	"sichtbare":      SICHTBARE,
	"auch":           AUCH,
}

func KeywordToTokenType(keyword string) TokenType {
	if v, ok := KeywordMap[keyword]; ok {
		return v
	}
	return IDENTIFIER
}
