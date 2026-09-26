# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This repository contains **Kddp** (Kompilierer Der Deutschen Programmiersprache), the compiler for the Deutsche Programmiersprache (DDP) — a statically-typed, imperative programming language designed to read like German. The compiler is written in Go, uses LLVM as its code generation backend, and includes a runtime library written in C and a standard library written in DDP itself.

### Key Components

- **Compiler Frontend** (`src/`): Go implementation of lexer, parser, AST, type checker, and semantic analysis
- **Code Generator** (`src/compiler/`): Converts typed AST to LLVM IR and native code
- **Runtime** (`lib/runtime/`): C library providing garbage collection, memory management, and runtime support
- **Standard Library** (`lib/stdlib/`): DDP standard library (Duden) including I/O, collections, and utilities
- **External Dependencies** (`lib/external/`): Vendored libraries (PCRE2, libarchive, zlib, liblzma, libbz2, liblz4)
- **CLI Tools** (`cmd/`): `kddp` (main compiler) and `ddp-setup` (installation utility)

## Compiler Architecture

The compilation pipeline follows this flow:

1. **Scanning** (`src/scanner/`): Tokenize source code into tokens
2. **Parsing** (`src/parser/`): Build Abstract Syntax Tree (AST) from tokens
3. **Type Checking & Analysis** (`src/typechecker/`, `src/compiler/annotators/`): Semantic analysis, type resolution, and symbol table management
4. **IR Generation** (`src/compiler/`): Translate typed AST to LLVM Intermediate Representation
5. **Code Generation**: LLVM compiles IR to object files; GCC links everything into executables

Key compiler entry point: `src/compiler/compiler.go`

**Important**: The compiler is tightly integrated with LLVM via CGO bindings (`src/compiler/llvm_bindings.go`). LLVM must be properly installed or built before compilation.

## Build & Development Commands

### Building

```bash
make                    # Build kddp, runtime, stdlib, and ddp-setup (default target)
make -j8                # Build with 8 parallel jobs (recommended)
make kddp               # Build only the kddp compiler binary
make runtime            # Build only the runtime library
make stdlib             # Build only the standard library
make ddp-setup          # Build only the setup/install utility
make debug              # Build with debug symbols (slower, prints debug info at runtime)
make kddp-debug         # Build kddp with debug symbols
make runtime-debug      # Build runtime with debug symbols
make stdlib-debug       # Build stdlib with debug symbols
```

Outputs are placed in `build/DDP/`:
- `build/DDP/bin/kddp(.exe)` — the compiler executable
- `build/DDP/bin/ddp-setup(.exe)` — setup tool
- `build/DDP/lib/stdlib/` — standard library and Duden modules
- `build/DDP/lib/runtime/` — runtime libraries

### Testing

```bash
make test               # Run all tests (unit + normal + memory leak + coverage)
make test-normal        # Run functional tests (parser, compiler, stdlib)
make test-memory        # Run memory leak detection (requires debug build)
make test-unit          # Run only Go unit tests
make test-sumtypes      # Validate that sumtypes in the source tree are correctly used
make coverage           # Generate coverage report for stdlib tests
make test-with-optimizations    # Run all tests with -O2 optimization
make test-without-optimizations # Run all tests with -O0 (no optimization)
```

### Running Specific Tests

Tests are in `tests/` and driven by Go's test framework:

```bash
# Run only a specific test function
go test -v ./tests -run TestKDDP

# Run tests in a specific test directory (if set via env var)
DDPTEST_TEST_DIRS=testdata/stdlib make test-normal

# Pass extra compiler arguments to tests
DDPTEST_KDDP_ARGS="-O0" make test-normal
```

### LLVM Management

```bash
make llvm               # Clone and build LLVM from llvm-project submodule (2-3 hours)
make checkout-llvm      # Just clone the LLVM submodule (doesn't build)
```

The Makefile automatically detects a locally-built LLVM in `llvm_build/` and uses it in preference to system LLVM.

### Cleaning

```bash
make clean              # Remove build/ directory and clean all components
make clean-outdir       # Remove only build/DDP/ directory
make clean-all          # clean + clean external dependencies
make clean-cmd          # Clean only compiler binaries
make clean-runtime      # Clean only runtime build artifacts
make clean-stdlib       # Clean only stdlib build artifacts
```

### Code Formatting

```bash
make format             # Format stdlib and runtime (C code only)
make format-stdlib      # Format standard library
make format-runtime     # Format runtime library
```

### Help

```bash
make help               # Display all available make targets with descriptions
```

## Directory Structure

