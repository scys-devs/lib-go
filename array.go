package lib

import (
	"sort"

	"golang.org/x/exp/constraints"
)

func Index[T constraints.Ordered](ll []T, n T) int {
	for i, l := range ll {
		if l == n {
			return i
		}
	}
	return -1
}

func Chunk[T any](ll []T, size int) [][]T {
	output := make([][]T, 0)
	for i := 0; i < len(ll); i += size {
		end := i + size
		if end > len(ll) {
			end = len(ll)
		}
		output = append(output, ll[i:end])
	}
	return output
}

func ChunkWith[T any](ll []T, size int, with func(chunk []T)) {
	for i := 0; i < len(ll); i += size {
		end := i + size
		if end > len(ll) {
			end = len(ll)
		}
		with(ll[i:end])
	}
}

func Map[T any, M any](ll []T, action func(item T) M) []M {
	output := make([]M, 0, len(ll))
	for _, item := range ll {
		output = append(output, action(item))
	}
	return output
}

func Filter[T any](ll []T, check func(item T) bool) []T {
	output := make([]T, 0)
	for _, item := range ll {
		if check(item) {
			output = append(output, item)
		}
	}
	return output
}

func FindIndex[T any](ll []T, check func(item T) bool) int {
	for i, item := range ll {
		if check(item) {
			return i
		}
	}
	return -1
}

func Group[T any, M constraints.Ordered](ll []T, key func(item T) M) (output map[M]T) {
	output = make(map[M]T, len(ll))
	for _, item := range ll {
		output[key(item)] = item
	}
	return
}

func GroupArr[T any, M constraints.Ordered](ll []T, key func(item T) M) (output map[M][]T) {
	output = make(map[M][]T)
	for _, item := range ll {
		_key := key(item)
		if _, ok := output[_key]; !ok {
			output[_key] = make([]T, 0)
		}
		output[_key] = append(output[_key], item)
	}
	return
}

func Sort[T any](ll []T, key ...func(T) any) {
	sort.SliceStable(ll, func(i, j int) (flag bool) {
		for _, k := range key {
			vv1 := k(ll[i])
			vv2 := k(ll[j])
			if vv1 == vv2 { // 如果相当的话，就continue
				continue
			}

			switch vv1.(type) {
			case int:
				return vv1.(int) < vv2.(int)
			case int64:
				return vv1.(int64) < vv2.(int64)
			case float64:
				return vv1.(float64) < vv2.(float64)
			case string:
				return vv1.(string) < vv2.(string)
			default:
				panic("invalid type")
			}
		}
		return
	})
}

func Max[T constraints.Ordered](a ...T) T {
	base := a[0]
	for i := 1; i < len(a); i++ {
		if base < a[i] {
			base = a[i]
		}
	}
	return base
}

func Min[T constraints.Ordered](a ...T) T {
	base := a[0]
	for i := 1; i < len(a); i++ {
		if base > a[i] {
			base = a[i]
		}
	}
	return base
}

// a-b的差集
func Difference[T constraints.Ordered](a, b []T) (ll []T) {
	m := make(map[T]struct{})

	for _, v := range b {
		m[v] = struct{}{}
	}

	for _, v := range a {
		if _, ok := m[v]; !ok { // A中有 B中没有
			ll = append(ll, v)
		}
	}
	return
}

// Intersection a与b的交集
func Intersection[T constraints.Ordered](a, b []T) (ll []T) {
	m := make(map[T]struct{})

	for _, v := range b {
		m[v] = struct{}{}
	}

	for _, v := range a {
		if _, ok := m[v]; ok { // A中有 B中也有
			ll = append(ll, v)
		}
	}
	return
}

func Contains[T comparable](elems []T, v T) bool {
	for _, s := range elems {
		if v == s {
			return true
		}
	}
	return false
}

func Reduce[T, M any](s []T, f func(M, T) M, initValue M) M {
	acc := initValue
	for _, v := range s {
		acc = f(acc, v)
	}
	return acc
}
