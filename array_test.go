package lib

import (
	"fmt"
	"testing"
)

func TestChunk(t *testing.T) {
	l1 := []string{"1", "2", "3", "4", "5", "6"}
	fmt.Println(Chunk(l1, 2))
	l2 := []int64{1, 2, 3, 4, 5}
	fmt.Println(Chunk(l2, 2))
}

func TestSort(t *testing.T) {
	type User struct {
		ID   int
		Name string
	}
	var ll = []User{
		{ID: 2, Name: "bbb"},
		{ID: 3, Name: "ccc"},
		{ID: 2, Name: "aaa"},
		{ID: 5, Name: "aaa"},
		{ID: 4, Name: "ccc2"},
		{ID: 1, Name: "bbb"},
	}
	Sort(ll,
		func(t User) any {
			return t.ID
		},
		func(t User) any {
			return t.Name
		},

	)
	fmt.Println(ll)
}
