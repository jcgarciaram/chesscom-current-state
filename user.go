package main

type userStats struct {
	User          string
	Wins          int
	Losses        int
	Draws         int
	Points        float64
	WinPercentage float64
	WinStreak     int
}

type userStatsByWinPercDesc []userStats

func (a userStatsByWinPercDesc) Len() int           { return len(a) }
func (a userStatsByWinPercDesc) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a userStatsByWinPercDesc) Less(i, j int) bool { return a[i].WinPercentage > a[j].WinPercentage }
