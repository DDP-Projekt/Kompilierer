# Fix remaining GC segfaults in `build/test.ddp`

## Context

This is a multi-bug debugging session for segfault(s) in `build/test.ddp` (parses `"kddp hallo --ausgabe welt"` via `Duden/Befehlszeile`). **Three** real, independent bugs have been found and fixed so far. A fourth bug (the original one this session set out to find) is still open, and a **fifth, possibly-related** bug was just discovered via GDB and is not yet root-caused. Picking this up again: read this whole file before doing anything, especially "Where to pick this back up".

### Already fixed (confirmed, do not re-touch)

1. **`ddp_deep_copy_any` publish-before-init** (`lib/runtime/source/DDP/ddptypes.c:94-118`): the destination `value_ptr` buffer wasn't zeroed before the field-by-field `deep_copy_func` ran, so a GC cycle triggered mid-copy could read garbage as a nested Any's `vtable_ptr`. Fixed with `memset(ret->value_ptr, 0, ...)` right after allocation. Also added `memset(any->value, 0, DDP_SMALL_ANY_BUFF_SIZE)` in `ddp_free_any` for the small-any case (confirmed safe: `DDP_IS_SMALL_ANY` only reads `vtable_ptr->type_size`, never the value buffer).
2. **`Duden/HashTabelle.ddp` copy-paste bug**: `Tabellen_Wert_Lesen`, `Tabelle_Hat_Schlüssel`, and `Tabellen_Wert_Löschen` each had the resize-on-write block copy-pasted in from `Tabellen_Wert_Setzen`, causing pure reads/contains-checks/deletes to spuriously allocate a 16-slot backing array as a side effect. Fixed by replacing each with a cheap `Wenn die Kapazität von tabelle gleich 0 ist, ...` early-out. Confirmed real, but was **not** the cause of bug #4 below — that crash signature was byte-identical before and after this fix.
3. **`createStructDeepCopy` publish-before-init** (`src/compiler/ir_struct_type.go`, `createStructDeepCopy`, around line 252-278): same hazard class as bug #1, but at the compiler level for arbitrary struct types (not just `Any`). The generated `ddp_deep_copy_<Type>` function copied fields into `ret` one at a time **without ever zeroing `ret` first**, and unlike `trace_any`, generic struct/ptrmask tracing has no "vtable_ptr == NULL → skip" safety valve — a not-yet-copied `Any`-typed field (e.g. `Option.standardwert`) read as raw garbage if a GC cycle fired mid-copy. Fixed by storing the struct's zero value into `ret` as the very first instruction:
   ```go
   llFuncBuilder.CreateStore(llvm.ConstNull(structTyp.typ), ret)
   ```
   **Important gotcha hit while writing this fix**: `structTyp.DefaultValue()` looks like the obvious thing to use here, but it isn't set yet at this point — `defineOrDeclareStructType` calls `createStructDeepCopy` *before* it sets `structType.defaultValue = llvm.ConstNull(structType.typ)`. Using `DefaultValue()` here crashes the *compiler itself* (`LLVMBuildStore` called with a Go zero-value `llvm.Value{}`, i.e. a null value — this is a real, reproducible Go-level panic, not a runtime bug). Use `llvm.ConstNull(structTyp.typ)` directly instead, as the fix above does.
   - **Confirmed fixed**: this eliminated a real, reproducible crash — `trace_any` in `gc.c:832` (`DDP_ANY_VALUE_PTR(any)`) with a garbage `any` pointer, reached via `Option_Hinzufügen` (`Befehlszeile.ddp:88`, `option als Variable` — a "big" Any box of the 48-byte `Option` struct, which goes through `castNonAnyToAny` → `claimOrCopy` → `deepCopyInto` → this exact generated function) → `Tabellen_Wert_Setzen`'s own end-of-function cleanup (`ddp_free_string` on its local `schlüssel` copy) → `ddp_reallocate` → opportunistic `ddp_gc()` → `mark_stack_roots` finds the still-live `wert` parameter (the freshly-boxed Option) and traces into its now-corrupted `standardwert` field. Before the fix, this crashed on every direct (non-debugger) run. After the fix, direct execution no longer hits it — it proceeds further into the program (see bug #4/#5 below).

### Diagnostic infrastructure added along the way (keep, useful going forward)

