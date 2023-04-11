package token

const (
	ILLEGAL TokenType = iota
	EOF
	IDENTIFIER
	ALIAS_PARAMETER // <x> only found in function aliases
	COMMENT         // [...]

	INT    // 1 2
	FLOAT  // 2,2 3,4
	STRING // "hallo" "hi\n"
	CHAR   // 'H' 'ü'
	TRUE   // wahr
	FALSE  // falsch

	PI  // pi
	E   // e
	TAU // tau
	PHI // phi

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
	NICHT        // nicht
	GLEICH       // gleich
	UNGLEICH     // ungleich
	KLEINER      // kleiner (als)
	GRÖßER       // größer (als)(, oder) groesser (als)(, oder)
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
	BIS
	MIT
	SCHRITTGRÖßE
	ZAHL
	ZAHLEN
	KOMMAZAHL
	KOMMAZAHLEN
	BOOLEAN
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
	VERLASSE
	COUNT_MAL
	ALIAS
	STEHT
	OEFFENTLICHE

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

	INT:    "eine Zahl",
	FLOAT:  "eine Kommazahl",
	STRING: "ein Text",
	CHAR:   "ein Buchstabe",
	TRUE:   "wahr",
	FALSE:  "falsch",

	PI:  "pi",
	E:   "e",
	TAU: "tau",
	PHI: "phi",

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
	NICHT:        "nicht",
	GLEICH:       "gleich",
	UNGLEICH:     "ungleich",
	KLEINER:      "kleiner",
	GRÖßER:       "größer",
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

	DER:          "der",
	DIE:          "die",
	VON:          "von",
	ALS:          "als",
	WENN:         "wenn",
	DANN:         "dann",
	ABER:         "aber",
	SONST:        "sonst",
	SOLANGE:      "solange",
	FÜR:          "für",
	JEDE:         "jede",
	JEDEN:        "jeden",
	BIS:          "bis",
	MIT:          "mit",
	SCHRITTGRÖßE: "Schrittgröße",
	ZAHL:         "Zahl",
	ZAHLEN:       "Zahlen",
	KOMMAZAHL:    "Kommazahl",
	KOMMAZAHLEN:  "Kommazahlen",
	BOOLEAN:      "Boolean",
	BUCHSTABE:    "Buchstabe",
	BUCHSTABEN:   "Buchstaben",
	TEXT:         "Text",
	LISTE:        "Liste",
	LISTEN:       "Listen",
	REFERENZ:     "Referenz",
	FUNKTION:     "Funktion",
	BINDE:        "Binde",
	EIN:          "ein",
	GIB:          "Gib",
	ZURÜCK:       "zurück",
	NICHTS:       "nichts",
	UM:           "um",
	BIT:          "Bit",
	NACH:         "nach",
	VERSCHOBEN:   "verschoben",
	LOGISCH:      "logisch",
	MACHE:        "mache",
	WIEDERHOLE:   "wiederhole",
	DEM:          "dem",
	PARAMETER:    "Parameter",
	DEN:          "den",
	PARAMETERN:   "Parametern",
	VOM:          "vom",
	TYP:          "Typ",
	GIBT:         "gibt",
	EINE:         "eine",
	EINEN:        "einen",
	MACHT:        "macht",
	KANN:         "kann",
	SO:           "so",
	BENUTZT:      "benutzt",
	WERDEN:       "werden",
	SPEICHERE:    "Speichere",
	DAS:          "das",
	ERGEBNIS:     "Ergebnis",
	IN:           "in",
	AN:           "an",
	STELLE:       "Stelle",
	VONBIS:       "von bis", // as operator
	DEFINIERT:    "definiert",
	LEERE:        "leere",
	LEEREN:       "leeren",
	AUS:          "aus",
	BESTEHT:      "besteht",
	EINER:        "einer",
	VERLASSE:     "Verlasse",
	COUNT_MAL:    "Mal",
	ALIAS:        "Alias",
	STEHT:        "Steht",
	OEFFENTLICHE: "öffentliche",

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
	"pi":             PI,
	"e":              E,
	"tau":            TAU,
	"phi":            PHI,
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
	"nicht":          NICHT,
	"gleich":         GLEICH,
	"ungleich":       UNGLEICH,
	"kleiner":        KLEINER,
	"größer":         GRÖßER,
	"groesser":       GRÖßER,
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
	"bis":            BIS,
	"mit":            MIT,
	"Schrittgröße":   SCHRITTGRÖßE,
	"Schrittgroesse": SCHRITTGRÖßE,
	"Zahl":           ZAHL,
	"Zahlen":         ZAHLEN,
	"Kommazahl":      KOMMAZAHL,
	"Kommazahlen":    KOMMAZAHLEN,
	"Boolean":        BOOLEAN,
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
	"verlasse":       VERLASSE,
	"Mal":            COUNT_MAL,
	"Alias":          ALIAS,
	"steht":          STEHT,
	"öffentliche":    OEFFENTLICHE,
	"oeffentliche":   OEFFENTLICHE,
}

func KeywordToTokenType(keyword string) TokenType {
	if v, ok := KeywordMap[keyword]; ok {
		return v
	}
	return IDENTIFIER
}
