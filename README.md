# go-toml V2

Development branch. Probably does not work.

[👉 Discussion on github](https://github.com/pelletier/go-toml/discussions/471).

## Must do

- [x] Unmarshal into maps.
- [ ] Attach comments to AST (gated by parser flag).
- [ ] Abstract AST.
- [ ] Support Array Tables
- [ ] Support Date / times.
- [ ] Support Unmarshaler interface.
- [ ] Support struct tags annotations.
- [ ] Benchmark!

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
