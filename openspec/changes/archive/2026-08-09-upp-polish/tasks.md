# Tasks: upp-polish — Post-rewrite polish

## Review Workload Forecast

Decision needed before apply: No
Chained PRs recommended: Yes
Chain strategy: stacked-to-main
400-line budget risk: High

| Field | Value |
|-------|-------|
| Estimated changed lines | ~1,150 total (docs ~250, style ~80, ci/build ~190, tests ~630) |
| 400-line budget risk | High overall; Low/Medium per slice |
| Chained PRs recommended | Yes |
| Suggested split | 5 stacked PRs: docs → style → build+CI → test seam/detect/check → test update/info |
| Delivery strategy | auto-chain |
| Chain strategy | **stacked-to-main** (recommended) — 5 slices independientes, cada una aterriza en main y es revertible. feature-branch-chain solo si se requiere un punto de integración único. |

### Suggested Work Units

| Unit | Goal | PR | Focused test command | Runtime harness | Rollback boundary |
|------|------|----|----------------------|-----------------|-------------------|
| 1 | Docs+registry+config commit | PR 1 | `! grep -q 'Migration from upp.sh' README.md && grep -q sdd-tasks .atl/skill-registry.md` | N/A — no runtime code | revert README.md, .atl/*, config.yaml |
| 2 | gofmt -s en 9 archivos | PR 2 | `test -z "$(gofmt -s -l .)" && go test ./... -count=1` | N/A — solo formato | revert 9 archivos .go |
| 3 | Makefile release/install + ci.yml | PR 3 | `make release && test -f dist/checksums.txt` | `bash scripts/smoke-test.sh --skip-build` | revert Makefile; borrar ci.yml |
| 4 | Seam + tablas detect/check | PR 4 | `go test ./internal/adapters/official/ -count=1 -run 'Detect\|Check'` | N/A — fakes herméticos, sin exec real | revert helper.go; borrar 3 test files |
| 5 | Tablas update/info + refactors | PR 5 | `go test ./internal/adapters/official/ -count=1 -coverprofile=/tmp/c.out && go tool cover -func=/tmp/c.out \| tail -1` (≥80%) | N/A — fakes herméticos | revert 3 test files |

## Phase 1: Docs & Repo Hygiene — PR 1

- [x] 1.1 Commitear `openspec/config.yaml` (Go stack, strict_tdd). Verify: archivo tracked.
- [x] 1.2 Ejecutar `gentle-ai skill-registry refresh`; commitear `.atl/skill-registry.md` + `.atl/.skill-registry-cache.json` (10 skills sdd-*). Verify: `grep sdd-tasks .atl/skill-registry.md`.
- [x] 1.3 Reescribir `README.md` (install Releases/script/source; commands init/update/check/list/export/import + flags; config; security; platforms; development). RED: `grep -q 'Migration from upp.sh' README.md` → GREEN: ausente.

## Phase 2: Style — PR 2

- [x] 2.1 RED: `gofmt -s -l .` lista 9 archivos (cli/parser, config/config, output/language, output/render, platform/catalog, security/confirm, adapter_test.go, adapter_update_test.go, security_expanded_test.go). → PASS: exactamente los 9 esperados.
- [x] 2.2 GREEN: `gofmt -s -w` en esos archivos, commit `style:` dedicado. Verify: `test -z "$(gofmt -s -l .)"` + `go test ./... -count=1` verde. → commit `1bde8b2` en `style/gofmt-upp-polish`.

## Phase 3: Build + CI — PR 3

- [x] 3.1 `Makefile`: `.PHONY release` = build-all + tar.gz/zip por plataforma + `checksums.txt` (sha256sum, fallback shasum), sin tag/push; `install` = go build + `install -m 0755` a PREFIX (default /usr/local/bin). Verify: `make release`; `make install PREFIX=$(mktemp -d)`. → commit `bf72740` "build: add release and install targets to Makefile"
- [x] 3.2 Crear `.github/workflows/ci.yml`: job `test` (ubuntu-latest, Go 1.22, cache, vet, fmt gate `test -z "$(gofmt -s -l .)"`, `go test ./... -count=1`, `-race`, build + smoke); job `lint` BLOQUEANTE (golangci-lint-action); job `release` (dispatch + tag v*, `make release`, upload artifact dist/**, sin publish). Verify: push a branch → verde. → commit `19cb69c` "ci: add test, lint and release workflows"; YAML validado localmente; run real pendiente del push del orquestador.
- [x] 3.3 RED: correr golangci-lint local; si hay findings pre-existentes, fix en commit `style:` dedicado (dentro de este slice). Verify: job lint verde. → 97 findings (errcheck 96 + staticcheck QF1011 1 + unused 1; v2.12.2 full-run); fix commit `eada7d6` "style: fix pre-existing golangci-lint findings (errcheck, staticcheck, unused)"; lint 0 issues (v1.60.3+Go1.22.12 y v2.12.2+Go1.26.5).

## Phase 4: Test seam + detect/check — PR 4 (TDD)

- [x] 4.1 RED: `exec_mock_test.go` — fakeResult, execFakes, `setExecFakes(t, f)` con restore en t.Cleanup; tablas detect (lookPath true/false; nvm NVM_DIR presente/ausente), check (normal/vacío→"unknown"/multilínea/pre-release; apt-cache policy incl. `(none)`; npm/pnpm/bun outdated presente/ausente). Falla: seam vars indefinidas. → commit `f53514a` (seam + detect) y `5f52689` (check tables); RED: build failure "undefined: runCmdFn/runCmdArgsFn/lookPathFn".
- [x] 4.2 GREEN: agregar vars `runCmdFn`/`runCmdArgsFn`/`lookPathFn` en `helper.go` (único cambio prod). Verify: `go test ./internal/adapters/official/ -count=1 -run 'Detect|Check'` verde, hermético. → 30 subtests Detect + 30 subtests Check, 0 ejecuciones reales; suite completa 7/7 pkgs; gofmt/vet limpios; cobertura official 60.7% → 65.5%; race OK.

## Phase 5: Update/info + refactors — PR 5 (TDD)

- [x] 5.1 RED: tablas update (dry-run shortcut, error de comando, stderr E:/ERR!/error, before/after; docker solo en GOOS actual), info (golden ID/name/platforms/trust). Los dry-run tests con exec real fallan (RED). → RED real confirmado: `Update()` llamaba a `runCmd`/`runCmdArgs` DIRECTAMENTE, saltándose el seam (14 call sites en 12 adapters) → `TestUpdate` ejecutaba comandos reales (`exit status 1/127`) y fallaba. GREEN: patrón delegado de slice 4 aplicado a runCmd/runCmdArgs en `helper.go` (único cambio prod).
- [x] 5.2 GREEN: plegar Name/Detect/Info en tablas en `adapter_test.go` (TestAdapterNames golden; Detect→detect_test.go, Info→info_test.go, Check→check_test.go, Update→update_test.go); reemplazar `TestDryRunDoesNotExecute` + dry-runs skip-if-absent en `adapter_update_test.go` por filas mockeadas deterministas con el seam; TestCommandOutput/TestShellOutput ahora herméticos vía seam; versión-helper tables plegadas en supersets *_EdgeCases.
- [x] 5.3 Gate: cobertura `internal/adapters/official` ≥80% → 95.4% (comando de Unit 5); `go test ./... -count=1 -race` verde; `gofmt -s -l .` vacío; `go vet ./...` limpio.

Threat matrix: todas las filas N/A (docs/config/CI son declarativas; los tests jamás ejecutan subprocesos reales) — sin RED tests de seguridad.
