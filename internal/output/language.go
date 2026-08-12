// Package output handles terminal rendering with color, emoji, and
// graceful degradation for pipes and dumb terminals.
package output

// Strings holds localized strings for output messages.
type Strings struct {
	Updated         string
	Skipped         string
	Failed          string
	Available       string
	Current         string
	NotInstalled    string
	AllClean        string
	ReviewErrors    string
	AllNotInstalled string
	NothingToDo     string
	DryRunHeader    string
	Updating        string
	Preparing       string
	Initializing    string
	DetectingTools  string
	ConfigGenerated string
	Proceed         string
	Yes             string
	No              string

	// Self-update strings (design D8). The detection-hint string is
	// added together with the check hint (U6), not here.
	SelfUpdatePrompt       string // "Update upp from %s to %s?"
	SelfUpdateTarget       string // "Target: %s"
	SelfUpdateDevBuild     string
	SelfUpdateUpToDate     string // "already up to date (%s)"
	SelfUpdateDeniedCI     string
	SelfUpdateDeniedNotTTY string
	SelfUpdateUnsupported  string
	SelfUpdateDone         string // "upp updated: %s → %s"
	// SelfUpdateHint is the opt-in check hint (design D9): exactly one
	// line after the check summary when check_self_update is enabled and
	// a newer release is known (spec ux-patterns: "⬆️ upp v{latest}
	// available (current {current}) — run "upp self-update""). Unlike
	// the confirm prompt, quiet mode DOES suppress it.
	SelfUpdateHint string
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

		SelfUpdatePrompt:       "Update upp from %s to %s?",
		SelfUpdateTarget:       "Target: %s",
		SelfUpdateDevBuild:     "development build; self-update is only available for release builds",
		SelfUpdateUpToDate:     "already up to date (%s)",
		SelfUpdateDeniedCI:     "self-update denied in --ci mode; run upp self-update interactively to confirm",
		SelfUpdateDeniedNotTTY: "self-update requires an interactive terminal; run upp self-update in a terminal",
		SelfUpdateUnsupported:  "self-update is not supported on this platform yet",
		SelfUpdateDone:         "upp updated: %s → %s",
		SelfUpdateHint:         "⬆️ upp %s available (current %s) — run \"upp self-update\"",
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

		SelfUpdatePrompt:       "¿Actualizar upp de %s a %s?",
		SelfUpdateTarget:       "Destino: %s",
		SelfUpdateDevBuild:     "build de desarrollo; self-update solo está disponible en builds de release",
		SelfUpdateUpToDate:     "ya actualizado (%s)",
		SelfUpdateDeniedCI:     "self-update denegado en modo --ci; ejecuta upp self-update de forma interactiva para confirmar",
		SelfUpdateDeniedNotTTY: "self-update requiere una terminal interactiva; ejecuta upp self-update en una terminal",
		SelfUpdateUnsupported:  "self-update aún no es compatible con esta plataforma",
		SelfUpdateDone:         "upp actualizado: %s → %s",
		SelfUpdateHint:         "⬆️ upp %s disponible (actual %s) — ejecuta \"upp self-update\"",
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
