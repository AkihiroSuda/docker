package master

func chunkStrings(x []string, sz int) [][]string {
	var result [][]string
	for i := 0; i < len(x); i += sz {
		lb, ub := i, i+sz
		if ub > len(x) {
			ub = len(x)
		}
		chunk := x[lb:ub]
		result = append(result, chunk)
	}
	return result
}
