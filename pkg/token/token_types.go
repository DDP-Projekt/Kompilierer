package token

const (
	ILLEGAL TokenType = iota
	EOF
	IDENTIFIER
	ALIAS_PARAMETER // <x> only found in function aliases

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

	PLUS     // plus
	MINUS    // minus
	MAL      // mal
	DURCH    // durch
	MODULO   // modulo
	HOCH     // hoch
	WURZEL   // (n.) Wurzel (von)
	BETRAG   // Betrag (von)
	SINUS    // Sinus (von)
	KOSINUS  // Kosinus (von)
	TANGENS  // Tangens (von)
	ARKSIN   // Arkussinus (von)
	ARKKOS   // Arkuskosinus (von)
	ARKTAN   // Arkustangens (von)
	HYPSIN   // Hyperbelsinus (von)
	HYPKOS   // Hyperbelkosinus (von)
	HYPTAN   // Hyperbeltangens (von)
	UND      // und
	ODER     // oder
	NICHT    // nicht
	GLEICH   // gleich
	UNGLEICH // ungleich
	KLEINER  // kleiner (als)
	GRÖßER   // größer (als)(, oder) groesser (als)(, oder)
	KLEINERODER
	GRÖßERODER
	NEGATE // -
	IST    // ist
	LINKS  // links
	RECHTS // rechts
	GRÖßE  // Größe von
	LÄNGE  // Länge von
	KONTRA // kontra
	LOGISCHODER
	LOGISCHUND
	LOGISCHNICHT
	VERKETTET     // verkettet mit
	ADDIERE       // +=
	ERHÖHE        // +=
	SUBTRAHIERE   // -=
	VERRINGERE    // -=
	MULTIPLIZIERE // *=
	VERVIELFACHE  // *=
	DIVIDIERE     // /=
	TEILE         // /=
	VERSCHIEBE    // >>= <<=
	NEGIERE       // x = !x ~=
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

	DOT    // .
	COMMA  // ,
	COLON  // :
	LPAREN // (
	RPAREN // )
)

