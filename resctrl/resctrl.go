package resctrl

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"
)

/*
#include <stdlib.h>
#include <stdio.h>

unsigned long initCache(int size){
	char * pc = (char*)malloc(size);
	return (unsigned long)pc;
}
void * deInitCache(unsigned long pc) {
	if(pc!=0)
		free((void*)pc);
}

void cacheLoop(unsigned long pl, int size){
	int i;
	char * pc;
	if( pl == 0)
		return;
	pc = (char*)pl;

	for (i=0;i<size;i++){
		*(pc+i) = (char)i;
	}
}

void _getCPUID(long catalog, long rev,
		long * a, long *b, long *c, long* d){
        __asm__ __volatile__("cpuid"
        :"=a"(*a),"=b"(*b),"=c"(*c),"=d"(*d)
        :"a"(catalog),"c"(rev));
}

void getCPUID(long catalog, long rev){
	long a,b,c,d;
	_getCPUID(catalog, rev, &a,&b,&c,&d);
	printf("CPUID[0x%x Rev%d] = 0x%016lx - %016lx - %016lx - %016lx \n",
	catalog, rev, a,b,c,d);
}

*/
import "C"

const ResPath = "/sys/fs/resctrl"

const (
	RES_TYPE_VALUE int = iota
	RES_TYPE_STRING
)

type resNameAndType struct {
	Name string
	Type int
}

type resourceInfo struct {
	Res   resNameAndType
	Value interface{}
}

var RdtInfo []resNameAndType = []resNameAndType{
	{"L3/cbm_mask", RES_TYPE_STRING},
	{"L3/min_cbm_bits", RES_TYPE_STRING},
	{"L3/num_closids", RES_TYPE_VALUE},
	{"L3/sharable_bits", RES_TYPE_VALUE},
	{"L3_MON/max_threshold_occupancy", RES_TYPE_VALUE},
	{"L3_MON/mon_features", RES_TYPE_STRING},
	{"L3_MON/rmids", RES_TYPE_VALUE},
}

type ResctrlFs struct {
	bMount bool
	Info   []resourceInfo
}

func NewResctrlFs() (*ResctrlFs, error) {
	res := &ResctrlFs{
		bMount: false,
	}
	return res, nil
}

func (r *ResctrlFs) Mount() {
	fmt.Println("Mount 'resctrl' file system")
	r.bMount = true
}

// Allocating cache service-of-class policy
// - mkdir
// - set CPU mask
// - set Cache mask
func (r *ResctrlFs) CreateCacheQuota(
	cpumask []uint64,
	cpumaskLen int64,
	cbmMask int64) {
	fmt.Println("Create cache allocation policy")
}

// Report cache information
// - Total cache
// - cache way
// - capacity per cache way
func (r *ResctrlFs) CacheInfo() {
	stat, err := os.Stat(ResPath)
	if err != nil || !stat.IsDir() {
		fmt.Println("No resctrl root folder, mount first")
		return
	}

	cacheInfoCollection := make([]resourceInfo, len(RdtInfo))
	for index, resName := range RdtInfo {
		cacheInfoCollection[index].Res.Name = RdtInfo[index].Name
		cacheInfoCollection[index].Res.Type = RdtInfo[index].Type
		if RdtInfo[index].Type == RES_TYPE_STRING {
			cacheInfoCollection[index].Value = "error"
		} else {
			cacheInfoCollection[index].Value = 0
		}
		//fmt.Println(ResPath + "/info/" + resName.Name)
		if file, err := os.Open(ResPath + "/info/" + resName.Name); err == nil {
			buf := make([]byte, 256)
			if _, err := file.Read(buf); err != nil {
				fmt.Println("Failed in getting info from ",
					resName.Name,
					": ", err.Error())

				// TODO: using 'switch/case'
				if resName.Type == RES_TYPE_STRING {
					cacheInfoCollection[index].Value = "error"
				} else {
					cacheInfoCollection[index].Value = 0
				}
			} else {
				//fmt.Println(buf)
				if resName.Type == RES_TYPE_STRING {
					cacheInfoCollection[index].Value = string(buf)
				} else {
					lines := strings.Split(string(buf), "\n")
					if len(lines) > 0 {
						// should only be valid for first line
						cacheInfoCollection[index].Value, _ = strconv.ParseInt(lines[0], 10, 0)

					}
				}
			}
			file.Close()
		}

	}
	r.Info = cacheInfoCollection
}

func (r *ResctrlFs) WkldAllocaCache(size int) int64 {
	return int64(C.initCache(C.int(size)))
}

func (r *ResctrlFs) WkldFreeCache(pl int64) {
	C.deInitCache(C.ulong(pl))
}

func (r *ResctrlFs) WkldCacheLoop(pl int64, size int) {
	C.cacheLoop(C.ulong(pl), C.int(size))

}

// Workload for a specific cache occupancy
func (r *ResctrlFs) WkldCache(occupancyKB int, delay time.Duration) bool {
	addr := r.WkldAllocaCache(occupancyKB * 1024)
	if addr == 0 {
		return false
	}

	tMark := time.Now().Add(delay)

	for time.Now().Before(tMark) {
		r.WkldCacheLoop(int64(addr), occupancyKB*1024)
	}
	r.WkldFreeCache(int64(addr))
	return true
}

