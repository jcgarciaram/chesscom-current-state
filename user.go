package main

type userStats struct {
	User   string
	Wins   int
	Losses int
	Draws  int
	Points float64
}

type userStatsByPointsDesc []userStats

func (a userStatsByPointsDesc) Len() int           { return len(a) }
func (a userStatsByPointsDesc) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a userStatsByPointsDesc) Less(i, j int) bool { return a[i].Points > a[j].Points }