var tokenStrings = [...]string{
	ILLEGAL:         "ILLEGAL",
	EOF:             "EOF",
	IDENTIFIER:      "IDENTIFIER",
	ALIAS_PARAMETER: "ALIAS_PARAMETER",

	INT:    "INT LIT",
	FLOAT:  "FLOAT LIT",
	STRING: "STRING LIT",
	CHAR:   "CHAR LIT",
	TRUE:   "TRUE",
	FALSE:  "FALSE",

	PI:  "PI",
	E:   "E",
	TAU: "TAU",
	PHI: "PHI",

	PLUS:          "PLUS",
	MINUS:         "MINUS",
	MAL:           "MAL",
	DURCH:         "DURCH",
	MODULO:        "MODULO",
	HOCH:          "HOCH",
	WURZEL:        "WURZEL",
	BETRAG:        "BETRAG",
	SINUS:         "SINUS",
	KOSINUS:       "KOSINUS",
	TANGENS:       "TANGENS",
	ARKSIN:        "ARKUSSINUS",
	ARKKOS:        "ARKUSKOSINUS",
	ARKTAN:        "ARKUSTANGENS",
	HYPSIN:        "HYPERBELSINUS",
	HYPKOS:        "HYPERBELKOSINUS",
	HYPTAN:        "HYPERBELTANGENS",
	UND:           "UND",
	ODER:          "ODER",
	NICHT:         "NICHT",
	GLEICH:        "GLEICH",
	UNGLEICH:      "UNGLEICH",
	KLEINER:       "KLEINER",
	GRÖßER:        "GRÖßER",
	KLEINERODER:   "KLEINER ODER",
	GRÖßERODER:    "GRÖßER ODER",
	NEGATE:        "NEGATE",
	IST:           "IST",
	LINKS:         "LINKS",
	RECHTS:        "RECHTS",
	GRÖßE:         "GRÖßE",
	LÄNGE:         "LÄNGE",
	KONTRA:        "KONTRA",
	LOGISCHUND:    "LOGISCHUND",
	LOGISCHNICHT:  "LOGISCHNICHT",
	LOGISCHODER:   "LOGISCHODER",
	VERKETTET:     "VERKETTET",
	ADDIERE:       "ADDIERE",
	ERHÖHE:        "ERHÖHE",
	SUBTRAHIERE:   "SUBTRAHIERE",
	VERRINGERE:    "VERRINGERE",
	MULTIPLIZIERE: "MULTIPLIZIERE",
	VERVIELFACHE:  "VERVIELFACHE",
	DIVIDIERE:     "DIVIDIERE",
	TEILE:         "TEILE",
	VERSCHIEBE:    "VERSCHIEBE",
	NEGIERE:       "NEGIERE",
	LOGARITHMUS:   "LOGARITHMUS",
	ZUR:           "ZUR",
	BASIS:         "BASIS",

	DER:          "DER",
	DIE:          "DIE",
	VON:          "VON",
	ALS:          "ALS",
	WENN:         "WENN",
	DANN:         "DANN",
	ABER:         "ABER",
	SONST:        "SONST",
	SOLANGE:      "SOLANGE",
	FÜR:          "FÜR",
	JEDE:         "JEDE",
	JEDEN:        "JEDEN",
	BIS:          "BIS",
	MIT:          "MIT",
	SCHRITTGRÖßE: "SCHRITTGRÖßE",
	ZAHL:         "Zahl",
	ZAHLEN:       "ZAHLEN",
	KOMMAZAHL:    "Kommazahl",
	KOMMAZAHLEN:  "KOMMAZAHLEN",
	BOOLEAN:      "Boolean",
	BUCHSTABE:    "Buchstabe",
	BUCHSTABEN:   "BUCHSTABEN",
	TEXT:         "Text",
	LISTE:        "LISTE",
	FUNKTION:     "FUNKTION",
	BINDE:        "BINDE",
	EIN:          "EIN",
	GIB:          "GIB",
	ZURÜCK:       "ZURÜCK",
	NICHTS:       "NICHTS",
	UM:           "UM",
	BIT:          "BIT",
	NACH:         "NACH",
	VERSCHOBEN:   "VERSCHOBEN",
	LOGISCH:      "LOGISCH",
	MACHE:        "MACHE",
	DEM:          "DEM",
	PARAMETER:    "PARAMETER",
	DEN:          "DEN",
	PARAMETERN:   "PARAMETERN",
	VOM:          "VOM",
	TYP:          "TYP",
	GIBT:         "GIBT",
	EINE:         "EINE",
	EINEN:        "EINEN",
	MACHT:        "MACHT",
	KANN:         "KANN",
	SO:           "SO",
	BENUTZT:      "BENUTZT",
	WERDEN:       "WERDEN",
	SPEICHERE:    "SPEICHERE",
	DAS:          "DAS",
	ERGEBNIS:     "ERGEBNIS",
	IN:           "IN",
	AN:           "AN",
	STELLE:       "STELLE",
	VONBIS:       "VON BIS", // as operator
	DEFINIERT:    "DEFINIERT",
	LEERE:        "LEERE",

	DOT:    "DOT",
	COMMA:  "COMMA",
	COLON:  "COLON",
	LPAREN: "LPAREN",
	RPAREN: "RPAREN",
}

func (t TokenType) String() string {
	return tokenStrings[t]
}

