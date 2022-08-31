package ui

import (
	"io"
	"os"
)

// Verbose adds additional info messages e.g. in case of checking errors
var Verbose = false

var Writer io.Writer = os.Stdout

// IconMedal emoji
const IconMedal = "🥇"

// IconRocket emoji
const IconRocket = "🚀"

// IconCross emoji
const IconCross = "❌"

// IconSuggestion emoji
const IconSuggestion = "💡"

// IconDocumentation emoji
const IconDocumentation = "📖"

// IconError emoji
const IconError = "💔"
