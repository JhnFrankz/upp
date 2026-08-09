#!/usr/bin/env bash
# upp — 🚀 Gestor de actualizaciones para tu entorno de desarrollo en Linux
# alias: update-all, upp

set -euo pipefail

# ── Configuración por defecto ────────────────────────
MODE="update"
VERBOSE=false
QUIET=false

# ── Parseo de argumentos ─────────────────────────────
ONLY_TOOLS=()
SKIP_TOOLS=()

while [[ $# -gt 0 ]]; do
  case "$1" in
    --help|-h)
      echo ""
      echo -e "  \033[1;36mupp\033[0m — 🚀 Gestor de actualizaciones de desarrollo"
      echo ""
      echo -e "  \033[1mUso:\033[0m"
      echo "    upp                  Actualizar todo"
      echo "    upp --check          Ver qué tiene actualización disponible"
      echo "    upp --dry-run        Ver qué haría sin ejecutar nada"
      echo "    upp --list           Ver qué herramientas tienes instaladas"
      echo "    upp --only <tool>    Actualizar solo herramientas específicas"
      echo "    upp --skip <tool>    Saltar herramientas específicas"
      echo ""
      echo -e "  \033[1mEjemplos:\033[0m"
      echo "    upp --only brew node     Solo actualizar brew y node"
      echo "    upp --skip apt docker    Actualizar todo excepto apt y docker"
      echo "    upp --check --verbose    Verificar con más detalle"
      echo ""
      echo -e "  \033[1mHerramientas disponibles:\033[0m"
      echo "    apt, brew, node, npm, pnpm, opencode, bun, gh, docker, go"
      echo ""
      exit 0
      ;;
    --check)     MODE="check" ;;
    --dry-run)   MODE="dry-run" ;;
    --list)      MODE="list" ;;
    --verbose|-v) VERBOSE=true ;;
    --quiet|-q)  QUIET=true ;;
    --only)
      shift
      IFS=',' read -ra ONLY_TOOLS <<< "${1:-}"
      ;;
    --skip)
      shift
      IFS=',' read -ra SKIP_TOOLS <<< "${1:-}"
      ;;
    *)
      echo -e "  \033[0;31m❌ Opción desconocida: $1\033[0m"
      echo "  Ejecuta 'upp --help' para ver las opciones disponibles"
      exit 1
      ;;
  esac
  shift
done

