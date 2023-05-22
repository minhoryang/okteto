package benchmark

import (
	"encoding/csv"
	"fmt"
	"os"
	"sort"
	"sync"
	"time"
)

type span struct {
	start time.Time
	end   time.Time
}

var timers = make(map[string]span)
var mutex = &sync.Mutex{} // added mutex for synchronization

func StartTimer(name string) {
	mutex.Lock() // lock before writing to the map
	timers[name] = span{start: time.Now()}
	mutex.Unlock() // unlock after writing to the map
}

func StopTimer(name string) {
	mutex.Lock() // lock before accessing the map
	if _, exists := timers[name]; !exists {
		fmt.Printf("Timer %s doesn't exist\n", name)
		mutex.Unlock() // unlock before returning
		return
	}
	s := timers[name]
	timers[name] = span{start: s.start, end: time.Now()}
	mutex.Unlock() // unlock after writing to the map
	PrintTimers()
}

func PrintTimers() {
	mutex.Lock() // lock before reading from the map
	// Create a slice for keys and sort it
	keys := make([]string, 0, len(timers))
	for key := range timers {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	// Print the timers by sorted key (name)
	fmt.Printf("%-10s %-20s\n", "Name", "Duration")
	for _, name := range keys {
		fmt.Printf("%-10s %-20v\n", name, timers[name].end.Sub(timers[name].start))
	}
	mutex.Unlock() // unlock after reading from the map
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

	mutex.Lock() // lock before reading from the map
	for name, timer := range timers {
		writer.Write([]string{name, timer.end.Sub(timer.start).String()})
	}
	mutex.Unlock() // unlock after reading from the map
}
