//go:build !solution

package hotelbusiness

import "sort"

type Guest struct {
	CheckInDate  int
	CheckOutDate int
}

type Load struct {
	StartDate  int
	GuestCount int
}

func ComputeLoad(guests []Guest) []Load {
	dateCount := make(map[int]int)
	for _, guest := range guests {
		dateCount[guest.CheckInDate]++
		dateCount[guest.CheckOutDate]--
	}

	dates := make([]int, 0, len(dateCount))
	for date := range dateCount {
		dates = append(dates, date)
	}
	sort.Ints(dates)

	count := 0
	loads := make([]Load, 0)
	for _, date := range dates {
		if dateCount[date] != 0 {
			count += dateCount[date]
			loads = append(loads, Load{StartDate: date, GuestCount: count})
		}
	}

	return loads
}
