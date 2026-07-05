package edit_test

import (
	"fmt"

	"github.com/pelletier/go-toml/v2/unstable/edit"
)

func Example() {
	doc := `# my app config

title = 'My App'
debug = false # do not touch

[database]
host = 'localhost'
`

	d, err := edit.Parse([]byte(doc))
	if err != nil {
		panic(err)
	}

	// Replacing a value only rewrites the value itself: the comment trailing
	// it stays.
	if err := d.Set([]string{"debug"}, true); err != nil {
		panic(err)
	}
	// New keys are appended to the table they belong to.
	if err := d.Set([]string{"database", "port"}, 5432); err != nil {
		panic(err)
	}
	d.Delete([]string{"title"})

	fmt.Print(d.String())
	// Output:
	// # my app config
	//
	// debug = true # do not touch
	//
	// [database]
	// host = 'localhost'
	// port = 5432
}

func ExampleDocument_Set() {
	d, err := edit.Parse(nil)
	if err != nil {
		panic(err)
	}
	if err := d.Set([]string{"title"}, "example"); err != nil {
		panic(err)
	}
	// Missing tables are created with a single header.
	if err := d.Set([]string{"servers", "alpha", "ip"}, "10.0.0.1"); err != nil {
		panic(err)
	}
	fmt.Print(d.String())
	// Output:
	// title = 'example'
	//
	// [servers.alpha]
	// ip = '10.0.0.1'
}
