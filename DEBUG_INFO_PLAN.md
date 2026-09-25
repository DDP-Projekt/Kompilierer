# Emit DWARF debug info from the DDP compiler

## Context

You're chasing a difficult GC/compiler segfault, and you've already been manually
unwinding with libunwind and resolving the resulting PCs with LLVM tools — but
without line information that workflow tops out fast. You also want proper DWARF
debug info as a permanent compiler feature, not just a one-off debugging hack, so
this plan wires up real `DISubprogram`/`DILocation` emission rather than a
throwaway PC dump.

The good news from exploration: almost everything needed already exists, just
unconnected.

- `src/compiler/llvm/dibuilder.go` is a **complete** `DIBuilder` Go wrapper
  (`CreateCompileUnit`, `CreateFile`, `CreateFunction`, `CreateLexicalBlock`,
  `CreateAutoVariable`/`CreateParameterVariable`, type builders, etc.) — fully
  functional, currently called from nowhere in `src/compiler/*.go`.
- `src/compiler/llvm/ir.go:1458-1471` already wraps
  `Builder.SetCurrentDebugLocation(line, col, scope, inlinedAt)` and
  `Value.SetSubprogram` (`dibuilder.go:621-628`) — also unused.
- `visitNode` (`src/compiler/compiler.go:390-396`) already tracks a "current AST
  node" on every single statement/expression visit, and every `ast.Node` exposes
  `GetRange() token.Range` → `Position{Line, Column uint}`
  (`src/token/range.go:11-14`) — this is the natural hook to set a `!dbg`
  location per instruction with almost no new plumbing.
- `libunwind` is already linked into every DDP executable
  (`cmd/internal/linker/link.go:134`) and is already load-bearing for the GC's
  precise stack-root scanning (`lib/runtime/source/DDP/gc.c:998-1089`), so the
  unwinding side of your workflow is already proven — this plan only adds the
  missing symbolization data (DWARF `.debug_info`/`.debug_line`) so `llvm-symbolizer`/
  `addr2line`/GDB can turn those PCs into `file:line` + function name.

One real gap found: `LLVMAddModuleFlag` (needed to set `"Debug Info Version" = 3`,
without which LLVM silently strips all debug metadata) exists in the vendored
LLVM C headers but is **not yet wrapped** in `src/compiler/llvm/ir.go` — small,
first thing to add.

Two things intentionally **out of scope** here, since this task was scoped to
compiler-emitted DWARF only (not the in-process runtime backtrace option):

- The `SIGSEGV` handler (`lib/runtime/source/DDP/runtime.c:18-24`) currently runs
  a full GC pass and `exit()`s before you'd ever get a debugger/core dump — worth
  fixing separately (`sigaction`+`SA_SIGINFO`, no GC before reporting), but it's
  independent of DWARF emission and not part of this plan.
