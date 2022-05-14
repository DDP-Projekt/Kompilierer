package token

const (
	ILLEGAL TokenType = iota
	EOF
	IDENTIFIER
	ALIAS_EXPRESSION // [x] only found in function aliases

	literal_begin
	INT    // 1 2
	FLOAT  // 2,2 3,4
	STRING // "hallo" "hi\n"
	CHAR
	TRUE  // wahr
	FALSE // falsch
	literal_end

	operator_begin
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
	VERKETTET // verkettet mit
	operator_end

	keyword_begin
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
	BIS
	MIT
	SCHRITTGRÖßE
	ZAHL
	KOMMAZAHL
	BOOLEAN
	BUCHSTABE
	TEXT
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
	keyword_end

	symbols_begin
	DOT    // .
	COMMA  // ,
	COLON  // :
	LPAREN // (
	RPAREN // )
	symbols_end
)

var tokenStrings = [...]string{
	ILLEGAL:          "ILLEGAL",
	EOF:              "EOF",
	IDENTIFIER:       "IDENTIFIER",
	ALIAS_EXPRESSION: "ALIAS_EXPRESSION",

	//literal_begin
	INT:    "INT LIT",
	FLOAT:  "FLOAT LIT",
	STRING: "STRING LIT",
	CHAR:   "CHAR LIT",
	TRUE:   "TRUE",
	FALSE:  "FALSE",
	//literal_end

	//operator_begin
	PLUS:         "PLUS",
	MINUS:        "MINUS",
	MAL:          "MAL",
	DURCH:        "DURCH",
	MODULO:       "MODULO",
	HOCH:         "HOCH",
	WURZEL:       "WURZEL",
	BETRAG:       "BETRAG",
	SINUS:        "SINUS",
	KOSINUS:      "KOSINUS",
	TANGENS:      "TANGENS",
	UND:          "UND",
	ODER:         "ODER",
	NICHT:        "NICHT",
	GLEICH:       "GLEICH",
	UNGLEICH:     "UNGLEICH",
	KLEINER:      "KLEINER",
	GRÖßER:       "GRÖßER",
	KLEINERODER:  "KLEINER ODER",
	GRÖßERODER:   "GRÖßER ODER",
	NEGATE:       "NEGATE",
	IST:          "IST",
	LINKS:        "LINKS",
	RECHTS:       "RECHTS",
	GRÖßE:        "GRÖßE",
	LÄNGE:        "LÄNGE",
	KONTRA:       "KONTRA",
	LOGISCHUND:   "LOGISCHUND",
	LOGISCHNICHT: "LOGISCHNICHT",
	LOGISCHODER:  "LOGISCHODER",
	VERKETTET:    "VERKETTET",
	//operator_end

	//keyword_begin
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
	BIS:          "BIS",
	MIT:          "MIT",
	SCHRITTGRÖßE: "SCHRITTGRÖßE",
	ZAHL:         "ZAHL",
	KOMMAZAHL:    "KOMMAZAHL",
	BOOLEAN:      "BOOLEAN",
	BUCHSTABE:    "BUCHSTABE",
	TEXT:         "TEXT",
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
	//keyword_end

	//symbols_begin
	DOT:    "DOT",
	COMMA:  "COMMA",
	COLON:  "COLON",
	LPAREN: "LPAREN",
	RPAREN: "RPAREN",
	//symbols_end
}

func (t TokenType) String() string {
	return tokenStrings[t]
}

var keywordMap = map[string]TokenType{
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
	"Sinus":          SINUS,
	"Kosinus":        KOSINUS,
	"Tangens":        TANGENS,
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
	"Solange":        SOLANGE,
	"für":            FÜR,
	"fuer":           FÜR,
	"jede":           JEDE,
	"bis":            BIS,
	"mit":            MIT,
	"Schrittgröße":   SCHRITTGRÖßE,
	"Schrittgroesse": SCHRITTGRÖßE,
	"Zahl":           ZAHL,
	"Kommazahl":      KOMMAZAHL,
	"Boolean":        BOOLEAN,
	"Buchstabe":      BUCHSTABE,
	"Text":           TEXT,
	"Funktion":       FUNKTION,
	"Binde":          BINDE,
	"ein":            EIN,
	"gib":            GIB,
	"zurück":         ZURÜCK,
	"nichts":         NICHTS,
	"um":             UM,
	"Bit":            BIT,
	"nach":           NACH,
	"links":          LINKS,
	"rechts":         RECHTS,
	"verschoben":     VERSCHOBEN,
	"Größe":          GRÖßE,
	"Länge":          LÄNGE,
	"kontra":         KONTRA,
	"logisch":        LOGISCH,
	"mache":          MACHE,
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
}

func KeywordToTokenType(keyword string) TokenType {
	if v, ok := keywordMap[keyword]; ok {
		return v
	}
	return IDENTIFIER
}
