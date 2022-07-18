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
