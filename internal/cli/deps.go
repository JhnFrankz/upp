package cli

// cliDeps is the package-level injection point for the cobra entry points.
// Each New*Command RunE body passes the matching field down to its run*
// function, whose deps struct nil-defaults every nil seam to the production
// implementation — the zero value IS production behavior (design D2, mirrors
// setExecFakes in official/helper.go).
//
// Sequential-only: this package has no t.Parallel tests, so mutation from
// tests (setCLIDeps) needs no synchronization. Adding t.Parallel tests
// requires a mutex or per-command dep construction.
var cliDeps struct {
	dashboard  dashboardDeps
	check      checkDeps
	update     updateDeps
	list       listDeps
	selfUpdate selfUpdateDeps
}
