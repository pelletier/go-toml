# go-toml V2

Development branch. Probably does not work.

[👉 Discussion on github](https://github.com/pelletier/go-toml/discussions/471).

## Must do

- [x] Unmarshal into maps.
- [x] Support Array Tables.
- [ ] Unmarshal into pointers.  
  > Was supposed to be done, but seems like there are still some assignation
  > issues.
- [x] Support Date / times.
- [ ] Support Unmarshaler interface.
- [x] Support struct tags annotations.
- [ ] Original go-toml unmarshal tests pass.
- [ ] Benchmark!
- [ ] Abstract AST.
- [ ] Attach comments to AST (gated by parser flag).
- [ ] Track file position (line, column) for errors.
- [ ] Benchmark again!

## Further work

- [ ] Rewrite AST to use a single array as storage instead of one allocation per
      node.
- [ ] Provide "minimal allocations" option that uses `unsafe` to reuse the input
      byte array as storage for strings.

## Ideas

- [ ] Allow types to implement a `ASTUnmarshaler` interface to unmarshal
      straight from the AST?

## License

The MIT License (MIT). Read [LICENSE](LICENSE).
