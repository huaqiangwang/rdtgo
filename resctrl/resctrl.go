package resctrl

import (
	"fmt"
	"time"
	"os"
	"syscall"
	"io/ioutil"
	"strings"
	"strconv"
)

/*
#include <stdlib.h>

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
 */
import "C"

type ResctrlFs struct{
	bMount bool
}

func NewResctrlFs()( *ResctrlFs,  error){
	res := &ResctrlFs{
		bMount:false,
	}
	return res, nil
}

func (r *ResctrlFs) Mount(){
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
	cbmMask int64 ){
	fmt.Println("Create cache allocation policy")
}

// Report cache information
// - Total cache
// - cache way
// - capacity per cache way
func (r * ResctrlFs) CacheInfo(){

}

func (r* ResctrlFs) WkldAllocaCache(size int) int64 {
	return int64(C.initCache(C.int(size)))
}

func (r* ResctrlFs) WkldFreeCache(pl int64){
	C.deInitCache(C.ulong(pl))
}

func (r*ResctrlFs) WkldCacheLoop(pl int64, size int){
	C.cacheLoop(C.ulong(pl), C.int(size) )

}
// Workload for a specific cache occupancy
func (r* ResctrlFs) WkldCache( occupancyKB int, delay time.Duration) bool{
	addr := r.WkldAllocaCache(occupancyKB*1024)
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

const ResPath = "/sys/fs/resctrl"

func (r*ResctrlFs) CreateMONGroup( group string) bool{
	folderName := ResPath+"/"+group
	if stat,err := os.Stat(folderName); err==nil && stat.IsDir(){
		return true
	}
	if err := os.Mkdir(folderName,0555); err != nil {
		fmt.Print(err.Error())
		return false
	}
	return true
}

func (r*ResctrlFs) DestroyMonGroup( group string) bool{
	folderName := ResPath+"/"+group
	if stat,err := os.Stat(folderName); err==nil && stat.IsDir() {
		syscall.Rmdir(folderName)
	}

	if _,err := os.Stat(folderName); os.IsNotExist(err){
		return true
	}
	return false
}

func (r*ResctrlFs) CheckL3CacheOccupancy(group string) []int64{
	folderName := ResPath+"/" + group + "/mon_data/"
	var occuL3 []int64
	if stat,err := os.Stat(folderName); err==nil && stat.IsDir() {
		const occuL3FolderBase = "mon_L3_0"
		files, err := ioutil.ReadDir(folderName)
		if err != nil{
			return nil
		}
		for _, f := range files {
			if f.IsDir() && strings.Contains(f.Name(), occuL3FolderBase){
				file,err := os.Open(folderName+"/"+f.Name()); if err==nil{
					buf := make([]byte,256)
					file.Read(buf)
					occpancy,_ := strconv.ParseInt(string(buf), 10, 0)
					occuL3 = append(occuL3, occpancy)
					//fmt.Println(f.Name()," cache occupancy ", occpancy)
				}
				file.Close()
			}
		}

	}
	return occuL3
}

func (r * ResctrlFs) BindTasktoGroup(
	pid int,
	group string ) error {

	fileName := ResPath + "/" + group + "/tasks"
	fileCPUList, err := os.OpenFile(fileName,os.O_APPEND|os.O_WRONLY,0); if err!=nil {
		fmt.Println("Error in open tasks file: ", err.Error())
		return err
	}
	_,err = fileCPUList.WriteString(string(pid)); if err!=nil{
		fmt.Println("Error in set tasks", err.Error())
	}
	fileCPUList.Close()
	return err
}

func (r* ResctrlFs) BindCPUtoGroup(
	cpuRangeStart int, cpuRangeEnd int, group string) error{
	fileName := ResPath + "/" + group + "/cpus_list"
	fileCPUList, err := os.OpenFile(fileName,os.O_APPEND|os.O_WRONLY,0); if err!=nil {
		fmt.Println("Error in open cpus_list file: ", err.Error())
		return err
	}
	_,err = fileCPUList.WriteString(
		string(cpuRangeStart)+"-"+string(cpuRangeEnd)); if err!=nil{
		fmt.Println("Error in append CPU list", err.Error())
	}
	fileCPUList.Close()
	return err
}

func (r* ResctrlFs) SetCacheSettoGroup(
	strSchemata string, group string) error{
	fileName := ResPath + "/" + group + "/schemata"
	fileCPUList, err := os.OpenFile(fileName,os.O_APPEND|os.O_WRONLY,0); if err!=nil {
		fmt.Println("Error in open schemata file: ", err.Error())
		return err
	}
	// TODO: or get the detail error messgae from last_cmd_status
	_,err = fileCPUList.WriteString(strSchemata); if err !=nil{
		fmt.Println("Error in set schemata", err.Error())
	}
	fileCPUList.Close()
	return err
}