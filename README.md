# go-toml V2

Development branch. Use at your own risk.

[👉 Discussion on github](https://github.com/pelletier/go-toml/discussions/471).

* `toml.Unmarshal()` should work as well as v1.

## Must do

### Unmarshal

- [x] Unmarshal into maps.
- [x] Support Array Tables.
- [x] Unmarshal into pointers.
- [x] Support Date / times.
- [x] Support struct tags annotations.
- [x] Support Arrays.
- [x] Support Unmarshaler interface.
- [x] Original go-toml unmarshal tests pass.
- [x] Benchmark!
- [x] Abstract AST.
- [x] Original go-toml testgen tests pass.
- [x] Track file position (line, column) for errors.
- [ ] Strict mode.

### Marshal

- [ ] Minimal implementation

### Document

- [ ] Gather requirements and design API.

## Ideas

- [ ] Allow types to implement a `ASTUnmarshaler` interface to unmarshal
      straight from the AST?
- [x] Rewrite AST to use a single array as storage instead of one allocation per
      node.
- [ ] Provide "minimal allocations" option that uses `unsafe` to reuse the input
      byte array as storage for strings.
- [x] Cache reflection operations per type.
- [ ] Optimize tracker pass.

## Differences with v1

* [unmarshal](https://github.com/pelletier/go-toml/discussions/488)

## License

The MIT License (MIT). Read [LICENSE](LICENSE).
