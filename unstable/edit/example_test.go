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

func ExampleDocument_Set_arraysAndInline() {
	doc := `[[servers]]
ip = '10.0.0.1'
ports = [8080]

[[servers]]
ip = '10.0.0.2'
ports = [8080]
`
	d, err := edit.Parse([]byte(doc))
	if err != nil {
		panic(err)
	}

	// Array elements (of tables or not) are addressed by decimal indexes.
	if err := d.Set([]string{"servers", "1", "ip"}, "10.0.0.3"); err != nil {
		panic(err)
	}
	// An index equal to the length appends: a port to the first server...
	if err := d.Set([]string{"servers", "0", "ports", "1"}, 8081); err != nil {
		panic(err)
	}
	// ...or a whole new [[servers]] element.
	if err := d.Set([]string{"servers", "2", "ip"}, "10.0.0.4"); err != nil {
		panic(err)
	}

	fmt.Print(d.String())
	// Output:
	// [[servers]]
	// ip = '10.0.0.1'
	// ports = [8080, 8081]
	//
	// [[servers]]
	// ip = '10.0.0.3'
	// ports = [8080]
	//
	// [[servers]]
	// ip = '10.0.0.4'
}

func ExampleDocument_SetComment() {
	d, err := edit.Parse([]byte("[server]\nport = 8080\n"))
	if err != nil {
		panic(err)
	}
	if err := d.SetComment([]string{"server"}, "Connection settings."); err != nil {
		panic(err)
	}
	if err := d.SetTrailingComment([]string{"server", "port"}, "default"); err != nil {
		panic(err)
	}
	fmt.Print(d.String())
	// Output:
	// # Connection settings.
	// [server]
	// port = 8080 # default
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