- `lib/runtime/source/DDP/common.c`: `print_backtrace()` — libunwind (`unw_step`) stack walk, symbolized via `libbacktrace` (DWARF) with a `GetModuleHandleEx`/`unw_get_proc_name` fallback for frames without debug info.
- `lib/runtime/source/DDP/runtime.c`: on Windows, `ddp_init_runtime` installs `CrashFilter` via `SetUnhandledExceptionFilter` instead of `signal(SIGSEGV, ...)`. Gives the real `EXCEPTION_RECORD` — exact faulting instruction address, and for access violations, read/write kind + exact faulting memory address — printed unbuffered before `print_backtrace()` runs. Non-Windows path unchanged.
- `src/compiler/compiler.go` / `src/compiler/llBuilder.go`: DIBuilder wiring so DDP-generated functions get real `!dbg` locations (file:line resolve in backtraces **and** GDB can set symbolic breakpoints on DDP-generated function names, e.g. `break ddp_free_Befehlszeile_mod_<hash>`).
- `lib/runtime/include/DDP/debug.h`: `DDP_DBGLOG` now flushes.
- `ddp_free_string` (`lib/runtime/source/DDP/ddptypes.c:27-33`) has an added unconditional `fprintf(stderr, "ddp_free_string: str=%p\n", ...)` at entry — useful, but the user has since rebuilt `kddp`/the runtime **without** `DDP_DBGLOG` (too slow to iterate on) and instead compiles `test.ddp` with the compiler's `--debug-informationen` flag (`cmd/kddp/build_cmd.go`, `EmitDebugInfo`) to get DWARF for GDB. **Going forward, prefer GDB over the DDP_DBGLOG-log-and-grep workflow** — it's much faster now that debug info is wired up on both the compiler-generated and C-runtime sides. GDB is at `C:\Users\Hendrik\mingw64\bin\gdb.exe`.
- Rebuild reminder that has cost real time twice: **`kddp.exe` is not auto-rebuilt** — always run `make kddp` after editing any `.go` file under `src/compiler/`, and confirm `build\DDP\bin\kddp.exe`'s timestamp is newer than your edit, before recompiling `test.ddp`.
- To compile with debug info: `.\build\DDP\bin\kddp.exe kompiliere --debug-informationen -o build\test.exe build\test.ddp`.

## Bug #4 (original target of this session): double-free during `Befehlszeile` cleanup

Still open. Confirmed facts, gathered across many runs before bug #3 was found (the crash signature was identical across all of them, i.e. bug #3 was not the cause of this one):

- **Call chain**: `ddp_ddpmain` (`test.ddp:7`, implicit end-of-scope free of `b`) → `ddp_free_Befehlszeile` (`Befehlszeile.ddp:123`) → `ddp_free_Text-Variable-Text-Variable-HashTabelle` (`HashTabelle.ddp:90`, i.e. one of `Befehlszeile.kurzschreibweisen`/`.langschreibweisen`) → `ddp_free_ddpText-Variable-Text-Variable-Eintraglist` → `ddp_free_Text-Variable-Text-Variable-Eintrag` (`HashTabelle.ddp:47`) → `ddp_free_string` (`ddptypes.c:31`, the `str->str = NULL;` line).
- **Confirmed via `SetUnhandledExceptionFilter`**: `EXCEPTION_ACCESS_VIOLATION`, a **write**, faulting IP exactly matches the `ddp_free_string` frame. Faulting address varies per run with ASLR, but the pattern is reproducible: that exact address was used as a stable, frequently-recycled `ddpstring` slot **100+ times** over the course of the run (repeated `_ddp_deep_copy_string`/`ddp_free_string` pairs at the identical address), successfully freed one last time shortly before, then faults on a second free attempt with nothing logged touching it in between.
- **Ruled out**: the "arrlen=16 `Text-Variable-Eintrag` table" GC sweep that always appears immediately before the crash in the log is a red herring — different memory region entirely from the faulting address.
- **Ruled out**: field-index aliasing between `kurzschreibweisen`/`langschreibweisen` at the compiler level. `c.indexStruct` (`src/compiler/ir_helper.go:96-98`) is a plain positional GEP, not type-keyed.
- **Ruled out**: premature-GC-during-explicit-free via missing `structParam` liveness registration in `createStructFree` — a fix was applied (`scp.addTemporary`/`claimTemporary` around the struct pointer) but had **zero effect** on the crash, confirming this wasn't the mechanism.

**Leading hypothesis, still not confirmed**: `kurzschreibweisen` and `langschreibweisen` end up referencing (aliasing) the same underlying heap array, so `ddp_free_Befehlszeile`'s two separate free calls walk the same array twice. Alternative: a bug within a single table's own rehashing (`Kapazität_Anpassen`, `Duden/HashTabelle.ddp:145-162`) leaves one logical entry reachable at two indices in the same array.

**After fixing bug #3, this crash was reproduced once more** via direct (non-GDB) execution — same chain, same pattern, new address (`0x20cda1e0c08`) — confirming it's still present and unrelated to bug #3.

## Bug #5 (new, found via GDB, not yet root-caused): a second timing-sensitive publish-before-init-shaped crash

While trying to reach bug #4 under GDB (to set a breakpoint at `ddp_free_Befehlszeile` and directly inspect `kurzschreibweisen`/`langschreibweisen`'s array pointers — see the GDB technique below), the breakpoint was **never hit**. Instead, running the *exact same, already-fixed* binary under GDB reliably (2/2 attempts) crashes **earlier**, with the same shape as bug #3 but through a different path:

