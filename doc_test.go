// code examples for godoc

package toml

import "fmt"

func ExampleNodeFilterFn_filterExample() {
	tree, _ := Load(`
      [struct_one]
      foo = "foo"
      bar = "bar"

      [struct_two]
      baz = "baz"
      gorf = "gorf"
    `)

	// create a query that references a user-defined-filter
	query, _ := CompileQuery("$[?(bazOnly)]")

	// define the filter, and assign it to the query
	query.SetFilter("bazOnly", func(node interface{}) bool {
		if tree, ok := node.(*TomlTree); ok {
			return tree.Has("baz")
		}
		return false // reject all other node types
	})

	// results contain only the 'struct_two' TomlTree
	query.Execute(tree)
}

func ExampleQuery_queryExample() {
	config, _ := Load(`
      [[book]]
      title = "The Stand"
      author = "Stephen King"
      [[book]]
      title = "For Whom the Bell Tolls"
      author = "Ernest Hemmingway"
      [[book]]
      title = "Neuromancer"
      author = "William Gibson"
    `)

	// find and print all the authors in the document
	authors, _ := config.Query("$.book.author")
	for _, name := range authors.Values() {
		fmt.Println(name)
	}
}
