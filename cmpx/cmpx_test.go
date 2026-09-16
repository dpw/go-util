package cmpx

import (
	"slices"
	"testing"

	"github.com/alecthomas/assert/v2"
)

type person struct {
	name string
	age  int
}

func TestBy_ComparatorResult(t *testing.T) {
	byAge := By(func(p person) int { return p.age })

	assert.True(t, byAge(person{age: 1}, person{age: 2}) < 0)
	assert.True(t, byAge(person{age: 2}, person{age: 1}) > 0)
	assert.Zero(t, byAge(person{name: "a", age: 5}, person{name: "b", age: 5}))
}

func TestBy_SortIntKey(t *testing.T) {
	people := []person{
		{"alice", 30},
		{"bob", 25},
		{"carol", 40},
	}
	slices.SortFunc(people, By(func(p person) int { return p.age }))

	assert.Equal(t, []person{
		{"bob", 25},
		{"alice", 30},
		{"carol", 40},
	}, people)
}

func TestBy_SortStringKey(t *testing.T) {
	people := []person{
		{"carol", 40},
		{"alice", 30},
		{"bob", 25},
	}
	slices.SortFunc(people, By(func(p person) string { return p.name }))

	assert.Equal(t, []string{"alice", "bob", "carol"},
		[]string{people[0].name, people[1].name, people[2].name})
}

// SortFunc is not guaranteed stable, so By only defines the order of
// elements with distinct keys; equal keys may appear in any order.
func TestBy_IsSortedFunc(t *testing.T) {
	nums := []int{-3, 0, 1, 7, 42}
	assert.True(t, slices.IsSortedFunc(nums, By(func(n int) int { return n })))
}
