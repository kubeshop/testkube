package expressions

type tokenType uint8

const (
	// Primitives
	tokenTypeAccessor tokenType = iota
	tokenTypePropertyAccessor
	tokenTypeJson

	// Math
	tokenTypeNot
	tokenTypeMath
	tokenTypeOpen
	tokenTypeClose

	// Logical
	tokenTypeTernary
	tokenTypeTernarySeparator

	// Functions
	tokenTypeComma
	tokenTypeSpread
)

type token struct {
	Type  tokenType
	Value interface{}
}

var (
	tokenNot              = token{Type: tokenTypeNot}
	tokenOpen             = token{Type: tokenTypeOpen}
	tokenClose            = token{Type: tokenTypeClose}
	tokenTernary          = token{Type: tokenTypeTernary}
	tokenTernarySeparator = token{Type: tokenTypeTernarySeparator}
	tokenComma            = token{Type: tokenTypeComma}
	tokenSpread           = token{Type: tokenTypeSpread}
)

func tokenMath(op string) token {
	return token{Type: tokenTypeMath, Value: op}
}

func tokenJson(value interface{}) token {
	return token{Type: tokenTypeJson, Value: value}
}

func tokenAccessor(value interface{}) token {
	return token{Type: tokenTypeAccessor, Value: value}
}

func tokenPropertyAccessor(value interface{}) token {
	return token{Type: tokenTypePropertyAccessor, Value: value}
}
