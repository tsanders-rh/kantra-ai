#!/bin/bash
# Helper script for running kantra-ai web planner with Podman

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

print_usage() {
    cat << EOF
Usage: $0 [command] [options]

Commands:
    build                   Build the kantra-ai container image
    run [args]             Run kantra-ai plan with web interface
    shell                  Start interactive shell in container
    help                   Show this help message

Examples:
    # Build the container
    $0 build

    # Run web planner (will prompt for required paths)
    $0 run

    # Run with custom arguments
    $0 run --analysis /path/to/output.yaml --input /path/to/source

    # Open interactive shell
    $0 shell
EOF
}

build_image() {
    echo -e "${BLUE}Building kantra-ai container image...${NC}"
    cd "${PROJECT_ROOT}"

    podman build -t kantra-ai:latest -f Containerfile .

    echo -e "${GREEN}✓ Build complete!${NC}"
    echo -e "Image: ${BLUE}kantra-ai:latest${NC}"
}

run_web_planner() {
    local analysis_path=""
    local input_path=""
    local output_path="${PWD}/.kantra-ai-plan.yaml"

    # Parse arguments or prompt for required values
    while [[ $# -gt 0 ]]; do
        case $1 in
            --analysis)
                analysis_path="$2"
                shift 2
                ;;
            --input)
                input_path="$2"
                shift 2
                ;;
            --output)
                output_path="$2"
                shift 2
                ;;
            *)
                echo -e "${RED}Unknown option: $1${NC}"
                exit 1
                ;;
        esac
    done

    # Prompt for required values if not provided
    if [ -z "$analysis_path" ]; then
        read -p "Path to Konveyor analysis file (output.yaml): " analysis_path
        analysis_path="${analysis_path:-output.yaml}"
    fi

    if [ -z "$input_path" ]; then
        read -p "Path to source code directory: " input_path
    fi

    # Validate paths
    if [ ! -f "$analysis_path" ]; then
        echo -e "${RED}Error: Analysis file not found: $analysis_path${NC}"
        exit 1
    fi

    if [ ! -d "$input_path" ]; then
        echo -e "${RED}Error: Source directory not found: $input_path${NC}"
        exit 1
    fi

    # Convert to absolute paths
    analysis_path="$(cd "$(dirname "$analysis_path")" && pwd)/$(basename "$analysis_path")"
    input_path="$(cd "$input_path" && pwd)"
    output_dir="$(dirname "$output_path")"
    output_file="$(basename "$output_path")"

    echo -e "${BLUE}Starting kantra-ai web planner...${NC}"
    echo -e "  Analysis: ${analysis_path}"
    echo -e "  Source:   ${input_path}"
    echo -e "  Output:   ${output_path}"
    echo ""
    echo -e "${YELLOW}The web interface will be available at http://localhost:8080${NC}"
    echo -e "${YELLOW}Press Ctrl+C to stop the server${NC}"
    echo ""

    # Run container
    podman run -it --rm \
        --name kantra-ai-web \
        -v "${analysis_path}:/data/output.yaml:ro" \
        -v "${input_path}:/source:ro" \
        -v "${output_dir}:/output" \
        -p 8080:8080 \
        kantra-ai:latest \
        plan \
            --analysis /data/output.yaml \
            --input /source \
            --output "/output/${output_file}" \
            --interactive-web
}

run_shell() {
    echo -e "${BLUE}Starting interactive shell in kantra-ai container...${NC}"
    echo -e "${YELLOW}Mounting current directory to /workspace${NC}"
    echo ""

    podman run -it --rm \
        --name kantra-ai-shell \
        -v "${PWD}:/workspace" \
        -p 8080:8080 \
        -w /workspace \
        kantra-ai:latest \
        /bin/sh
}

# Main
case "${1:-help}" in
    build)
        build_image
        ;;
    run)
        shift
        run_web_planner "$@"
        ;;
    shell)
        run_shell
        ;;
    help|--help|-h)
        print_usage
        ;;
    *)
        echo -e "${RED}Unknown command: $1${NC}"
        echo ""
        print_usage
        exit 1
        ;;
esac