- Variable-level debug info (`dbg.declare`, `DIAutoVariable`/`DIParameterVariable`,
  and full `DIType`s for DDP's type system) is not needed to get named,
  line-accurate stack frames — skipped for now, noted as follow-up.

## What gets attached, and where

Per LLVM/DWARF requirements, three layers of metadata, all via the existing
`DIBuilder` wrapper:

1. **Module-level, once per `llvm.Module`** (one DDP source file = one `compiler`/
   one `llmod`, per `newCompiler`, `src/compiler/compiler.go:210-244`):
   - `"Debug Info Version"` module flag = 3 (new binding needed, see below) —
     without this LLVM drops all DI with no error.
   - One `DICompileUnit` (`dibuilder.CreateCompileUnit`) — needs a `DwarfLang`;
     only `DW_LANG_Go` is defined in `dibuilder.go:64`, so add a second constant
     (reuse an existing DWARF language code, e.g. `DW_LANG_C99`/0x0c, as a
     stand-in producer language — doesn't need to be a registered "DDP" DWARF
     language to work with GDB/LLDB/llvm-symbolizer).
   - One `DIFile` for `c.ddpModule.FileName`.

2. **Per function, once per `llvm.Value` function** — hook into
   `createBuilder` (`src/compiler/llBuilder.go:149-190`), which is the single
   place every function (user `FuncDecl`/`FuncDef`, `ddp_ddpmain`,
   `module_init`/`module_dispose`) gets created:
   - A minimal `DISubroutineType` — `CreateSubroutineType(diFile, nil, 0)` is
     valid and sufficient; you don't need real DDP `DIType`s for named,
     line-accurate frames.
   - `dibuilder.CreateFunction(...)` → `DISubprogram`, attached via
     `builder.llFn.SetSubprogram(sp)`.
   - Store it on `llBuilder` (new field, e.g. `diSubprogram llvm.Metadata`) so
     `visitNode` can use it as the scope for every instruction in that function.
   - Set an initial `SetCurrentDebugLocation` at function entry (using the
     function's own def line) so the parameter-copy allocas/stores emitted in
     `defineFuncBody` (`compiler.go:893-960`, before the first real statement is
     visited) still get a valid `!dbg` location — LLVM requires every
     instruction in a function with a subprogram to have one, or the verifier
     (once re-enabled) will reject the module.
   - For compiler-synthesized functions with no matching `ast.FuncDecl`
     (`ddp_ddpmain`, `module_init`/`dispose`), a best-effort `DISubprogram` at
     line 0/1 is fine — they don't need per-statement precision.

3. **Per instruction, via the existing `visitNode`** (`compiler.go:390-396`):
   extend the existing save/restore-`currentNode` pattern to also
   save/restore the debug location, symmetrically:
   ```go
   func (c *compiler) visitNode(node ast.Node) {
       b := c.builder()
       oldNode := b.currentNode
       b.currentNode = node
       if b.diSubprogram.C != nil {
           r := node.GetRange()
           b.SetCurrentDebugLocation(r.Start.Line, r.Start.Column, b.diSubprogram, llvm.Metadata{})
       }
       node.Accept(c)
       b.currentNode = oldNode
       if b.diSubprogram.C != nil && oldNode != nil {
           r := oldNode.GetRange()
           b.SetCurrentDebugLocation(r.Start.Line, r.Start.Column, b.diSubprogram, llvm.Metadata{})
       }
   }
   ```
   This gives every emitted instruction — including calls, which is what
   actually shows up in a stack trace — a source line, with no per-call-site
   changes needed elsewhere (`createCall`/`callAsStatepoint` in `llBuilder.go`
   don't need to change; they inherit whatever location is "current" on the
   builder when they run).

## Concrete changes

- **`src/compiler/llvm/ir.go`**: add `Module.AddModuleFlag(behavior ModuleFlagBehavior, key string, value uint32)` wrapping `LLVMAddModuleFlag` (present in the vendored `llvm-c/Core.h` but currently unwrapped), plus a small `ModuleFlagBehavior` enum (`Warning = 2` is the one you need).
- **`src/compiler/llvm/dibuilder.go`**: add a second `DwarfLang` constant next to `DW_LANG_Go` (`dibuilder.go:60-65`) to use as the compile unit's language.
- **`src/compiler/interface.go`**: add `Options.EmitDebugInfo bool`, threaded into `newCompiler` (`compiler.go:210`) — keep it opt-in rather than always-on, since DI interacts with `RunPasses("default<O2>", ...)` (`llvm_bindings.go:130`) and you'll want to validate correctness at `-O0` first.
- **`src/compiler/compiler.go`**:
  - `compiler` struct: add `diBuilder llvm.DIBuilder`, `diCompileUnit`, `diFile llvm.Metadata`.
  - `newCompiler`/`setup()`: when `EmitDebugInfo`, create the `DIBuilder`, set the module flag, create the compile unit + file.
  - `visitNode`: the save/restore change above.
  - `compile()`: call `c.diBuilder.Finalize()` once, before `c.disposeBuilders()` (currently line 320), and before/alongside re-enabling `llvm.VerifyModule` (`compiler.go:322-324`, currently commented out) — re-enabling the verifier is important here specifically to catch malformed DI metadata (missing `!dbg` on some instruction, bad scope nesting) as a clear verifier error instead of an opaque backend crash later.
- **`src/compiler/llBuilder.go`**: add `diSubprogram llvm.Metadata` field; in `createBuilder` (149-190), when `c.diBuilder` is set and `!declarationOnly`, build and attach the `DISubprogram` as described above.
- **`src/compiler/llvm_bindings.go`**: re-enable `options.SetVerifyEach(true)` (line 126-128) alongside the module verifier, at least while validating this feature — same reasoning (surface bad DI immediately, not as a miscompile).
- **`cmd/kddp/build_cmd.go`**: add a `--debug-info`/German-named flag next to the existing `optimierungs-stufe` flag (`build_cmd.go:154,195,208`) wired to the new `Options.EmitDebugInfo`, so the feature is actually reachable from the CLI for testing.
- **`lib/runtime/Makefile`**: add `-g` to `CCFLAGS` (line ~10, currently no `-g` anywhere in the runtime build) so the C runtime frames (`gc.c`, `runtime.c`, etc.) also resolve to source lines — a mixed DDP+C backtrace is only fully readable if both sides have DWARF. One-line change, independent of the compiler work, but worth doing in the same pass since it directly serves "line information" for the frames most likely to be involved in the current GC bug.

## Risks / things that fail silently (watch for these)

- Forgetting the `"Debug Info Version"` module flag → LLVM strips all DI, no error, you just get no `.debug_info` and won't know why.
- Missing `!dbg` on any instruction in a function that has a subprogram → verifier error (once re-enabled) or, if the verifier stays off, a confusing backend crash. The entry-of-function default location plus the visitNode hook should cover this, but compiler-synthesized calls that happen *outside* a `visitNode` call (if any exist) need checking.
- Optimizations (`-O2`) can legitimately drop/merge/move `!dbg` locations — validate at `-O0` first to confirm the DI pipeline itself is correct, then check how much fidelity survives at `-O2`.
- The GC's correctness depends on `.llvm_stackmaps` PC→root-location records staying accurate (`gc.c:998-1089`, `find_stackmap_record`) — DI metadata itself shouldn't move code, but this is worth explicitly re-verifying (run a GC-heavy test program end to end) after turning DI on, since it's the one thing in this repo where "just metadata" bugs would show up as a *different* segfault, not a compile error.

## Verification

1. Build with `-O 0 --debug-info` (or whatever the new flag ends up named) on a small DDP program with a few nested function calls.
2. `llvm-dwarfdump --debug-line <obj/exe>` — confirm a line table exists and maps back to the right `.ddp` source lines.
3. `gdb --batch -ex 'break <ddp funcname>' -ex run -ex bt ./program` — confirm DDP frames show real function names and `file:line`, not just addresses.
4. Repeat with `llvm-symbolizer`/`addr2line -e ./program <pc>` on a PC obtained the way you already do (your existing libunwind-based collection) — this is the actual workflow you'll use to debug the GC segfault, so confirm it now resolves to a line.
5. Run the existing DDP test suite (and specifically a GC-heavy program, given the recent GC bug history) at both `-O 0` and `-O 2` with DI enabled, to catch any interaction with the statepoint/stackmap pipeline before trusting it for the real debugging session.
6. Re-run once without `--debug-info` to confirm nothing regressed for the default (no-DI) path.

## Status (implemented)

Implemented as described above: `--debug-informationen` flag on `kompiliere`,
module flag + `DICompileUnit`/`DIFile` in `newCompiler`/`setupDebugInfo()`
(`src/compiler/compiler.go`), `DISubprogram` per function in `createBuilder`
(`src/compiler/llBuilder.go`), per-instruction `!dbg` via `visitNode`, verifier
re-enabled (gated on `EmitDebugInfo`, not the global `DEBUG` flag, to avoid
surfacing unrelated latent verifier issues in every debug build), `-g` added
to the runtime's release `CCFLAGS`. Verified with `objdump -h` on a built
`.exe`: `.debug_info`/`.debug_line`/`.debug_frame`/etc. are present and
non-empty.

Also fixed in passing: `print_backtrace()` (`lib/runtime/source/DDP/common.c`)
was truncating `unw_word_t pc/off` (== `uintptr_t`, 8 bytes) to `long` (only
4 bytes on Windows/LLP64) before printing with `%lx`, corrupting every
printed address on Windows regardless of DI. Fixed to use `PRIxPTR`.

### Follow-up (not done): in-process name/file:line resolution

`print_backtrace()`'s frames still print as `-- <unknown>` on Windows even
with the DWARF info now correct and the address bug fixed. Traced to
`llvm-project/libunwind/src/AddressSpace.hpp:668-694`,
`LocalAddressSpace::findFunctionName`: on Windows it's an unconditional stub
that always returns `false` (the only implemented paths are `dladdr()` on
Linux/macOS, and AIX traceback tables) — `unw_get_proc_name()` can never
succeed here, independent of debug info. And even where `dladdr()` does
work, it resolves names from the symbol table, not from `.debug_info`/
`.debug_line`, so it would never give file:line either way.

To get real in-process name+file:line resolution from the DWARF this plan
now emits (no manual external-tool step at crash time), a DWARF-consuming
library is needed — e.g. **libbacktrace** (bundled with GCC, parses ELF/PE +
DWARF directly, no subprocess; what Rust/Julia/etc. use for this). The
alternative is shelling out to `addr2line`/`llvm-symbolizer` from
`print_backtrace()` itself. Decided to leave this as a future task for now;
current workflow is: `print_backtrace()` prints correct raw PCs, resolve
externally with `llvm-symbolizer`/`addr2line -e <exe> <pc>` or `gdb`.
