package iterx_test

import (
	"fmt"
	"slices"

	"github.com/dpw/go-util/iterx"
)

func ExampleFilter() {
	nums := slices.Values([]int{1, 2, 3, 4})
	evens := iterx.Filter(nums, func(n int) bool { return n%2 == 0 })
	for n := range evens {
		fmt.Println(n)
	}
	// Output:
	// 2
	// 4
}