# ── Función para verificar si una herramienta debe ejecutarse ──
should_run() {
  local tool="$1"
  if [[ ${#ONLY_TOOLS[@]} -gt 0 ]]; then
    for t in "${ONLY_TOOLS[@]}"; do
      [[ "$t" == "$tool" ]] && return 0
    done
    return 1
  fi
  for t in "${SKIP_TOOLS[@]}"; do
    [[ "$t" == "$tool" ]] && return 1
  done
  return 0
}

# ── Colores ──────────────────────────────────────────
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
CYAN='\033[0;36m'
BOLD='\033[1m'
DIM='\033[2m'
NC='\033[0m'

# ── Contadores ───────────────────────────────────────
UPDATED=0
SKIPPED=0
FAILED=0
UPDATES_AVAIL=0
CURRENT_COUNT=0

# ── Funciones de output ──────────────────────────────
header() {
  echo ""
  echo -e "${CYAN}╔══════════════════════════════════════════════════╗${NC}"
  echo -e "${CYAN}║${NC}  ${BOLD}$1${NC}"
  echo -e "${CYAN}╚══════════════════════════════════════════════════╝${NC}"
}

section() {
  echo ""
  echo -e "  ${BOLD}$1${NC}"
  echo -e "  ${DIM}────────────────────────────────────────────${NC}"
}

ok()      { UPDATED=$((UPDATED + 1)); echo -e "    ${GREEN}✅${NC} $1"; }
skip()    { SKIPPED=$((SKIPPED + 1)); echo -e "    ${YELLOW}⏭️${NC}  $1"; }
fail()    { FAILED=$((FAILED + 1));   echo -e "    ${RED}❌${NC} $1"; }
info()    { echo -e "    ${DIM}ℹ️  $1${NC}"; }
update()  { UPDATES_AVAIL=$((UPDATES_AVAIL + 1)); echo -e "    ${YELLOW}⬆️${NC}  $1"; }
current(){ CURRENT_COUNT=$((CURRENT_COUNT + 1)); echo -e "    ${GREEN}✔️${NC}  $1"; }

run() {
  [[ "$MODE" == "dry-run" ]] && { echo -e "    ${YELLOW}🔍${NC}  [dry-run] $*"; return 0; }
  eval "$@"
}

divider() {
  echo -e "  ${DIM}────────────────────────────────────────────${NC}"
}

# ═══════════════════════════════════════════════════════
#  MODO LIST — Ver qué herramientas están instaladas
# ═══════════════════════════════════════════════════════
if [[ "$MODE" == "list" ]]; then
  header "📋 upp --list"
  echo ""
  echo -e "  ${BOLD}Herramientas instaladas en tu sistema:${NC}"
  echo ""

  INSTALLED=0

  # APT
  section "📦 Sistema (apt)"
  for tool in git curl wget make gcc python3 unzip nano tmux; do
    if command -v "$tool" &>/dev/null; then
      VER=$($tool --version 2>/dev/null | head -1 | awk '{print $3}' || echo "?")
      echo -e "    ${GREEN}✔️${NC}  $tool v$VER"
      INSTALLED=$((INSTALLED + 1))
    fi
  done

  # BREW
  section "🍺 Homebrew"
  if command -v brew &>/dev/null; then
    echo -e "    ${GREEN}✔️${NC}  brew"
    INSTALLED=$((INSTALLED + 1))
    BREW_PKGS=$(brew list 2>/dev/null | tr '\n' ', ' | sed 's/,$//')
    [[ -n "$BREW_PKGS" ]] && echo -e "       ${DIM}Paquetes: $BREW_PKGS${NC}"
  else
    echo -e "    ${YELLOW}⏭️${NC}  brew no instalado"
  fi

  # NODE/NPM/PNPM
  section "🟢 JavaScript"
  export NVM_DIR="$HOME/.nvm"
  if [[ -s "$NVM_DIR/nvm.sh" ]]; then
    SCRIPT_ARGS=("$@")
    set --
    \. "$NVM_DIR/nvm.sh"
    set -- "${SCRIPT_ARGS[@]}"
    NODE_VER=$(nvm current 2>/dev/null | sed 's/v//')
    echo -e "    ${GREEN}✔️${NC}  node v$NODE_VER (nvm)"
    INSTALLED=$((INSTALLED + 1))
  fi
  if command -v npm &>/dev/null; then
    NPM_VER=$(npm --version 2>/dev/null)
    echo -e "    ${GREEN}✔️${NC}  npm v$NPM_VER"
    INSTALLED=$((INSTALLED + 1))
  fi
  if command -v pnpm &>/dev/null; then
    PNPM_VER=$(pnpm --version 2>/dev/null)
    echo -e "    ${GREEN}✔️${NC}  pnpm v$PNPM_VER"
    INSTALLED=$((INSTALLED + 1))
  fi
  if [[ -f "$HOME/.bun/bin/bun" ]]; then
    BUN_VER=$("$HOME/.bun/bin/bun" --version 2>/dev/null)
    echo -e "    ${GREEN}✔️${NC}  bun v$BUN_VER"
    INSTALLED=$((INSTALLED + 1))
  fi

  # DEV TOOLS
  section "🛠️  Herramientas de desarrollo"
  if [[ -f "$HOME/.opencode/bin/opencode" ]]; then
    OC_VER=$("$HOME/.opencode/bin/opencode" --version 2>/dev/null)
    echo -e "    ${GREEN}✔️${NC}  opencode v$OC_VER"
    INSTALLED=$((INSTALLED + 1))
  fi
  if command -v gh &>/dev/null; then
    GH_VER=$(gh --version 2>/dev/null | head -1 | awk '{print $3}')
    echo -e "    ${GREEN}✔️${NC}  gh v$GH_VER"
    INSTALLED=$((INSTALLED + 1))
  fi
  if command -v docker &>/dev/null; then
    DOCKER_VER=$(docker --version 2>/dev/null | awk '{print $3}' | tr -d ',')
    echo -e "    ${GREEN}✔️${NC}  docker v$DOCKER_VER"
    INSTALLED=$((INSTALLED + 1))
  fi
  if command -v go &>/dev/null; then
    GO_VER=$(go version 2>/dev/null | awk '{print $3}' | sed 's/go//')
    echo -e "    ${GREEN}✔️${NC}  go v$GO_VER"
    INSTALLED=$((INSTALLED + 1))
  fi

  echo ""
  divider
  echo ""
  echo -e "  ${BOLD}📊 Total: $INSTALLED herramientas instaladas${NC}"
  echo ""
  exit 0
fi

# ═══════════════════════════════════════════════════════
#  MODO CHECK — Verificar actualizaciones disponibles
# ═══════════════════════════════════════════════════════
if [[ "$MODE" == "check" ]]; then
  header "🔍 upp --check"
  echo ""

  # APT
  section "📦 APT"
  APT_UPGRADABLE=$(apt list --upgradable 2>/dev/null | grep -c upgradable || true)
  if [[ "$APT_UPGRADABLE" -gt 0 ]]; then
    update "apt: $APT_UPGRADABLE paquetes con actualización"
    apt list --upgradable 2>/dev/null | grep upgradable | sed 's/^/       /'
  else
    current "apt: todo al día"
  fi

  # BREW
  section "🍺 Homebrew"
  if command -v brew &>/dev/null; then
    BREW_OUTDATED=$(brew outdated 2>/dev/null | wc -l || true)
    if [[ "$BREW_OUTDATED" -gt 0 ]]; then
      update "brew: $BREW_OUTDATED paquetes con actualización"
      brew outdated 2>/dev/null | sed 's/^/       /'
    else
      current "brew: todo al día (gentle-ai, engram, gga, supabase)"
    fi
  else
    skip "brew no instalado"
  fi

  # NODE.JS
  section "🟢 Node.js"
  export NVM_DIR="$HOME/.nvm"
  if [[ -s "$NVM_DIR/nvm.sh" ]]; then
    SCRIPT_ARGS=("$@")
    set --
    \. "$NVM_DIR/nvm.sh"
    set -- "${SCRIPT_ARGS[@]}"
    CURRENT_NODE=$(nvm current 2>/dev/null | sed 's/v//')
    LATEST_NODE=$(nvm version-remote stable 2>/dev/null | sed 's/v//' || echo "")
    if [[ -n "$LATEST_NODE" && "$CURRENT_NODE" != "$LATEST_NODE" ]]; then
      update "Node.js: v$CURRENT_NODE → v$LATEST_NODE"
    else
      current "Node.js v$CURRENT_NODE: ya en la última versión"
    fi
  else
    skip "nvm no instalado"
  fi

  # NPM GLOBAL
  section "📦 npm global"
  if command -v npm &>/dev/null; then
    NPM_OUTDATED=$(npm outdated -g --parseable 2>/dev/null | wc -l || true)
    if [[ "$NPM_OUTDATED" -gt 0 ]]; then
      update "npm: $NPM_OUTDATED paquetes con actualización"
      npm outdated -g 2>/dev/null | sed 's/^/       /' || true
    else
      current "npm global: todo al día"
    fi
  else
    skip "npm no instalado"
  fi

  # PNPM GLOBAL
  section "📦 pnpm global"
  if command -v pnpm &>/dev/null; then
    PNPM_LIST=$(timeout 30 pnpm outdated -g 2>/dev/null || true)
    PNPM_OUTDATED=$(echo "$PNPM_LIST" | grep "│" | grep -cv "Package\|───\|┌\|└\|├" || true)
    if [[ "$PNPM_OUTDATED" -gt 0 ]]; then
      update "pnpm: $PNPM_OUTDATED paquetes con actualización"
      echo "$PNPM_LIST" | sed 's/^/       /'
    else
      current "pnpm global: todo al día"
    fi
  else
    skip "pnpm no instalado"
  fi

  # OPENCODE
  section "🔧 OpenCode"
  if [[ -f "$HOME/.opencode/bin/opencode" ]]; then
    OC_VER=$("$HOME/.opencode/bin/opencode" --version 2>/dev/null || echo "unknown")
    current "OpenCode v$OC_VER"
    info "Para actualizar: curl -fsSL https://opencode.ai/install | bash"
  else
    skip "OpenCode no instalado"
  fi

  # BUN
  section "🥟 Bun"
  if [[ -f "$HOME/.bun/bin/bun" ]]; then
    BUN_VER=$("$HOME/.bun/bin/bun" --version 2>/dev/null || echo "unknown")
    current "Bun v$BUN_VER"
  else
    skip "Bun no instalado"
  fi

  # GITHUB CLI
  section "🐙 GitHub CLI"
  if command -v gh &>/dev/null; then
    GH_VER=$(gh --version 2>/dev/null | head -1 | awk '{print $3}')
    current "GitHub CLI v$GH_VER"
  else
    skip "GitHub CLI no instalado"
  fi

  # DOCKER
  section "🐳 Docker"
  if command -v docker &>/dev/null; then
    DOCKER_VER=$(docker --version 2>/dev/null | awk '{print $3}' | tr -d ',')
    current "Docker v$DOCKER_VER"
  else
    skip "Docker no instalado"
  fi

  # GO
  section "🐹 Go"
  if command -v go &>/dev/null; then
    GO_VER=$(go version 2>/dev/null | awk '{print $3}' | sed 's/go//')
    LATEST_GO=$(curl -fsSL https://go.dev/dl/?mode=json 2>/dev/null | grep -o '"version": "go[^"]*"' | head -1 | grep -oP 'go\K[0-9.]+' || echo "?")
    if [[ "$GO_VER" != "$LATEST_GO" && "$LATEST_GO" != "?" ]]; then
      update "Go: v$GO_VER → v$LATEST_GO"
    else
      current "Go v$GO_VER: ya en la última versión"
    fi
  else
    skip "Go no instalado"
  fi

  # ── Resumen Check ───────────────────────────────────
  echo ""
  divider
  echo ""
  echo -e "  ${BOLD}📊 Resumen${NC}"
  echo ""
  if [[ $UPDATES_AVAIL -gt 0 ]]; then
    echo -e "    ${YELLOW}⬆️${NC}  ${BOLD}$UPDATES_AVAIL${NC} con actualización disponible"
  fi
  if [[ $CURRENT_COUNT -gt 0 ]]; then
    echo -e "    ${GREEN}✔️${NC}  ${BOLD}$CURRENT_COUNT${NC} ya actualizados"
  fi
  echo ""
  if [[ $UPDATES_AVAIL -eq 0 ]]; then
    echo -e "  ${GREEN}🎉 Todo está al día. ¡Nada que actualizar!${NC}"
  else
    echo -e "  ${YELLOW}💡 Ejecuta ${BOLD}upp${NC}${YELLOW} para actualizar${NC}"
  fi
  echo ""
  exit 0
fi

# ═══════════════════════════════════════════════════════
#  MODO UPDATE / DRY-RUN
# ═══════════════════════════════════════════════════════
ACTUALIZADOS=()
YA_ACTUALIZADOS=()
SALTADOS=()
FALLIDOS=()

if [[ "$MODE" == "dry-run" ]]; then
  header "🔍 upp --dry-run"
else
  header "🚀 upp"
fi

# ── 1. APT ───────────────────────────────────────────
if should_run "apt"; then
  section "📦 APT"
  APT_BEFORE=$(apt list --upgradable 2>/dev/null | grep -c upgradable || true)
  if run "sudo apt update -qq && sudo apt upgrade -y -qq"; then
    if [[ "$APT_BEFORE" -gt 0 ]]; then
      ok "apt: $APT_BEFORE paquetes actualizados"
      ACTUALIZADOS+=("apt ($APT_BEFORE paquetes)")
    else
      current "apt: ya al día"
      YA_ACTUALIZADOS+=("apt")
    fi
  else
    fail "apt: error al actualizar"
    FALLIDOS+=("apt")
  fi
else
  skip "apt: saltado por --skip"
  SALTADOS+=("apt")
fi

# ── 2. HOMEBREW ──────────────────────────────────────
if should_run "brew"; then
  section "🍺 Homebrew"
  if command -v brew &>/dev/null; then
    BREW_BEFORE=$(brew outdated 2>/dev/null | wc -l || true)
    run "brew update" && info "brew update" || fail "brew update"
    run "brew upgrade" && info "brew upgrade" || fail "brew upgrade"
    run "brew cleanup" && info "brew cleanup"
    if [[ "$BREW_BEFORE" -gt 0 ]]; then
      ok "brew: $BREW_BEFORE paquetes actualizados"
      ACTUALIZADOS+=("brew ($BREW_BEFORE paquetes)")
    else
      current "brew: ya al día"
      YA_ACTUALIZADOS+=("brew")
    fi
  else
    skip "brew no instalado"
    SALTADOS+=("brew")
  fi
else
  skip "brew: saltado por --skip"
  SALTADOS+=("brew")
fi

# ── 3. NODE.JS ───────────────────────────────────────
if should_run "node"; then
  section "🟢 Node.js"
  export NVM_DIR="$HOME/.nvm"
  if [[ -s "$NVM_DIR/nvm.sh" ]]; then
    SCRIPT_ARGS=("$@")
    set --
    \. "$NVM_DIR/nvm.sh"
    set -- "${SCRIPT_ARGS[@]}"
    NODE_BEFORE=$(nvm current 2>/dev/null | sed 's/v//')
    LATEST_NODE=$(nvm version-remote stable 2>/dev/null | sed 's/v//' || echo "")
    info "Node actual: v$NODE_BEFORE"
    if [[ -n "$LATEST_NODE" && "$NODE_BEFORE" != "$LATEST_NODE" ]]; then
      # Only use --reinstall-packages-from if current version is valid (not N/A)
      if [[ "$NODE_BEFORE" != "N/A" && -n "$NODE_BEFORE" ]]; then
        run "nvm install stable --reinstall-packages-from=current" && info "nvm install" || fail "nvm install"
      else
        run "nvm install stable" && info "nvm install" || fail "nvm install"
      fi
      # Verify the install actually worked
      NEW_NODE=$(nvm current 2>/dev/null | sed 's/v//')
      if [[ "$NEW_NODE" != "N/A" && "$NEW_NODE" != "$NODE_BEFORE" ]]; then
        ok "Node: v$NODE_BEFORE → v$NEW_NODE"
        ACTUALIZADOS+=("Node v$NODE_BEFORE → v$NEW_NODE")
      else
        fail "Node: instalación no completada correctamente"
        FALLIDOS+=("Node.js")
      fi
    else
      current "Node.js v$NODE_BEFORE: ya al día"
      YA_ACTUALIZADOS+=("Node.js")
    fi
  else
    skip "nvm no instalado"
    SALTADOS+=("node")
  fi
else
  skip "node: saltado por --skip"
  SALTADOS+=("node")
fi

# ── 4. NPM GLOBAL ────────────────────────────────────
if should_run "npm"; then
  section "📦 npm global"
  if command -v npm &>/dev/null; then
    NPM_BEFORE=$(npm outdated -g --parseable 2>/dev/null | wc -l || true)
    run "npm update -g" && info "npm update -g" || fail "npm update -g"
    if [[ "$NPM_BEFORE" -gt 0 ]]; then
      ok "npm: $NPM_BEFORE paquetes actualizados"
      ACTUALIZADOS+=("npm global ($NPM_BEFORE paquetes)")
    else
      current "npm global: ya al día"
      YA_ACTUALIZADOS+=("npm global")
    fi
  else
    skip "npm no instalado"
    SALTADOS+=("npm")
  fi
else
  skip "npm: saltado por --skip"
  SALTADOS+=("npm")
fi

# ── 5. PNPM GLOBAL ───────────────────────────────────
if should_run "pnpm"; then
  section "📦 pnpm global"
  if command -v pnpm &>/dev/null; then
    PNPM_BEFORE=$(pnpm outdated -g 2>/dev/null | grep -c "Latest" || true)
    PNPM_UPDATE_ERR=$(mktemp)
    PNPM_UPDATE_OUT=$(mktemp)
    run "pnpm update -g 2>$PNPM_UPDATE_ERR 1>$PNPM_UPDATE_OUT"
    PNPM_EXIT=$?
    cat "$PNPM_UPDATE_OUT"
    if [[ $PNPM_EXIT -ne 0 ]]; then
      fail "pnpm update -g: error (código $PNPM_EXIT)"
      # Show first 5 lines of error for context
      head -5 "$PNPM_UPDATE_ERR" | sed 's/^/       /'
      FALLIDOS+=("pnpm global")
    elif grep -q "ENOENT\|corrupted\|error" "$PNPM_UPDATE_ERR" 2>/dev/null; then
      fail "pnpm update -g: store corrupto o incompleto"
      head -5 "$PNPM_UPDATE_ERR" | sed 's/^/       /'
      info "Intenta: pnpm store prune && pnpm install -g"
      FALLIDOS+=("pnpm global")
    else
      if [[ "$PNPM_BEFORE" -gt 0 ]]; then
        ok "pnpm: $PNPM_BEFORE paquetes actualizados"
        ACTUALIZADOS+=("pnpm global ($PNPM_BEFORE paquetes)")
      else
        current "pnpm global: ya al día"
        YA_ACTUALIZADOS+=("pnpm global")
      fi
    fi
    rm -f "$PNPM_UPDATE_ERR" "$PNPM_UPDATE_OUT"
  else
    skip "pnpm no instalado"
    SALTADOS+=("pnpm")
  fi
else
  skip "pnpm: saltado por --skip"
  SALTADOS+=("pnpm")
fi

# ── 6. OPENCODE ──────────────────────────────────────
if should_run "opencode"; then
  section "🔧 OpenCode"
  if [[ -f "$HOME/.opencode/bin/opencode" ]]; then
    OC_BEFORE=$("$HOME/.opencode/bin/opencode" --version 2>/dev/null || echo "unknown")
    info "OpenCode actual: v$OC_BEFORE"
    run 'curl -fsSL https://opencode.ai/install | bash' && info "OpenCode reinstall" || fail "OpenCode update"
    OC_AFTER=$("$HOME/.opencode/bin/opencode" --version 2>/dev/null || echo "unknown")
    if [[ "$OC_BEFORE" != "$OC_AFTER" ]]; then
      ok "OpenCode: v$OC_BEFORE → v$OC_AFTER"
      ACTUALIZADOS+=("OpenCode v$OC_BEFORE → v$OC_AFTER")
    else
      current "OpenCode v$OC_AFTER: ya al día"
      YA_ACTUALIZADOS+=("OpenCode")
    fi
  else
    skip "OpenCode no instalado"
    SALTADOS+=("OpenCode")
  fi
else
  skip "opencode: saltado por --skip"
  SALTADOS+=("opencode")
fi

# ── 7. BUN ───────────────────────────────────────────
if should_run "bun"; then
  section "🥟 Bun"
  if [[ -f "$HOME/.bun/bin/bun" ]]; then
    BUN_BEFORE=$("$HOME/.bun/bin/bun" --version 2>/dev/null || echo "unknown")
    info "Bun actual: v$BUN_BEFORE"
    BUP_OUT=$(mktemp)
    BUP_ERR=$(mktemp)
    run "$HOME/.bun/bin/bun upgrade 1>$BUP_OUT 2>$BUP_ERR"
    BUP_EXIT=$?
    cat "$BUP_OUT"
    if [[ $BUP_EXIT -ne 0 ]]; then
      fail "Bun upgrade falló (código $BUP_EXIT)"
      head -3 "$BUP_ERR" | sed 's/^/       /'
      FALLIDOS+=("Bun")
    else
      BUN_AFTER=$("$HOME/.bun/bin/bun" --version 2>/dev/null || echo "unknown")
      if [[ "$BUN_BEFORE" != "$BUN_AFTER" ]]; then
        ok "Bun: v$BUN_BEFORE → v$BUN_AFTER"
        ACTUALIZADOS+=("Bun v$BUN_BEFORE → v$BUN_AFTER")
      else
        current "Bun v$BUN_AFTER: ya al día"
        YA_ACTUALIZADOS+=("Bun")
      fi
    fi
    rm -f "$BUP_OUT" "$BUP_ERR"
  else
    skip "Bun no instalado"
    SALTADOS+=("Bun")
  fi
else
  skip "bun: saltado por --skip"
  SALTADOS+=("bun")
fi

# ── 8. GITHUB CLI ────────────────────────────────────
if should_run "gh"; then
  section "🐙 GitHub CLI"
  if command -v gh &>/dev/null; then
    GH_BEFORE=$(gh --version 2>/dev/null | head -1 | awk '{print $3}')
    run "sudo apt update -qq && sudo apt install gh -y -qq" && info "gh upgrade" || fail "gh upgrade"
    GH_AFTER=$(gh --version 2>/dev/null | head -1 | awk '{print $3}')
    if [[ "$GH_BEFORE" != "$GH_AFTER" ]]; then
      ok "GitHub CLI: v$GH_BEFORE → v$GH_AFTER"
      ACTUALIZADOS+=("GitHub CLI v$GH_BEFORE → v$GH_AFTER")
    else
      current "GitHub CLI v$GH_AFTER: ya al día"
      YA_ACTUALIZADOS+=("GitHub CLI")
    fi
  else
    skip "GitHub CLI no instalado"
    SALTADOS+=("GitHub CLI")
  fi
else
  skip "gh: saltado por --skip"
  SALTADOS+=("gh")
fi

# ── 9. DOCKER ────────────────────────────────────────
if should_run "docker"; then
  section "🐳 Docker"
  if command -v docker &>/dev/null; then
    DOCKER_BEFORE=$(docker --version 2>/dev/null | awk '{print $3}' | tr -d ',')
    info "Docker actual: v$DOCKER_BEFORE"
    run "sudo apt update -qq && sudo apt install -y -qq docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin" && info "docker upgrade" || fail "Docker upgrade"
    DOCKER_AFTER=$(docker --version 2>/dev/null | awk '{print $3}' | tr -d ',')
    if [[ "$DOCKER_BEFORE" != "$DOCKER_AFTER" ]]; then
      ok "Docker: v$DOCKER_BEFORE → v$DOCKER_AFTER"
      ACTUALIZADOS+=("Docker v$DOCKER_BEFORE → v$DOCKER_AFTER")
    else
      current "Docker v$DOCKER_AFTER: ya al día"
      YA_ACTUALIZADOS+=("Docker")
    fi
  else
    skip "Docker no instalado"
    SALTADOS+=("Docker")
  fi
else
  skip "docker: saltado por --skip"
  SALTADOS+=("docker")
fi

# ── 10. GO ───────────────────────────────────────────
if should_run "go"; then
  section "🐹 Go"
  if command -v go &>/dev/null; then
    GO_BEFORE=$(go version 2>/dev/null | awk '{print $3}' | sed 's/go//')
    LATEST_GO=$(curl -fsSL https://go.dev/dl/?mode=json 2>/dev/null | grep -o '"version": "go[^"]*"' | head -1 | grep -oP 'go\K[0-9.]+' || echo "?")
    info "Go actual: v$GO_BEFORE"
    if [[ "$GO_BEFORE" != "$LATEST_GO" && "$LATEST_GO" != "?" ]]; then
      run 'wget -q "https://go.dev/dl/go'"$LATEST_GO"'.linux-amd64.tar.gz" -O /tmp/go-update.tar.gz && sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf /tmp/go-update.tar.gz && rm /tmp/go-update.tar.gz' && info "go reinstall" || fail "Go update"
      GO_AFTER=$(go version 2>/dev/null | awk '{print $3}' | sed 's/go//')
      if [[ "$GO_BEFORE" != "$GO_AFTER" ]]; then
        ok "Go: v$GO_BEFORE → v$GO_AFTER"
        ACTUALIZADOS+=("Go v$GO_BEFORE → v$GO_AFTER")
      else
        current "Go v$GO_AFTER: ya al día"
        YA_ACTUALIZADOS+=("Go")
      fi
    else
      current "Go v$GO_BEFORE: ya al día"
      YA_ACTUALIZADOS+=("Go")
    fi
  else
    skip "Go no instalado"
    SALTADOS+=("Go")
  fi
else
  skip "go: saltado por --skip"
  SALTADOS+=("go")
fi

# ═══════════════════════════════════════════════════════
#  RESUMEN
# ═══════════════════════════════════════════════════════
echo ""
divider
echo ""
echo -e "  ${BOLD}📊 Resumen${NC}"
echo ""

if [[ ${#ACTUALIZADOS[@]} -gt 0 ]]; then
  echo -e "    ${GREEN}✅ Actualizados (${#ACTUALIZADOS[@]}):${NC}"
  for item in "${ACTUALIZADOS[@]}"; do
    echo -e "       ${GREEN}•${NC} $item"
  done
fi

if [[ ${#YA_ACTUALIZADOS[@]} -gt 0 ]]; then
  echo -e "    ${GREEN}✔️  Ya al día (${#YA_ACTUALIZADOS[@]}):${NC}"
  for item in "${YA_ACTUALIZADOS[@]}"; do
    echo -e "       ${GREEN}•${NC} $item"
  done
fi

if [[ ${#SALTADOS[@]} -gt 0 ]]; then
  echo -e "    ${YELLOW}⏭️  No instalados (${#SALTADOS[@]}):${NC}"
  for item in "${SALTADOS[@]}"; do
    echo -e "       ${YELLOW}•${NC} $item"
  done
fi

if [[ ${#FALLIDOS[@]} -gt 0 ]]; then
  echo -e "    ${RED}❌ Fallidos (${#FALLIDOS[@]}):${NC}"
  for item in "${FALLIDOS[@]}"; do
    echo -e "       ${RED}•${NC} $item"
  done
fi

echo ""
if [[ ${#FALLIDOS[@]} -eq 0 ]]; then
  echo -e "  ${GREEN}🎉 ¡Todo limpio! Tu entorno de desarrollo está actualizado.${NC}"
else
  echo -e "  ${RED}⚠️  Revisa los errores arriba y vuelve a intentar.${NC}"
fi
echo ""