var keywordMap = map[string]TokenType{
	"pi":              PI,
	"e":               E,
	"tau":             TAU,
	"phi":             PHI,
	"wahr":            TRUE,
	"falsch":          FALSE,
	"plus":            PLUS,
	"minus":           MINUS,
	"mal":             MAL,
	"durch":           DURCH,
	"modulo":          MODULO,
	"hoch":            HOCH,
	"Wurzel":          WURZEL,
	"Betrag":          BETRAG,
	"Sinus":           SINUS,
	"Kosinus":         KOSINUS,
	"Tangens":         TANGENS,
	"Arkussinus":      ARKSIN,
	"Arkuskosinus":    ARKKOS,
	"Arkustangens":    ARKTAN,
	"Hyperbelsinus":   HYPSIN,
	"Hyperbelkosinus": HYPKOS,
	"Hyperbeltangens": HYPTAN,
	"und":             UND,
	"oder":            ODER,
	"nicht":           NICHT,
	"gleich":          GLEICH,
	"ungleich":        UNGLEICH,
	"kleiner":         KLEINER,
	"größer":          GRÖßER,
	"groesser":        GRÖßER,
	"ist":             IST,
	"der":             DER,
	"die":             DIE,
	"von":             VON,
	"als":             ALS,
	"wenn":            WENN,
	"dann":            DANN,
	"aber":            ABER,
	"sonst":           SONST,
	"solange":         SOLANGE,
	"für":             FÜR,
	"fuer":            FÜR,
	"jede":            JEDE,
	"jeden":           JEDEN,
	"bis":             BIS,
	"mit":             MIT,
	"Schrittgröße":    SCHRITTGRÖßE,
	"Schrittgroesse":  SCHRITTGRÖßE,
	"Zahl":            ZAHL,
	"Zahlen":          ZAHLEN,
	"Kommazahl":       KOMMAZAHL,
	"Kommazahlen":     KOMMAZAHLEN,
	"Boolean":         BOOLEAN,
	"Buchstabe":       BUCHSTABE,
	"Buchstaben":      BUCHSTABEN,
	"Text":            TEXT,
	"Liste":           LISTE,
	"Funktion":        FUNKTION,
	"Binde":           BINDE,
	"ein":             EIN,
	"gib":             GIB,
	"zurück":          ZURÜCK,
	"zurueck":         ZURÜCK,
	"nichts":          NICHTS,
	"um":              UM,
	"Bit":             BIT,
	"nach":            NACH,
	"links":           LINKS,
	"rechts":          RECHTS,
	"verschoben":      VERSCHOBEN,
	"Größe":           GRÖßE,
	"Groesse":         GRÖßE,
	"Länge":           LÄNGE,
	"Laenge":          LÄNGE,
	"kontra":          KONTRA,
	"logisch":         LOGISCH,
	"mache":           MACHE,
	"dem":             DEM,
	"Parameter":       PARAMETER,
	"den":             DEN,
	"Parametern":      PARAMETERN,
	"vom":             VOM,
	"Typ":             TYP,
	"gibt":            GIBT,
	"eine":            EINE,
	"einen":           EINEN,
	"macht":           MACHT,
	"kann":            KANN,
	"so":              SO,
	"benutzt":         BENUTZT,
	"werden":          WERDEN,
	"speichere":       SPEICHERE,
	"das":             DAS,
	"Ergebnis":        ERGEBNIS,
	"in":              IN,
	"verkettet":       VERKETTET,
	"addiere":         ADDIERE,
	"erhöhe":          ERHÖHE,
	"erhoehe":         ERHÖHE,
	"subtrahiere":     SUBTRAHIERE,
	"verringere":      VERRINGERE,
	"multipliziere":   MULTIPLIZIERE,
	"vervielfache":    VERVIELFACHE,
	"dividiere":       DIVIDIERE,
	"teile":           TEILE,
	"verschiebe":      VERSCHIEBE,
	"negiere":         NEGIERE,
	"an":              AN,
	"Stelle":          STELLE,
	"Logarithmus":     LOGARITHMUS,
	"zur":             ZUR,
	"Basis":           BASIS,
	"definiert":       DEFINIERT,
	"leere":           LEERE,
}

func KeywordToTokenType(keyword string) TokenType {
	if v, ok := keywordMap[keyword]; ok {
		return v
	}
	return IDENTIFIER
}
