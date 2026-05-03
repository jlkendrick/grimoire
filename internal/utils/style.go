package utils

const (
	colorAccent = "\033[38;5;136m" // amber/dark gold — aged bronze feel
	colorSpell  = "\033[38;5;167m" // muted red — runic hint
	colorDim    = "\033[38;5;242m" // stone grey — supporting text
	colorReset  = "\033[0m"
)

func AccentStyle(s string) string { return colorAccent + s + colorReset }
func SpellStyle(s string) string  { return colorSpell + s + colorReset }
func DimStyle(s string) string    { return colorDim + s + colorReset }
