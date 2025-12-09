cmake_minimum_required(VERSION 3.16)
project(voting VERSION 1.0.0 LANGUAGES C)  # C needed for GNUInstallDirs

# Find Go compiler
find_program(GO_EXECUTABLE go REQUIRED)

# Get Go version
execute_process(
    COMMAND ${GO_EXECUTABLE} version
    OUTPUT_VARIABLE GO_VERSION_OUTPUT
    OUTPUT_STRIP_TRAILING_WHITESPACE
)
message(STATUS "Found Go: ${GO_VERSION_OUTPUT}")

# Project directories
set(CMD_DIR ${CMAKE_SOURCE_DIR}/cmd/derbyvote)
set(INTERNAL_DIR ${CMAKE_SOURCE_DIR}/internal)
set(TEMPLATES_DIR ${CMAKE_SOURCE_DIR}/web/templates)

# Output binary name
set(BINARY_NAME derbyvote)
if(WIN32)
    set(BINARY_NAME derbyvote.exe)
endif()

# Build output directory (use source dir to avoid Go module conflicts)
set(BUILD_OUTPUT_DIR ${CMAKE_SOURCE_DIR}/bin)
file(MAKE_DIRECTORY ${BUILD_OUTPUT_DIR})

# Build flags
set(GO_BUILD_FLAGS "" CACHE STRING "Additional Go build flags")
set(GO_LDFLAGS "" CACHE STRING "Go linker flags")

# Build type specific flags
if(CMAKE_BUILD_TYPE STREQUAL "Release")
    set(GO_LDFLAGS "-s -w ${GO_LDFLAGS}")
    message(STATUS "Release build: stripping debug symbols")
endif()

# CGO is required for SQLite
set(CGO_ENABLED "1")

# Find UPX for compression
find_program(UPX_EXECUTABLE upx)
if(UPX_EXECUTABLE)
    message(STATUS "Found UPX: ${UPX_EXECUTABLE}")
else()
    message(STATUS "UPX not found - compression will be skipped")
endif()

# Main build target
add_custom_target(derbyvote ALL
    COMMAND ${CMAKE_COMMAND} -E env CGO_ENABLED=${CGO_ENABLED}
        ${GO_EXECUTABLE} build
        ${GO_BUILD_FLAGS}
        -o ${BUILD_OUTPUT_DIR}/${BINARY_NAME}
        $<$<NOT:$<STREQUAL:${GO_LDFLAGS},>>:-ldflags=${GO_LDFLAGS}>
        ./cmd/derbyvote/...
    WORKING_DIRECTORY ${CMAKE_SOURCE_DIR}
    COMMENT "Building derbyvote (CGO_ENABLED=${CGO_ENABLED})..."
    SOURCES
        ${CMD_DIR}/main.go
        ${INTERNAL_DIR}/handlers/admin.go
        ${INTERNAL_DIR}/handlers/vote.go
        ${INTERNAL_DIR}/models/models.go
        ${INTERNAL_DIR}/websocket/websocket.go
)

# UPX compression target
if(UPX_EXECUTABLE)
    add_custom_target(upx
        COMMAND ${UPX_EXECUTABLE} --best --lzma ${BUILD_OUTPUT_DIR}/${BINARY_NAME}
        WORKING_DIRECTORY ${CMAKE_SOURCE_DIR}
        DEPENDS derbyvote
        COMMENT "Compressing binary with UPX..."
    )
endif()

# Download dependencies
add_custom_target(deps
    COMMAND ${GO_EXECUTABLE} mod download
    COMMAND ${GO_EXECUTABLE} mod tidy
    WORKING_DIRECTORY ${CMAKE_SOURCE_DIR}
    COMMENT "Downloading Go dependencies..."
)

# Run target
add_custom_target(run
    COMMAND ${BUILD_OUTPUT_DIR}/${BINARY_NAME}
    WORKING_DIRECTORY ${CMAKE_SOURCE_DIR}
    DEPENDS derbyvote
    COMMENT "Running derbyvote..."
)

# Run with go run (faster for development)
add_custom_target(run-dev
    COMMAND ${GO_EXECUTABLE} run ./cmd/derbyvote/...
    WORKING_DIRECTORY ${CMAKE_SOURCE_DIR}
    COMMENT "Running derbyvote (development mode)..."
)

# Test target
add_custom_target(test
    COMMAND ${GO_EXECUTABLE} test -v ./...
    WORKING_DIRECTORY ${CMAKE_SOURCE_DIR}
    COMMENT "Running tests..."
)

