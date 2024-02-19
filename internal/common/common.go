package common

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

func Ptr[T any](v T) *T {
	return &v
}

func MapPtr[T any, U any](v *T, fn func(T) U) *U {
	if v == nil {
		return nil
	}
	return Ptr(fn(*v))
}

func PtrOrNil[T comparable](v T) *T {
	var zero T
	if zero == v {
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
