package main

func getPreviousMonth(year, month int) (int, int) {
	previousMonth := month
	previousYear := year
	if month == 1 {
		previousMonth = 12
		previousYear--
	} else {
		previousMonth--
	}

	return previousYear, previousMonth
}
