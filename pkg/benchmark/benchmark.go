package benchmark

import (
	"encoding/csv"
	"fmt"
	"os"
	"time"
)

type span struct {
	start time.Time
	end   time.Time
}

var timers = make(map[string]span)

func StartTimer(name string) {
	timers[name] = span{start: time.Now()}
}

func StopTimer(name string) {
	if _, exists := timers[name]; !exists {
		fmt.Printf("Timer %s doesn't exist\n", name)
		return
	}
	s := timers[name]
	timers[name] = span{start: s.start, end: time.Now()}
}

func PrintTimers() {
	fmt.Printf("%-10s %-20s\n", "Name", "Duration")
	for name, timer := range timers {
		fmt.Printf("%-10s %-20v\n", name, timer.end.Sub(timer.start))
	}
}

func ExportToCSV() {
	file, err := os.Create("timers.csv")
	if err != nil {
		fmt.Println("Cannot create file", err)
		return
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	for name, timer := range timers {
		writer.Write([]string{name, timer.end.Sub(timer.start).String()})
	}
}