```
#0 trace_any (any=0x...) at gc.c:832        <- DDP_ANY_VALUE_PTR(any), garbage pointer
#1 trace_any (any=0x5ffa..) at gc.c:852     <- tracing nested any, i=3 (Option.standardwert offset)
#2 trace_root at gc.c:869
#3 mark_stack_roots at gc.c:1086
#4-5 ddp_gc
#6 ddp_reallocate (oldSize=10, newSize=0)   <- a free, not an allocation
#7 ddp_free_string (str=0x5ff9..) at ddptypes.c:30
#8 <Option_Hinzufügen_mod_...> at Befehlszeile.ddp:88
#9 ddp_ddpmain at test.ddp:1
#10 main
```

This is the **same shape** as bug #3 (an `Option.standardwert` Any traced as garbage, triggered by an opportunistic GC cycle during `ddp_free_string`'s cleanup inside `Tabellen_Wert_Setzen`, called from `Option_Hinzufügen` line 88's `option als Variable` boxing) — but bug #3's fix is confirmed present in this binary (verified: direct execution of the same binary does *not* hit this crash, only GDB-driven execution does, consistently). So either:
- (a) it's a genuinely separate, still-unfixed race in the same general area (a different construction path than the one bug #3 fixed — `evaluateStructLiteral`, `compiler.go:2404-2451`, was inspected as a candidate and *looks* structurally correct: fields are registered as individually-protected GC temporaries one at a time, only after being written — so no confirmed second bug yet, just a reproducible symptom), or
- (b) it's the *same* underlying corruption as bug #4, just exposed earlier by GDB's altered timing (Windows enables the debug heap under a debugger, which changes allocation timing/addresses) — in which case root-causing this might resolve bug #4 too.

**Not yet determined which.** This is genuinely reproducible under GDB (unlike bug #4, which GDB currently prevents you from ever reaching), so it's a good target to chase next with GDB's more powerful tools (watchpoints, or `record`/`reverse-continue` to find the actual corrupting write) — but stopped here for this session.

## GDB technique notes (for resuming)

- Binary: `C:\Users\Hendrik\mingw64\bin\gdb.exe`. Compile `test.ddp` with `--debug-informationen` first.
- DDP-generated function names are mangled with a content hash, e.g. `ddp_free_Befehlszeile_mod_fa225fb88194813eb286a5f9d47dedf7687f78d8dc43854ef8941674c07148e5` — get exact names via `objdump`/`nm` on `test.ll`/`test.exe`, or from a prior backtrace.
- DIBuilder currently only attaches `DISubprogram` info (function name + line), **not** `DILocalVariable`/parameter type info — so `info args`, `print <param_name>` etc. **do not work**. To inspect a compiler-generated function's struct-pointer parameter, disassemble it first (`disassemble <mangled_name>`) to see the calling convention in use (Windows x64: first arg in `$rcx` at function entry) and known field offsets (computed by hand from the DDP struct's field order — e.g. `Befehlszeile`: `kurzschreibweisen` at offset `0x0`, `langschreibweisen` at `0x20`, `optionWerte` at `0x40`, `argumente` at `0x58`, `unterBefehle` at `0x70`; each `Tabelle`/`Text-Variable-HashTabelle` is `{einträge: {arr:ptr, len:i64, cap:i64}, länge:i64}` = 32 bytes), then break with the **symbolic** function name (not a raw `break *0x...` address — the raw disassembled addresses are pre-ASLR-relocation and GDB will fail to insert the breakpoint) and read fields directly via GDB's `printf`, e.g.:
  ```
  break ddp_free_Befehlszeile_mod_<hash>
  run
  printf "kurz.arr = %p\n", *(void**)($rcx+0x0)
  printf "lang.arr = %p\n", *(void**)($rcx+0x20)
  ```
- This exact inspection is still pending for bug #4 — never got to run it because bug #5 intercepts execution first under GDB.

## Where to pick this back up

Two independent threads, either is a reasonable next step:

1. **Bug #5** (reproducible under GDB): use a watchpoint or `record full` + `reverse-continue` to catch the actual write that corrupts the `Option.standardwert` Any, to determine if it's a new bug or the same root cause as bug #4.
2. **Bug #4** (reproducible in direct/non-GDB execution): run `test.exe` directly (not under GDB) to reliably reach the crash; since GDB currently can't reach this point (bug #5 intercepts first), the `createStructFree` field-pointer inspection plan may need `fprintf`-based instrumentation again instead of GDB, *or* fix bug #5 first so GDB can run past it and reach bug #4's breakpoint.

Given they may be the same bug, **(1) is probably higher-leverage** — if it turns out to be the same root cause, bug #4 may resolve for free.

## Verification (once a fix is applied)

1. `make kddp` — confirm `build\DDP\bin\kddp.exe` is newer than any edited source before testing.
2. `.\build\DDP\bin\kddp.exe kompiliere --debug-informationen -o build\test.exe build\test.ddp`
3. `.\build\test.exe` directly (fast, no debug log) — confirm no `--- crash ---` output and expected program behavior.
4. Also try under GDB (`& gdb.exe -batch -ex run -ex bt -ex kill .\build\test.exe`) to make sure the fix holds under the debug-heap timing too, not just direct execution.
5. Re-run `make test-memory` / `make test-normal` to catch regressions.
