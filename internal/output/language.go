// Package output handles terminal rendering with color, emoji, and
// graceful degradation for pipes and dumb terminals.
package output

// Strings holds localized strings for output messages.
type Strings struct {
	Updated           string
	Skipped           string
	Failed            string
	Available         string
	Current           string
	NotInstalled      string
	AllClean          string
	ReviewErrors      string
	AllNotInstalled   string
	NothingToDo       string
	DryRunHeader      string
	Updating          string
	Preparing         string
 Initializing      string
	DetectingTools    string
	ConfigGenerated   string
	Proceed           string
	Yes               string
	No                string
}

// DefaultStrings returns English strings.
func DefaultStrings() *Strings {
	return &Strings{
		Updated:         "updated",
		Skipped:         "skipped",
		Failed:          "failed",
		Available:       "available",
		Current:         "current",
		NotInstalled:    "not installed",
		AllClean:        "All clean!",
		ReviewErrors:    "Review errors above",
		AllNotInstalled: "All tools not installed",
		NothingToDo:     "Nothing to do",
		DryRunHeader:    "Dry run — no changes will be made",
		Updating:        "Updating %d/%d: %s",
		Preparing:       "Preparing...",
		Initializing:    "upp init — detecting installed tools...",
		DetectingTools:  "Detecting installed tools...",
		ConfigGenerated: "Config written to %s",
		Proceed:         "Proceed? [y/N] ",
		Yes:             "yes",
		No:              "no",
	}
}

// SpanishStrings returns Spanish strings.
func SpanishStrings() *Strings {
	return &Strings{
		Updated:         "actualizados",
		Skipped:         "omitidos",
		Failed:          "fallidos",
		Available:       "disponibles",
		Current:         "actualizados",
		NotInstalled:    "no instalado",
		AllClean:        "¡Todo limpio!",
		ReviewErrors:    "Revisa los errores arriba",
		AllNotInstalled: "Todas las herramientas no están instaladas",
		NothingToDo:     "Nada que hacer",
		DryRunHeader:    "Simulación — no se realizarán cambios",
		Updating:        "Actualizando %d/%d: %s",
		Preparing:       "Preparando...",
		Initializing:    "upp init — detectando herramientas instaladas...",
		DetectingTools:  "Detectando herramientas instaladas...",
		ConfigGenerated: "Configuración escrita en %s",
		Proceed:         "¿Proceder? [s/N] ",
		Yes:             "sí",
		No:              "no",
	}
}

// GetStrings returns the appropriate string set for the given language.
func GetStrings(lang string) *Strings {
	switch lang {
	case "es":
		return SpanishStrings()
	default:
		return DefaultStrings()
	}
}