```
.
├── cmd/                    # Compiler binaries and CLI tools
│   ├── kddp/               # Main compiler source (Go)
│   ├── ddp-setup/          # Installation utility (Go)
│   └── internal/           # Shared internal utilities
├── src/                    # Compiler implementation (Go)
│   ├── compiler/           # Code generation to LLVM IR
│   ├── parser/             # Parsing DDP source code to AST
│   ├── scanner/            # Tokenization
│   ├── typechecker/        # Type inference and checking
│   ├── ddperror/           # Error handling
│   ├── ddptypes/           # Type system definitions
│   ├── ast/                # Abstract Syntax Tree definitions
│   └── token/              # Token definitions
├── lib/                    # Runtime and standard library
│   ├── runtime/            # Runtime library (C)
│   │   ├── source/DDP/     # Core runtime (GC, memory, builtins)
│   │   └── include/DDP/    # Runtime headers
│   ├── stdlib/             # Standard library (DDP source)
│   │   ├── Duden/          # Duden modules (language std libs)
│   │   ├── source/         # Stdlib implementation
│   │   └── include/        # Stdlib headers
│   ├── external/           # Vendored C libraries
│   │   ├── pcre2_build/    # Regex library
│   │   ├── libarchive/     # Archive handling
│   │   └── ...             # zlib, liblzma, libbz2, liblz4
│   └── gc_strategy/        # Garbage collection strategy (C)
├── tests/                  # Test suite (Go)
│   ├── kddp_test.go        # Main compiler tests
│   ├── stdlib_coverage_test.go  # Stdlib tests
│   └── testdata/           # Test files (DDP source and expected outputs)
├── llvm-project/           # LLVM source (submodule)
├── llvm_build/             # LLVM build artifacts (auto-generated)
├── build/DDP/              # Final build output (auto-generated)
├── .github/workflows/      # CI/CD workflows
├── Makefile                # Main build system
├── go.mod / go.sum         # Go module dependencies
└── CONTRIBUTING.md         # Contribution guidelines
```

## Key Files to Know

- **`cmd/kddp/main.go`**: Compiler entry point; handles CLI flags and orchestrates compilation
- **`src/compiler/compiler.go`**: Core compiler struct; coordinates all compilation phases
- **`src/compiler/llBuilder.go`**: LLVM IR builder; generates LLVM code for each AST node
- **`src/compiler/runtime_bindings.go`**: Bindings to runtime functions (GC, allocators, etc.)
- **`lib/runtime/source/DDP/gc.c`**: Garbage collection implementation
- **`lib/stdlib/Makefile`**: Builds the standard library into a static archive
- **`Makefile`**: Root build orchestration; delegates to component Makefiles
- **`cmd/Makefile`**: Handles compiler and CLI tool compilation

## Important Concepts

### Memory Management

- **Garbage Collection**: The runtime uses a mark-and-sweep GC (see `lib/runtime/source/DDP/gc.c`)
- **Pointer Tagging**: Type information is encoded in the low bits of pointers for efficiency (see `src/compiler/pointer_tagging.go`)
- **Reference Types**: DDP has reference types distinct from value types; handled specially during code generation

### Type System

- Types are defined in `src/ddptypes/` (base types) and `src/compiler/ir_*_type.go` (IR types)
- Generic list types are a key feature (see `src/compiler/list_types.go`)
- The compiler performs type inference and checking in `src/typechecker/`

### Testing

- Tests in `tests/` compile DDP programs and verify output against expected results
- Memory tests build with debug symbols and run under memory leak detection
- Coverage reports exercise the stdlib by compiling and executing programs

## Debugging & Development Tips

1. **Debug Output**: Build with `make debug` to enable runtime debug logging (printed to stderr)
2. **Compiler Debug**: Build kddp with `make kddp-debug` for more compiler output
3. **LLVM IR**: Use kddp's `-dump-ir` flag to output generated LLVM IR (useful for debugging codegen)
4. **Environment Variables**: 
   - `DDPTEST_KDDP_ARGS`: Pass extra flags to kddp during testing
   - `DDPTEST_TEST_DIRS`: Restrict tests to specific directories
5. **Memory Leaks**: Run `make test-memory` after `make debug` to detect leaks using Valgrind or similar tools
6. **Test Isolation**: Individual tests live in `tests/testdata/*/` directories and are self-contained

## Common Development Workflow

1. Make changes to compiler or runtime code
2. Build: `make -j8`
3. Run tests: `make test` (or `make test-normal` for faster iteration without memory checks)
4. Debug specific test: `DDPTEST_TEST_DIRS=testdata/dirname make test-normal`
5. Clean and rebuild: `make clean && make -j8`

## External Resources

- **Language Docs**: https://ddp.le0n.dev/Bedienungsanleitung/ (German user guide)
- **Playground**: https://ddp.le0n.dev/Spielplatz (browser-based IDE)
- **Contributing Guide**: `CONTRIBUTING.md` in this repo
- **GitHub Releases**: Pre-compiled binaries and LLVM builds at https://github.com/DDP-Projekt/Kompilierer/releases
