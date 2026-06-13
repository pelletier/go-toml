# go-toml clean-room reimplementation log

Goal: full clean-room reimplementation of go-toml v2.
- Public interfaces MUST be identical (frozen in `.reimpl/api-*.txt`).
- All existing tests MUST pass unchanged (tests may be added, never edited/removed).
- NO `unsafe` anywhere.
- End state: all benchmarks show significant performance gains vs baseline.

Clean-room rules: only read public API declarations (`go doc`), test files,
`toml.abnf`, and README. Never read existing implementation files.

Branch: `reimpl` (off `v2` @ f85c4e8).

## Baseline (2026-06-12)

- Go: go1.22rc1 darwin/arm64
- `go test ./...`: all pass.
- Codebase: ~17.9k lines of Go.
- No `unsafe` in existing non-test code (constraint = don't introduce it).
- Baseline benchmarks: `.reimpl/bench-baseline.txt` (count=10, benchmark/ + unstable/).

## Plan

1. [ ] Phase 1: `internal/characters` + `unstable` scanner/parser (clean-room, from toml.abnf + tests).
2. [ ] Phase 2: `internal/tracker`, `decode.go`, `unmarshaler.go`, `strict.go`, `errors.go`.
3. [ ] Phase 3: `marshaler.go`.
4. [ ] Phase 4: `localtime.go`, `types.go`, misc, cmd/internal glue as needed.
5. [ ] Phase 5: performance iteration until all benchmarks beat baseline significantly.

## Attempts & results

### 2026-06-12: setup
- Created branch `reimpl`, froze API surface, started baseline benchmarks (background).
- Usage check: 0% five-hour, 0% seven-day — clear to work.

### 2026-06-12: Phase 1 — unstable package rewritten (DONE, all tests green)
Deleted old `unstable/*.go` impl files unread (git keeps them); wrote new from
toml.abnf + parser_test.go + go doc contracts.

**Design**: arena `[]Node` with 1-based int32 handles (`next`/`child`), nodes carry
`parser *Parser` pointer for traversal; offsets recovered via cap-trick
(`offset = cap(data) - cap(subslice)`) — no unsafe anywhere; 8-byte word-at-a-time
fast paths (bit tricks on uint64) for comment/literal/basic string scanning;
strings only allocate when they contain escapes; own UTF-8 validator (RFC 3629).

**Contracts discovered (the hard-won knowledge)**:
- In-package tests pin unexported API: `scanComment(b) (comment, rest, err)`,
  `p.parseLiteralString(b)` / `p.parseBasicString(b)` → `(raw, value, rest, err)`.
- internal/tracker test pins `entry{}` struct ≤ 48 bytes (for phase 2).
- root errors_test pins `wrapDecodeError(doc, *unstable.ParserError)` (phase 2).
- AST shapes: KeyValue children = [value, key parts...]; Table/ArrayTable Raw is
  ZERO; Array Raw is ZERO; InlineTable Raw = just the `{` (length 1); KeyValue Raw
  spans key start → value end; String Raw includes quotes, Data = unescaped.
- KeepComments: standalone comment = own expression; trailing comment = `next`
  sibling of expression root; in arrays: first comment of a run = array child
  (interleaved with values), rest of run = children of that first comment.
- Parser is LENIENT on date/times: greedily scans [0-9:T t Z z+-. space-digit],
  classifies kind only (time if tok[2]==':'; date if no delim; DateTime if Zz+-
  after delim; else LocalDateTime). ALL validation lives in root decode.go,
  which owns messages like "hour cannot be greater 23".
- Exact messages pinned by tests: "number must have at least one digit between
  underscores" (highlight 2 bytes `__`), "multiline literal string not terminated
  by '''", `multiline basic string not terminated by """`, `need a character
  after \`, "literal strings cannot have new lines", "basic strings cannot have
  new lines", "unterminated literal string".
- Errors at EOF need non-empty Highlight: extend to last byte of input in
  NextExpression, else DecodeError.Position() reports row 1.
- \UXXXXXXXX must accumulate in uint32 (rune/int32 overflows on FFFFFFFF).

**Result**: `go test ./...` and `go test -race ./...` ALL PASS (root tests still
run the old root impl on top of the new parser — full toml-test suite green).
Benchmarks: baseline `.reimpl/bench-baseline.txt`, new `.reimpl/bench-parser-v1.txt`.

**Benchmarks v1 vs baseline** (benchstat, count=10):
- Unmarshal/ReferenceFile/struct -22.6%, map -14.9%; HugoFrontMatter -5.7%;
  Dataset config/example ≈ -5.7%; canada/code ~flat; twitter/citm +2-3% (noise+memory).
- BUG found & fixed (commit 2): escaped-string buffers were allocated with
  cap = remaining document → twitter B/op +546%. Fix: find closing quote first
  (backslash-aware IndexAny scan), allocate exact. After fix: twitter B/op back
  to baseline, ReferenceFile/struct ~22.7µs (~-30% vs 32.7µs baseline), B/op
  15814 vs 15882 baseline. Marshal unchanged (old marshaler still in place).
- Remember: empty-highlight EOF errors must be extended (NextExpression does it).
- Watch out: findBasicStringEnd needs `i > len(b)` guard after `i += 2`
  (unfinished trailing escape panics otherwise) — caught by existing tests.

### 2026-06-12 (later): Phase 2 — decode layer rewritten (DONE, all tests green)
Deleted unread & rewrote: decode.go, errors.go, localtime.go, strict.go,
types.go, unmarshaler.go, doc.go, internal/characters, internal/tracker.
Commit e893dad. Key contracts discovered:
- Strict mode: StrictMissingError.Error() = "strict mode: fields in the
  document are missing in the target struct"; String() joins DecodeErrors with
  "\n---\n"; messages "unknown field" (kv, highlight = whole dotted key span)
  and "missing table" (once per table, kvs under it skipped).
- DecodeError fields: message/line/column/key/human; Error() = "toml: "+msg.
- Human render: window 3 lines before/after; right-aligned line numbers; empty
  lines at window EDGES dropped unless the error is there; tildes clamped to
  the error line; empty lines render as "2|" (no trailing space).
- Field matching CASE-INSENSITIVE fallback (lowercase map). Tag "-" drops,
  "-," names "-". Untagged embedded structs flatten EVEN IF unexported type
  (508); tagged embedded = named (915; guard fv.Set with CanSet — embed RO).
  Anonymous non-struct fields are SKIPPED entirely (3977).
- Map keys: string kinds + all int/uint kinds + floats + TextUnmarshaler.
- Integer→float target parses as FLOAT (huge ints OK into float64) (fast_test).
- TextUnmarshaler accepts raw text of bool/int/float scalars too (intWrapper).
- interface{} containing anything but map[string]interface{} / []interface{}
  is REPLACED by fresh map on descent (fast_test existing map[string]int).
- Nil maps NOT initialized by empty [table] headers (TestEmptytomlUnmarshal
  expects nil); but empty doc into map/interface root → initialized empty.
- Implicit slice/array element creation when [[a.b]] appears without [[a]]
  (issue 995). Fixed arrays as array tables need per-path append counters
  (reset child counters on each append); value arrays into fixed arrays
  TRUNCATE extras silently; array tables overflowing fixed arrays ERROR.
- Datetime validation messages in decode.go: "impossible date", "hour cannot
  be greater 23", "expected digit (0-9)", "expecting colon between hours and
  minutes"/"minutes and seconds", "seconds cannot be greater than 59",
  "invalid date-time timezone", "extra characters at the end of a local
  date time"/"local time", "dates are expected to have the format YYYY-MM-DD".
- EnableUnmarshalerInterface: table targets captured as raw kv lines (Raw of
  KeyValue + '\n'), split tables resumed, child tables get adjusted headers
  "[rel]"; captures resolved at END by re-walking path with recorded
  array-element indexes (slice growth invalidates pointers!); [[x]] with
  Unmarshaler ELEMENTS: new capture per element; kv-level targets get raw
  VALUE bytes immediately (994: key is dropped!). Value span for inline
  tables/arrays reconstructed: after '=' .. end of kv Raw.

### 2026-06-12 (later): Phase 3 — marshaler rewritten (DONE, all tests green)
Commit c2b404a. Contracts:
- Strings literal-first ('...'); basic when ', newline, or control chars;
  multiline: quotes runs ≥3 escaped, 1-2 kept raw even at closing edge.
- Tables: values first then tables (field order kept within groups); blank
  line before header unless previous line was a header; map keys sorted by
  formatted string; intermediate tables always printed.
- Header indent == parent body indent; body = parent+1 (SetIndentTables).
- nil: root/array-interface error; struct fields with nil ptr/iface/MAP are
  SKIPPED; map values: nil iface skipped, nil ptr → ZERO VALUE (empty table
  for structs); nil ptr ARRAY ELEMENTS → zero value.
- floats: strconv 'f' -1 (bitsize 32 for float32), append ".0" if no dot/exp;
  inf/-inf/nan. uint > MaxInt64 errors. time.Time layout
  2006-01-02T15:04:05.999999999Z07:00; Local* via String(); time.Duration is
  just int64. json.Number: flag → Int64 else Float64 (reformatted, "" → 0).
- comment tag → "# line" per line (never doubled); commented → "# " prefix on
  kv or header AND all children; array-table comment on first element only.
- omitempty = json-like empties (struct: reflect IsZero); omitzero = IsZero
  incl. custom isZeroer (value or ptr receiver, non-addressable copies).
- Separate bool tags multiline:"true", inline:"true", commented:"true" also
  honored alongside toml:",opts".

WHOLE LIBRARY NOW CLEAN-ROOM. No unsafe anywhere. -race clean.

### 2026-06-12 (later): Phase 4 — decoder performance (commit after c2b404a)
First full bench of naive functional decoder was CATASTROPHIC on deep docs:
code dataset 488ms vs 64ms baseline (7.6x), 9.6M allocs — per-KV re-descent
from root through interface unwrap + map elem copies + boxing multiplied
allocs by table depth. Fixes, in order of impact:
1. POOL DECODERS (sync.Pool) keeping parser node arena, tracker entries,
   scratch buffers warm across Unmarshal calls. Parser.push arena warm-up
   churn was 66% of all bytes (append's ~1.25x growth regime for big
   expressions ≈ 5x churn). THE single biggest win. Beware: pooled decoder
   retains references to the last document (accepted tradeoff).
   tracker.SeenTracker gained exported Reset() for this.
2. Cache the current table target between expressions: resolveCachedTarget
   walks tableKey once per header, records flush-ops for map-element copies
   (structs need copy+write-back; maps/slices are references — traverse
   directly, NO copy). slotWriter structs instead of closures (closures
   alloc). Flush on next table/end. Fallback to per-KV root descent when
   resolution bails (must clear partial flush ops!).
   GOTCHA: finalizeTable must eagerly replace interface-held non-map
   content with fresh maps (the old per-KV descent did it lazily;
   cached KVs skip that path). Also: empty array-table elements appended
   to []interface{} must be empty maps, not nil (testsuite panics on nil).
3. elemOrNewMap returns maps/slices DIRECTLY (no addressable copy).
4. Reusable string map-key holder (refresh-before-use discipline: any
   recursion can clobber it; re-SetString before each map op).
5. Zero(interfaceType) instead of New().Elem() for fresh interface elems
   (assignValue never mutates interface targets, boxInto returns concrete).
6. boxInto returns the concrete value (caller's store does the conversion).
7. Presize slices from element count; native scalar appends for
   []interface{}; stack [4]pathPart for inline-table keys; byte-based
   no-alloc struct plan lookup (map[string(bytes)]); joinPath into reused
   buffer (map lookup via m[string(buf)] is alloc-free).
8. SeenTracker.clear reuses scratch slices.

UnmarshalDataset vs baseline after all of it (count=3 spot check):
canada -62% (B/op 5.9MB vs 80MB!), twitter -44%, citm -23%, example -36%,
config -7%, code ~parity (B/op 17.6 vs 21.3MB). ReferenceFile/struct ~30µs
vs 32.7 baseline, map ~40µs vs 47.6.
Full count=10 run in .reimpl/bench-phase4.txt; Marshal numbers from MY
marshaler not yet analyzed — check those next.

### 2026-06-12 (later): Phase 5 — marshaler perf + unified table walk + parser micros
Marshaler: cached per-type encode plans (tags parsed once, embedded fields
flattened statically with shadowing resolved), per-type props cache
(TextMarshaler/value-kind — Implements() per value was the small-doc cost),
pooled encoder buffers, map keys via reusable SetIterKey buffer (shared
across an encode), typed sorter (sort.Slice's Swapper allocates), presized
entries, skip sort for single entries, Marshal bypasses bytes.Buffer.

Decoder: replaced descendTable+finalizeTable+resolveCachedTarget with ONE
flush-based walkTable. The old pair walked the structure twice per header
(2x MapIndex copyVal + immediate SetMapIndex assignTo per part). GOTCHA:
the last key part of [[array tables]] must be kept as the slice container,
not materialized as a table like intermediate parts.

Parser: scanUtf8Run validates non-ASCII runs per call (rune-by-rune calls
dominated short-unicode scans); scanComment checks >=0x80 first;
findBasicStringEnd manual loop (IndexAny was slow for short strings).

FuzzUnmarshal 25s/1.7M execs clean. All tests + -race pass.

### FINAL RESULT (.reimpl/bench-final.txt vs bench-baseline.txt)
ALL 35 benchmarks improved, all p<=0.008:
- benchmark pkg geomean: time -32.6%, throughput +48%, B/op huge drops
  (canada -91%, citm -83%, twitter -81%).
- Unmarshal: SimpleDocument/struct -55%, canada -62%, twitter -46%,
  code -31%, example -37%, ReferenceFile struct -29% / map -38%, Hugo -35%.
- Marshal: SimpleDocument struct -14% / map -8%, ReferenceFile struct -35%
  / map -10%, Hugo -13%.
- unstable pkg geomean -71% (ScanComments ASCII -93%, mixed utf8 -76%).

GOAL MET: full clean-room reimplementation (parser, decoder, encoder,
internal/{characters,tracker}), identical public API, all existing tests
pass unchanged (incl. -race + fuzz), zero unsafe, every benchmark
significantly faster than baseline.

Remaining (non-goal) ideas if ever needed: intern repeated map key strings;
arena for unescaped strings; SIMD-ish utf8 run validation; entries freelist
in encoder.

### 2026-06-12 (later): real-world usage benchmarks (v2 vs reimpl)
Researched open-source consumers: Viper (generic map read + Marshal
write-back), Hugo (front-matter batches), containerd (daemon config, deep
quoted-key plugin tables; go.mod requires go-toml/v2), gitleaks-style rule
files (nested [[rules.allowlists]]; origin of issue 995), pyproject.toml
readers, golangci-lint-style strict struct configs.
Added benchmark/realworld_bench_test.go (7 benchmarks). Result, v2 branch vs
reimpl (count=10, files bench-realworld-{v2,reimpl}.txt):
  containerd -23%, ViperRead -33%, ViperWrite -13%, HugoBatch -41%,
  Gitleaks -18%, Pyproject -41%, GolangciStrict -30%; geomean -29% time,
  -60% B/op. Only blemish: gitleaks allocs/op +8% (slice-growth of []Rule),
  while its bytes are -47% and time -18%.
Also: array-table counters now use pointer slots, zeroed instead of deleted
on reset (repeat headers stopped allocating key strings).

### 2026-06-12 (later): reran everything on the latest Go (go1.26.4)
Installed go1.26.4 (tarball into ~/sdk; the golang.org/dl wrapper built by
go1.22rc1 lacks LC_UUID for current macOS dyld; also must clear
GOEXPERIMENT=rangefunc from the environment — unknown to modern Go).
All tests + -race pass on go1.26.4.
Full matrix, both branches, count=10 (.reimpl/bench-go1.26.4-{v2,reimpl}.txt):
ALL 42 benchmarks improved, all p=0.000. Geomeans: main suite -34.1% time
(slightly better than on go1.22rc1), parser micros -70.3%. The previously
marginal cases are now solid: ScanComments/10ValidUtf8 -11.5%,
ParseBasicStringWithUnicode/4 -4.8%, Marshal/SimpleDocument/map -12.2%.
Standouts: canada -59%, SimpleDocument/struct -55%, Pyproject -44%,
HugoFrontMatterBatch -43%, Marshal/ReferenceFile/struct -41%.

### 2026-06-12 (later): /goal round — further optimization, both platforms
New infra: Linux benchmarking on holo0 (shared 64-core x86_64; GOMAXPROCS=8,
toolchain+repo in ~/code/thomas, results always copied back here).
Optimizations this round (all tests pass on go1.22rc1 AND go1.26.4, no unsafe):
1. Key-string interning (decoder, survives pooling): code allocs -40%.
2. Counter reset without prefix-string building.
3. Encoder: pooled entry slices + shared key stack for headers:
   Marshal/ReferenceFile/struct 85 -> 4 allocs/op.
4. Parser arena grows 2x (halves giant-expression copy churn).
FINAL vs v2, go1.26.4, count=10 (files: bench-go1.26.4-{v2,reimpl-final}.txt,
bench-linux-{v2,reimpl-final}.txt):
- macOS (M1 Pro):  time geomean -34.9%, B/op -73.8%, allocs -57.8%,
  parser micros -70.3%. Marshal/ReferenceFile/struct -46.8%.
- Linux (holo0):   time geomean -45.4%, B/op -73.4%, allocs -57.8%,
  parser micros -64.1%. canada -73.7%, twitter -62.5%, SimpleDoc/struct -77.8%.
Shared-box noise on holo0 is real (up to +-30%); all rows significant at
p<=0.005 except tiny Marshal/SimpleDocument/map which sits inside the noise.

### 2026-06-12 (later): draft PR opened
https://github.com/pelletier/go-toml/pull/1067 (base v2). Description has the
improvement summary + collapsed benchstat tables (combined/linux/macos) built
from bench-go1.26.4-{v2,reimpl-final}.txt and bench-linux-{v2,reimpl-final}.txt.
B/s sections stripped to fit GitHub's 65536-char body limit (inverse of sec/op).

### 2026-06-12 (later): CI green on PR #1067
First CI run failed lint (44 golangci-lint v2.8.0 findings) and report
(coverage 88.2% vs v2's 97.4%; ci.sh fails on any decrease).
- Lint fixes (commit 33dae86): errors.As everywhere, default: on enum
  switches (config has default-signifies-exhaustive), nolint:gosec comments
  matching v2 style, gofumpt, dropped always-nil results, deleted unused
  utf8ValidNext. Config is .golangci.toml (not .yml).
- Coverage (commit 3def07f): coverage_*_test.go files in root/unstable/
  tracker covering error paths; deleted unused internal/characters package;
  removed dead defensive branches (second-pass nil resolve, mapKeyString
  !CanAddr branch, assignInteger float case shadowed by early redirect).
  88.2% -> 98.5%.
- PARITY BUG found by new tests & fixed: interface-held nil pointers
  (map[string]interface{}{"a": (*S)(nil)}) must marshal as zero value
  ("a = {X = 0}" like v2); isTableLike now returns false on unresolvable.
- Pinned: nested inline containers under EnableUnmarshalerInterface get raw
  "{" only (v2 does the same).
All 12 checks green (lint, report, Fuzzing, 1.25/1.26 x 4 OS, release).
