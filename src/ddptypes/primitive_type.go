package ddptypes

// enum type for primitive types (+ void)
type PrimitiveType int

const (
	ZAHL      PrimitiveType = iota // int64
	KOMMAZAHL                      // float64
	BOOLEAN                        // bool
	BUCHSTABE                      // int32
	TEXT                           // string
)

func (p PrimitiveType) Gender() GrammaticalGender {
	switch p {
	case ZAHL, KOMMAZAHL:
		return FEMININ
	case BOOLEAN, BUCHSTABE, TEXT:
		return MASKULIN
	}
	panic("invalid primitive type")
}

func (p PrimitiveType) String() string {
	switch p {
	case ZAHL:
		return "Zahl"
	case KOMMAZAHL:
		return "Kommazahl"
	case BOOLEAN:
		return "Boolean"
	case BUCHSTABE:
		return "Buchstabe"
	case TEXT:
		return "Text"
	}
	panic("invalid primitive type")
}