# Test with coverage
add_custom_target(test-coverage
    COMMAND ${GO_EXECUTABLE} test -v -coverprofile=${CMAKE_BINARY_DIR}/coverage.out ./...
    COMMAND ${GO_EXECUTABLE} tool cover -html=${CMAKE_BINARY_DIR}/coverage.out -o ${CMAKE_BINARY_DIR}/coverage.html
    WORKING_DIRECTORY ${CMAKE_SOURCE_DIR}
    COMMENT "Running tests with coverage..."
)

# Lint target (requires golangci-lint)
find_program(GOLANGCI_LINT golangci-lint)
if(GOLANGCI_LINT)
    add_custom_target(lint
        COMMAND ${GOLANGCI_LINT} run ./...
        WORKING_DIRECTORY ${CMAKE_SOURCE_DIR}
        COMMENT "Running linter..."
    )
else()
    add_custom_target(lint
        COMMAND ${CMAKE_COMMAND} -E echo "golangci-lint not found. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"
        COMMENT "Linter not available"
    )
endif()

# Format target
add_custom_target(fmt
    COMMAND ${GO_EXECUTABLE} fmt ./...
    WORKING_DIRECTORY ${CMAKE_SOURCE_DIR}
    COMMENT "Formatting Go code..."
)

# Vet target
add_custom_target(vet
    COMMAND ${GO_EXECUTABLE} vet ./...
    WORKING_DIRECTORY ${CMAKE_SOURCE_DIR}
    COMMENT "Running go vet..."
)

# Clean target (CMake provides this automatically, but we add Go-specific cleaning)
add_custom_target(clean-go
    COMMAND ${GO_EXECUTABLE} clean -cache
    COMMAND ${CMAKE_COMMAND} -E remove_directory ${BUILD_OUTPUT_DIR}
    WORKING_DIRECTORY ${CMAKE_SOURCE_DIR}
    COMMENT "Cleaning Go build cache..."
)

# Install target
include(GNUInstallDirs)
install(PROGRAMS ${BUILD_OUTPUT_DIR}/${BINARY_NAME}
    DESTINATION ${CMAKE_INSTALL_BINDIR}
    COMPONENT runtime
)
install(DIRECTORY ${TEMPLATES_DIR}
    DESTINATION ${CMAKE_INSTALL_DATADIR}/${PROJECT_NAME}
    COMPONENT runtime
)

# Cross-compilation targets
set(CROSS_COMPILE_TARGETS
    "linux-amd64:linux:amd64::gcc"
    "linux-arm64:linux:arm64::aarch64-linux-gnu-gcc"
    "linux-armv7:linux:arm:7:arm-linux-gnueabihf-gcc"
    "linux-armv6:linux:arm:6:arm-linux-gnueabihf-gcc"
    "darwin-amd64:darwin:amd64::"
    "darwin-arm64:darwin:arm64::"
    "windows-amd64:windows:amd64::x86_64-w64-mingw32-gcc"
)

foreach(target ${CROSS_COMPILE_TARGETS})
    string(REPLACE ":" ";" target_parts ${target})
    list(GET target_parts 0 target_name)
    list(GET target_parts 1 target_os)
    list(GET target_parts 2 target_arch)
    list(GET target_parts 3 goarm)
    list(GET target_parts 4 cc)

    set(output_name derbyvote-${target_name})
    if(target_os STREQUAL "windows")
        set(output_name ${output_name}.exe)
    endif()

    set(env_vars CGO_ENABLED=1 GOOS=${target_os} GOARCH=${target_arch})
    if(NOT goarm STREQUAL "")
        list(APPEND env_vars GOARM=${goarm})
    endif()
    if(NOT cc STREQUAL "")
        list(APPEND env_vars CC=${cc})
    endif()

    add_custom_target(build-${target_name}
        COMMAND ${CMAKE_COMMAND} -E env ${env_vars}
            ${GO_EXECUTABLE} build
            ${GO_BUILD_FLAGS}
            -ldflags "-s -w"
            -o ${BUILD_OUTPUT_DIR}/${output_name}
            ./cmd/derbyvote/...
        WORKING_DIRECTORY ${CMAKE_SOURCE_DIR}
        COMMENT "Building for ${target_os}/${target_arch} (requires cross-compiler)..."
    )
endforeach()

