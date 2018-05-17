package resctrl_test

import (
	"../resctrl"
	"fmt"
	"os"
	"testing"
	"time"
)

func TestWkldResourceAllocate(t *testing.T) {
	t.Log("Step1 - Allocating a specified size cache, size = 1M")
	r, _ := resctrl.NewResctrlFs()
	addr := r.WkldAllocaCache(1024 * 1024)
	t.Log("   - Got allocated address ", addr)
	if addr == 0 {
		t.Error("Nil address returned")
		return
	}

	t.Log("Step2 - Free address")
	r.WkldFreeCache(addr)

	t.Log("Step3 - Workload Processing")
	r.WkldCache(1024, time.Second)
}

func TestMonGroup(t *testing.T) {
	stage := 0
	const group = "p1"
	r, _ := resctrl.NewResctrlFs()
	if err := r.CreateMONGroup(group); err {
		stage += 1
		occuL3 := r.CheckL3CacheOccupancy(group)
		//t.Log("len of occupancy of L3 cache is ", len(occuL3))
		for i, occu := range occuL3 {
			t.Log("  L3 ", i, " = ", occu)
		}
		stage += 1
		r.DestroyMonGroup(group)
		stage += 1
	}
	if stage == 0 {
		t.Error("CreateMonGroup failed")
	}
	t.Log("Stage = ", stage)

}

func cacheComsumer(occupancyKB int, r *resctrl.ResctrlFs) {
	cacheSize := occupancyKB * 1024
	addr := r.WkldAllocaCache(cacheSize)
	for {
		r.WkldCacheLoop(addr, cacheSize)
	}
	defer r.WkldFreeCache(addr)
}

func dumpCacheOccupency(group string, r *resctrl.ResctrlFs) {
	for {
		occuL3 := r.CheckL3CacheOccupancy(group)
		for i, occu := range occuL3 {
			fmt.Println("  L3 ", i, " = ", occu)
		}
		time.Sleep(time.Second)
	}
}

func TestBindTaskToGroup(t *testing.T) {
	const group = "p1"
	r, _ := resctrl.NewResctrlFs()
	if err := r.CreateMONGroup(group); err {
		pid := os.Getpid()
		r.BindTasktoGroup(pid, group)
		r.BindCPUtoGroup(4, 8, group)

		go cacheComsumer(5*1024, r)
		go dumpCacheOccupency(group, r)
	}
	time.Sleep(1 * time.Second)
	r.DestroyMonGroup(group)
}

func TestBindTaskToGroupandMoveGroup(t *testing.T) {
	const group0 = "mon_groups/test1"
	const group1 = "p0"
	r, _ := resctrl.NewResctrlFs()
	pid := os.Getpid()
	go cacheComsumer(5*1024, r)

	if err := r.CreateMONGroup(group0); !err {
		t.Error("Failed in create ", group0)
		return
	}
	r.BindTasktoGroup(pid, group0)
	if err := r.DumpFileContentInt(group0, "tasks", "Dump " + group0+ "/tasks: ");
		err != nil {
		t.Error("Error in dump file content")
		return
	}

	if err := r.CreateMONGroup(group1); !err {
		t.Error("Failed in create ", group0)
		return
	}
	r.BindTasktoGroup(pid, group1)
	if err := r.DumpFileContentInt(group0, "tasks","Dump " + group0+ "/tasks: ");
		err != nil {
		t.Error("Error in dump file content")
		return
	}
	if err := r.DumpFileContentInt(group1, "tasks","Dump " + group1+ "/tasks: ");
			err != nil {
		t.Error("Error in dump file content")
		return
	}

	time.Sleep(1 * time.Second)
}

func TestResctrlFs_CacheInfo(t *testing.T) {
	r, _ := resctrl.NewResctrlFs()
	r.CacheInfo()
	// dump information

	for _, info := range r.Info {
		t.Log(info.Res.Name, " = ", info.Value)
	}
}
