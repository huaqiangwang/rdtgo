package main

import (
	"flag"
	"fmt"
	"github.com/shirou/gopsutil/process"
	"os"
	"time"
)

func init() {

}

func welcomme() {
	fmt.Println()
	fmt.Println("    Go scripts for evaluating Intel RDT(CMT,MBM)")
	fmt.Println()
}

// a workload for testing, merely do adding operation
func workload_counting() {
	var result int
	result = 0
	for {
		result = result + 1
	}
}

// Show current CPU id
func show_current_pid() {
	for {
		pid := os.Getpid()
		fmt.Println("    - PID: ", pid)
		pInstance, _ := process.NewProcess(int32(pid))
		/*
			if err {
				fmt.Println(err)
				continue
			}
		*/
		percent, _ := pInstance.CPUPercent()
		fmt.Println("Current CPU id\n", percent)

		time.Sleep(1 * time.Second)
	}
}

// cmt monitoring on self process
// - Get current pid
// - Create MON_GROUP, and the pid to corresponding task file
// - Set CPU mask
// TO check:
// 	* Effection for CPU binding and not binding
func cmt_mornitoring_start() {

}

// - CMT monitoring -
// only called after a successul invocation of cmt_mornitoring_start
func cmt_monitoring() {
}

func main() {
	flag.Parse()

	if flag.NArg() == 0 {
		welcomme()
	}

	// kick off an workload
	go workload_counting()

	// kick off cmt mornitoring thread
	go cmt_monitoring()

	go show_current_pid()

	time.Sleep(100 * time.Second)

	fmt.Println("Main thread existed")
}
