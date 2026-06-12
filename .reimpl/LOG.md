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

### Next: performance phase
- bench-baseline.txt = original; bench-phase3.txt = full reimpl (running).
- Ideas if needed: avoid per-expression table re-descent (cache resolved
  table target between expressions when safe); reduce allocs in strings
  (chunked arena); faster Unicode scanning; encoder buffer reuse.