# Build all platforms
add_custom_target(build-all
    DEPENDS
        build-linux-amd64
        build-linux-arm64
        build-linux-armv7
        build-linux-armv6
        build-darwin-amd64
        build-darwin-arm64
        build-windows-amd64
    COMMENT "Building for all platforms..."
)

# Package target (creates a distributable archive)
set(PACKAGE_DIR ${CMAKE_BINARY_DIR}/package)
add_custom_target(package
    COMMAND ${CMAKE_COMMAND} -E make_directory ${PACKAGE_DIR}
    COMMAND ${CMAKE_COMMAND} -E copy ${BUILD_OUTPUT_DIR}/${BINARY_NAME} ${PACKAGE_DIR}/
    COMMAND ${CMAKE_COMMAND} -E copy_directory ${TEMPLATES_DIR} ${PACKAGE_DIR}/templates
    COMMAND ${CMAKE_COMMAND} -E tar czf ${CMAKE_BINARY_DIR}/derbyvote-${PROJECT_VERSION}.tar.gz ${PACKAGE_DIR}
    DEPENDS derbyvote
    WORKING_DIRECTORY ${CMAKE_BINARY_DIR}
    COMMENT "Creating distribution package..."
)

# Print help
add_custom_target(help-targets
    COMMAND ${CMAKE_COMMAND} -E echo "Available targets:"
    COMMAND ${CMAKE_COMMAND} -E echo "  derbyvote      - Build the server [default]"
    COMMAND ${CMAKE_COMMAND} -E echo "  upx            - Compress binary with UPX (after build)"
    COMMAND ${CMAKE_COMMAND} -E echo "  deps           - Download Go dependencies"
    COMMAND ${CMAKE_COMMAND} -E echo "  run            - Build and run the server"
    COMMAND ${CMAKE_COMMAND} -E echo "  run-dev        - Run with go run [faster]"
    COMMAND ${CMAKE_COMMAND} -E echo "  test           - Run tests"
    COMMAND ${CMAKE_COMMAND} -E echo "  test-coverage  - Run tests with coverage"
    COMMAND ${CMAKE_COMMAND} -E echo "  lint           - Run golangci-lint"
    COMMAND ${CMAKE_COMMAND} -E echo "  fmt            - Format Go code"
    COMMAND ${CMAKE_COMMAND} -E echo "  vet            - Run go vet"
    COMMAND ${CMAKE_COMMAND} -E echo "  clean-go       - Clean Go build cache"
    COMMAND ${CMAKE_COMMAND} -E echo "  package        - Create distribution archive"
    COMMAND ${CMAKE_COMMAND} -E echo ""
    COMMAND ${CMAKE_COMMAND} -E echo "Cross-compilation targets (requires cross-compilers):"
    COMMAND ${CMAKE_COMMAND} -E echo "  build-linux-amd64   - Build for Linux x64"
    COMMAND ${CMAKE_COMMAND} -E echo "  build-linux-arm64   - Build for Linux ARM64 (Pi 3/4/5)"
    COMMAND ${CMAKE_COMMAND} -E echo "  build-linux-armv7   - Build for Linux ARMv7 (Pi 2/3)"
    COMMAND ${CMAKE_COMMAND} -E echo "  build-linux-armv6   - Build for Linux ARMv6 (Pi Zero/1)"
    COMMAND ${CMAKE_COMMAND} -E echo "  build-darwin-amd64  - Build for macOS Intel"
    COMMAND ${CMAKE_COMMAND} -E echo "  build-darwin-arm64  - Build for macOS Apple Silicon"
    COMMAND ${CMAKE_COMMAND} -E echo "  build-windows-amd64 - Build for Windows x64"
    COMMAND ${CMAKE_COMMAND} -E echo "  build-all           - Build for all platforms"
    COMMAND ${CMAKE_COMMAND} -E echo ""
    COMMAND ${CMAKE_COMMAND} -E echo "For release build: cmake -DCMAKE_BUILD_TYPE=Release .."
    COMMENT "Showing available targets"
)

# Summary
message(STATUS "")
message(STATUS "DerbyVote Build Configuration")
message(STATUS "==================================")
message(STATUS "Version:      ${PROJECT_VERSION}")
message(STATUS "Build type:   ${CMAKE_BUILD_TYPE}")
message(STATUS "Binary:       ${BUILD_OUTPUT_DIR}/${BINARY_NAME}")
message(STATUS "")
message(STATUS "Run 'cmake --build . --target help-targets' for available targets")
message(STATUS "")
