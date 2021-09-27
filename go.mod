module github.com/pelletier/go-toml/v2

go 1.16

require (
	github.com/BurntSushi/toml-test v1.0.1
	// latest (v1.7.0) doesn't have the fix for time.Time
	github.com/stretchr/testify v1.7.1-0.20210427113832-6241f9ab9942
)
