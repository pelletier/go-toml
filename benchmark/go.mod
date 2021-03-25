module github.com/pelletier/go-toml/v2/benchmark

go 1.16

replace github.com/pelletier/go-toml/v2 => ../

replace github.com/pelletier/go-toml-v1 => /home/thomas/src/github.com/pelletier/go-toml-v1

require (
	github.com/BurntSushi/toml v0.3.1
	github.com/pelletier/go-toml v1.8.1 // indirect
	github.com/pelletier/go-toml-v1 v0.0.0-00010101000000-000000000000
	github.com/pelletier/go-toml/v2 v2.0.0-00010101000000-000000000000
	github.com/stretchr/testify v1.7.0
)