func (r *ResctrlFs) CreateMONGroup(group string) bool {
	folderName := ResPath + "/" + group
	if stat, err := os.Stat(folderName); err == nil && stat.IsDir() {
		return true
	}
	if err := os.Mkdir(folderName, 0555); err != nil {
		fmt.Print(err.Error())
		return false
	}
	return true
}

func (r *ResctrlFs) DestroyMonGroup(group string) bool {
	folderName := ResPath + "/" + group
	if stat, err := os.Stat(folderName); err == nil && stat.IsDir() {
		syscall.Rmdir(folderName)
	}

	if _, err := os.Stat(folderName); os.IsNotExist(err) {
		return true
	}
	return false
}

func (r *ResctrlFs) CheckL3CacheOccupancy(group string) []int64 {
	folderName := ResPath + "/" + group + "/mon_data/"
	var occuL3 []int64
	if stat, err := os.Stat(folderName); err == nil && stat.IsDir() {
		const occuL3FolderBase = "mon_L3_0"
		files, err := ioutil.ReadDir(folderName)
		if err != nil {
			return nil
		}
		for _, f := range files {
			if f.IsDir() && strings.Contains(f.Name(), occuL3FolderBase) {
				os.Chdir(folderName + "/" + f.Name())
				file, err := os.Open("llc_occupancy")
				if err == nil {
					buf := make([]byte, 256)
					file.Read(buf)
					//fmt.Println(buf)
					lines := strings.Split(string(buf), "\n")
					strOccu := "0"
					if len(lines) != 0 {
						strOccu = lines[0]
					}
					//fmt.Println(strOccu)
					occupancy, _ := strconv.ParseInt(strOccu, 10, 0)
					occuL3 = append(occuL3, occupancy)
				}
				file.Close()
			}
		}

	}
	return occuL3
}

func (r *ResctrlFs) DumpFileContentInt(
	strFolder string, strFile string, msg string)  error {

	folderName := ResPath + "/" + strFolder
	os.Chdir(folderName)

	file, err := os.Open(strFile)
	defer file.Close()

	if err == nil {
		buf := make([]byte, 256)
		file.Read(buf)
		if buf[0] == 0 {
			fmt.Println(msg,"0")
		}else {
			fmt.Println(msg, string(buf))
		}
	}

	return err
}

func (r *ResctrlFs) BindTasktoGroup(
	pid int,
	group string) error {

	if err := os.Chdir(ResPath + "/" + group); err != nil {
		fmt.Println("Error in chdir, ", err.Error())
	}
	//dir,_= os.Getwd();fmt.Println(" - ",dir);
	fileMask, err := os.OpenFile("tasks", os.O_WRONLY, os.ModePerm)
	if err != nil {
		fmt.Println("Error in open tasks file: ", err.Error())
		return err
	}
	//fmt.Println("pid=",strconv.Itoa(pid))
	_, err = fileMask.WriteString(strconv.Itoa(pid))
	if err != nil {
		fmt.Println("Error in set tasks", err.Error())
	}
	fileMask.Close()
	return err
}

func (r *ResctrlFs) BindCPUtoGroup(
	cpuRangeStart int, cpuRangeEnd int, group string) error {
	os.Chdir(ResPath + "/" + group)
	fileCPUList, err := os.OpenFile("cpus_list", os.O_RDWR, os.ModePerm)
	if err != nil {
		fmt.Println("Error in open cpus_list file: ", err.Error())
		return err
	}
	_, err = fileCPUList.WriteString(
		strconv.Itoa(cpuRangeStart) + "-" + strconv.Itoa(cpuRangeEnd))
	if err != nil {
		fmt.Println("Error in append CPU list", err.Error())
	}
	fileCPUList.Close()
	return err
}

func (r *ResctrlFs) SetCacheSettoGroup(
	strSchemata string, group string) error {
	os.Chdir(ResPath + "/" + group)
	fileCPUList, err := os.OpenFile("schemata", os.O_WRONLY, os.ModePerm)
	if err != nil {
		fmt.Println("Error in open schemata file: ", err.Error())
		return err
	}
	// TODO: or get the detail error message from last_cmd_status
	_, err = fileCPUList.WriteString(strSchemata)
	if err != nil {
		fmt.Println("Error in set schemata", err.Error())
	}
	fileCPUList.Close()
	return err
}

func (r *ResctrlFs) GetRDTCPUIDInfo(){
	fmt.Println("RDT Allocation related CPUID info.")
	C.getCPUID(C.long(0x10), C.long(0))
	C.getCPUID(C.long(0x10), C.long(1))
	C.getCPUID(C.long(0x10), C.long(2))
	C.getCPUID(C.long(0x10), C.long(3))

	fmt.Println("RDT Monitoring related CPUID info.")
	C.getCPUID(C.long(0xf), C.long(0))
	C.getCPUID(C.long(0xf), C.long(1))
}

