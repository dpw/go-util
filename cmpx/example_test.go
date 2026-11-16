package cmpx_test

import (
	"fmt"
	"slices"

	"github.com/dpw/go-util/cmpx"
)

func ExampleBy() {
	type Person struct {
		Name string
		Age  int
	}

	people := []Person{{"Bob", 30}, {"Alice", 25}}
	slices.SortFunc(people, cmpx.By(func(p Person) int { return p.Age }))
	fmt.Println(people)
	// Output: [{Alice 25} {Bob 30}]
}
