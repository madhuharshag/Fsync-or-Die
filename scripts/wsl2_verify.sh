#!/usr/bin/env bash
# ==============================================================================
# FSYNC-OR-DIE — WSL2 DEMONSTRATION & VERIFICATION RUNNER
# Team Innovexa — "Don't Trust the Process. Trust the Proof."
#
# Primary Demonstration Target: Designated Windows Teammate's Laptop (WSL2 Ubuntu)
# ==============================================================================

set -euo pipefail

BOLD="\033[1m"
GREEN="\033[0;32m"
YELLOW="\033[0;33m"
RED="\033[0;31m"
BLUE="\033[0;34m"
RESET="\033[0m"

echo -e "${BOLD}${BLUE}====================================================================${RESET}"
echo -e "${BOLD}${BLUE}   FSYNC-OR-DIE :: TEAM INNOVEXA VERIFICATION SUITE (WSL2/LINUX)   ${RESET}"
echo -e "${BOLD}${BLUE}   \"Don't Trust the Process. Trust the Proof.\"                     ${RESET}"
echo -e "${BOLD}${BLUE}====================================================================${RESET}"

# 1. Environment & Architecture Check
echo -e "\n${BOLD}[1/5] Checking Environment & Host Architecture...${RESET}"
OS="$(uname -s)"
ARCH="$(uname -m)"
echo "Operating System : ${OS}"
echo "Kernel Release   : $(uname -r)"
echo "CPU Architecture : ${ARCH}"

if [ "${OS}" != "Linux" ]; then
    echo -e "${RED}[ERROR] This script must run inside Linux (e.g. WSL2 Ubuntu). Detected: ${OS}${RESET}"
    exit 1
fi

# Detect whether running inside WSL2
if grep -qi microsoft /proc/version 2>/dev/null; then
    echo -e "${GREEN}[OK] Detected Microsoft WSL2 Kernel environment.${RESET}"
else
    echo -e "${YELLOW}[NOTE] Running on native Linux kernel (non-WSL2). Proceeding.${RESET}"
fi

# Check workspace path: warn if running on 9p mount /mnt/c/
CURRENT_DIR="$(pwd)"
if [[ "${CURRENT_DIR}" == /mnt/* ]]; then
    echo -e "${YELLOW}[WARNING] Workspace is located under Windows mount path (${CURRENT_DIR}).${RESET}"
    echo -e "${YELLOW}          For maximum I/O performance and faithful fsync semantics,${RESET}"
    echo -e "${YELLOW}          run from native Linux filesystem (e.g., /home/username/Team-Innovexa).${RESET}"
else
    echo -e "${GREEN}[OK] Workspace is located on native ext4 filesystem: ${CURRENT_DIR}${RESET}"
fi

# 2. Toolchain Verification
echo -e "\n${BOLD}[2/5] Checking Go Toolchain...${RESET}"
if ! command -v go >/dev/null 2>&1; then
    echo -e "${RED}[ERROR] 'go' command not found in PATH.${RESET}"
    echo "Please ensure Go 1.23+ is installed inside your WSL2 environment."
    echo "Tip: sudo apt update && sudo apt install -y golang-go   OR install from https://go.dev/dl/"
    exit 1
fi

GO_VERSION="$(go version)"
echo -e "${GREEN}[OK] ${GO_VERSION}${RESET}"

# 3. Static Verification (vet & build)
echo -e "\n${BOLD}[3/5] Running Code Quality Checks & Building kip CLI...${RESET}"
go vet ./...
echo -e "${GREEN}[OK] go vet passed cleanly (0 errors, 0 warnings).${RESET}"
go build -o kip ./cmd/kip
echo -e "${GREEN}[OK] Successfully built kip binary.${RESET}"

# 4. Comprehensive Unit & Concurrency Test Suite
echo -e "\n${BOLD}[4/5] Executing Complete Test Suite (Core Engine, CLI, Torture)...${RESET}"
go test -count=1 ./src/kv
echo -e "${GREEN}[OK] Core storage engine tests passed.${RESET}"
go test -count=1 ./cmd/kip/...
echo -e "${GREEN}[OK] CLI tests passed.${RESET}"
go test -count=1 ./torture/...
echo -e "${GREEN}[OK] Torture acceptance tests passed.${RESET}"

# 5. Performance & Recovery Benchmarks
echo -e "\n${BOLD}[5/5] Measuring Empirical Performance & Recovery Speed...${RESET}"
go test -run=^$ -bench=BenchmarkRecovery_1M ./src/kv
rm -f ./kip

echo -e "\n${BOLD}${GREEN}====================================================================${RESET}"
echo -e "${BOLD}${GREEN}   VERIFICATION COMPLETE: ALL INTEGRITY & RECOVERY INVARIANTS PASS   ${RESET}"
echo -e "${BOLD}${GREEN}====================================================================${RESET}"
