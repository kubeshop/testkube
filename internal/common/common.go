package common

import "reflect"

// MergeMaps merges multiple maps into one, the later ones takes precedence over the first ones
func MergeMaps(ms ...map[string]string) map[string]string {
	res := map[string]string{}
	for _, m := range ms {
		for k, v := range m {
			res[k] = v
		}
	}
	return res
}

func Always[T any](_ T) bool {
	return true
}

func Never[T any](_ T) bool {
	return false
}

func DeepEqualCmp[T any](v1 T) func(T) bool {
	return func(v2 T) bool {
		return reflect.DeepEqual(v1, v2)
	}
}

func Ptr[T any](v T) *T {
	return &v
}

func MapPtr[T any, U any](v *T, fn func(T) U) *U {
	if v == nil {
		return nil
	}
	return Ptr(fn(*v))
}

func PtrOrNil[T any](v T) *T {
	if reflect.ValueOf(v).IsZero() {
		return nil
	}
	return &v
}

func ResolvePtr[T any](v *T, def T) T {
	if v == nil {
		return def
	}
	return *v
}

func MapEnumToString[T ~string](v T) string {
	return string(v)
}

func MapStringToEnum[T ~string](v string) T {
	return T(v)
}

func MapSlice[T any, U any](s []T, fn func(T) U) []U {
	if len(s) == 0 {
		return nil
	}
	result := make([]U, len(s))
	for i := range s {
		result[i] = fn(s[i])
	}
	return result
}

func FilterSlice[T any](s []T, fn func(T) bool) []T {
	if len(s) == 0 {
		return nil
	}
	result := make([]T, 0)
	for i := range s {
		if fn(s[i]) {
			result = append(result, s[i])
		}
	}
	return result
}

func UniqueSlice[T comparable](s []T) []T {
	if len(s) == 0 {
		return nil
	}
	result := make([]T, 0)
	seen := map[T]struct{}{}
	for i := range s {
		_, ok := seen[s[i]]
		if !ok {
			seen[s[i]] = struct{}{}
			result = append(result, s[i])
		}
	}
	return result
}

func MapMap[T any, U any](m map[string]T, fn func(T) U) map[string]U {
	if len(m) == 0 {
		return nil
	}
	res := make(map[string]U, len(m))
	for k, v := range m {
		res[k] = fn(v)
	}
	return res
}

func GetMapValue[T any, K comparable](m map[K]T, k K, def T) T {
	v, ok := m[k]
	if ok {
		return v
	}
	return def
}

func GetOr(v ...string) string {
	for i := range v {
		if v[i] != "" {
			return v[i]
		}
	}
	return ""
}
