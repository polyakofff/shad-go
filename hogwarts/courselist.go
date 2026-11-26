//go:build !solution

package hogwarts

func Dfs(v string, prereqs map[string][]string, order []string, color map[string]int) []string {
	color[v] = 1
	for _, prereq := range prereqs[v] {
		if color[prereq] == 1 {
			panic("Cycle!")
		}
		if color[prereq] == 0 {
			order = Dfs(prereq, prereqs, order, color)
		}
	}
	color[v] = 2
	return append(order, v)
}

func GetCourseList(prereqs map[string][]string) []string {
	// a: b c
	// b: d
	// => d b c a

	order := make([]string, 0)
	color := make(map[string]int)

	for course := range prereqs {
		if color[course] == 0 {
			order = Dfs(course, prereqs, order, color)
		}
	}
	return order
}
