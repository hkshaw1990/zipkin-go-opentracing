package perfevents

import (
	"encoding/binary"
	"fmt"
	"syscall"
	"unsafe"
)

// PerfEventAttr structure translated from linux/perf_event.h
// This struct defines various attributes for a perf event.
type PerfEventAttr struct {
	type_hw            uint32
	size_s             uint32
	config             uint64
	sample_period      uint64
	sample_type        uint64
	read_format        uint64
	properties         uint64
	wakeup_events      uint32
	bp_type            uint32
	config1            uint64
	config2            uint64
	branch_sample_type uint64
	sample_regs_user   uint64
	sample_stack_user  uint32
	clockid            int32
	sample_regs_intr   uint64
	aux_watermark      uint32
	reserved_2         uint32
}

// Bit fields for the PerfEventAttr.properties value translated from
// linux/perf_event.h
const (
	DISABLED                 = 0 // Starts from bit value 0
	INHERIT                  = 1
	PINNED                   = 2
	EXCLUSIVE                = 3
	EXCLUDE_USER             = 4
	EXCLUDE_KERNEL           = 5
	EXCLUDE_HV               = 6
	EXCLUDE_IDLE             = 7
	MMAP                     = 8
	COMM                     = 9
	FREQ                     = 10
	INHERIT_STAT             = 11
	ENABLE_ON_EXEC           = 12
	TASK                     = 13
	WATERMARK                = 14
	PRECISE_IP1              = 15
	PRECISE_IP2              = 16
	MMAP_DATA                = 17
	SAMPLE_ID_ALL            = 18
	EXCLUDE_HOST             = 19
	EXCLUDE_GUEST            = 20
	EXCLUDE_CALLCHAIN_KERNEL = 21
	EXCLUDE_CALLCHAIN_USER   = 22
	MMAP2                    = 23
	COMM_EXEC                = 24
	USE_CLOCKID              = 25
	CONTEXT_SWITCH           = 26
	RESERVED_1               = 27
)

// PMU hardware type definitions (from linux/perf_event.h)
const (
	PERF_TYPE_HARDWARE = 0
	PERF_TYPE_SOFTWARE = 1
)

// List of generic events supported (from linux/perf_event.h)
const (
	PERF_HW_CPU_CYCLES          = 0
	PERF_HW_INSTRUCTIONS        = 1
	PERF_HW_CACHE_REF           = 2
	PERF_HW_CACHE_MISSES        = 3
	PERF_HW_BRANCH_INSTRUCTIONS = 4
	PERF_HW_BRANCH_MISSES       = 5
	PERF_HW_BUS_CYCLES          = 6
)

// EventConfigType : The configuration struct for an event
type EventConfigType struct {
	typeHw uint32
	config uint64
}

// Initializes the event list
func initEventList() map[string]EventConfigType {
	return map[string]EventConfigType{
		"cpu-cycles":              {PERF_TYPE_HARDWARE, PERF_HW_CPU_CYCLES},
		"instructions":        {PERF_TYPE_HARDWARE, PERF_HW_INSTRUCTIONS},
		"cache-references":    {PERF_TYPE_HARDWARE, PERF_HW_CACHE_REF},
		"cache-misses":        {PERF_TYPE_HARDWARE, PERF_HW_CACHE_MISSES},
		"branch-instructions": {PERF_TYPE_HARDWARE, PERF_HW_BRANCH_INSTRUCTIONS},
		"branch-misses":       {PERF_TYPE_HARDWARE, PERF_HW_BRANCH_MISSES},
		"bus-cycles":          {PERF_TYPE_HARDWARE, PERF_HW_BUS_CYCLES},
	}
}

func setupPerfEventAttr(eventConfig EventConfigType) PerfEventAttr {
	var eventAttr PerfEventAttr
	eventAttr.type_hw = eventConfig.typeHw
	eventAttr.config = eventConfig.config
	eventAttr.size_s = uint32(unsafe.Sizeof(eventAttr))
	eventAttr.properties = setBit(eventAttr.properties, DISABLED)
	eventAttr.properties = setBit(eventAttr.properties, EXCLUDE_KERNEL)
	eventAttr.properties = setBit(eventAttr.properties, EXCLUDE_HV)

	return eventAttr
}

func fetchPerfEventAttr(event string) (PerfEventAttr, int) {
	var eventAttr PerfEventAttr
	evList := initEventList()
	evConf, ok := evList[event]
	if ok == false {
		//fmt.Println("`event not supported`")
		return eventAttr, -1
	}
	return setupPerfEventAttr(evConf), 0
}

// Perf IOCTL operations for x86
const (
	PERF_IOC_RESET_X86   = 0x2403
	PERF_IOC_ENABLE_X86  = 0x2400
	PERF_IOC_DISABLE_X86 = 0x2401
)

// Perf IOCTL operations for powerpc
const (
	PERF_IOC_RESET_PPC   = 0x20002403
	PERF_IOC_ENABLE_PPC  = 0x20002400
	PERF_IOC_DISABLE_PPC = 0x20002401
)

// PerfEventInfo holds the file descriptor for a perf event
type PerfEventInfo struct {
	EventName string
	Fd        int
	Data      uint64
}

// FetchPerfEventAttr is the same as that of the independent one, just to maintain consistency, this method is defined
// TODO: remove the independent version of this method and use only this method.
func (event *PerfEventInfo) FetchPerfEventAttr(eventName string) (PerfEventAttr, int) {
	var eventAttr PerfEventAttr
	eventAttr, err := fetchPerfEventAttr(eventName)
	if err == -1 {
		event.Fd = -1
		event.Data = 0
	}
	return eventAttr, err
}

// InitOpenEventEnable fetches the perf event attributes for event "string",
// opens the event, resets and then enables the event.
func (event *PerfEventInfo) InitOpenEventEnable(eventName string, pid int, cpu int, group_fd int, flags uint64) int {
	eventAttr, err := event.FetchPerfEventAttr(eventName)
	// eventAttr, err := eventInfo.fetchPerfEventAttr(eventName)
	if err == -1 {
		return -1
	}
	err = event.OpenEvent(eventAttr, pid, cpu, group_fd, flags)
	if err != 0 {
		return err
	}
	//fmt.Printf("InitOpenEventEnable, Fd : %d\n", event.Fd)
	event.EventName = eventName
	err = event.ResetEvent()
	if err != 0 {
		return err
	}

	err = event.EnableEvent()
	if err != 0 {
		return err
	}

	return 0
}

// InitOpenEventEnableSelf opens, enables an event for self process
func (event *PerfEventInfo) InitOpenEventEnableSelf(eventName string) int {
	return event.InitOpenEventEnable(eventName, 0, -1, -1, 0)
}

// DisableClose disables the event and then closes it.
func (event *PerfEventInfo) DisableClose() int {
	if event.Fd < 2 {
		fmt.Println("event fd is unset")
		return -1
	}

	err := event.DisableEvent()
	if err == -1 {
		return err
	}

	errClose := syscall.Close(int(event.Fd))
	if errClose != nil {
		return -1
	}

	return 0
}

// OpenEvent opens an event
func (event *PerfEventInfo) OpenEvent(eventAttr PerfEventAttr, pid int, cpu int, group_fd int, flags uint64) int {
	if event.Fd > 2 {
		fmt.Println("File descriptor already set")
		return -1
	}
	fd, _, err := syscall.Syscall6(syscall.SYS_PERF_EVENT_OPEN, uintptr(unsafe.Pointer(&eventAttr)), uintptr(pid), uintptr(cpu), uintptr(group_fd), uintptr(flags), uintptr(0))
	//fmt.Println(int(fd))
	//fmt.Println(err)
	if err > 0 {
		fmt.Println("error")
		return -1
	}
	if int(fd) == -1 {
		return -1
	}
	event.Fd = int(fd)
	return 0
}

// ResetEvent resets and event
func (event *PerfEventInfo) ResetEvent() int {
	if event.Fd < 2 {
		fmt.Println("File descriptor is not set")
		return -1
	}
	_, _, err := syscall.Syscall6(syscall.SYS_IOCTL, uintptr(event.Fd), uintptr(PERF_IOC_RESET_X86), uintptr(0), uintptr(0), uintptr(0), uintptr(0))
	//fmt.Println(err)
	if err != 0 {
		return -1
	}
	return 0
}

// EnableEvent enables an event
func (event *PerfEventInfo) EnableEvent() int {
	if event.Fd < 2 {
		fmt.Println("File descriptor is not set")
		return -1
	}
	_, _, err := syscall.Syscall6(syscall.SYS_IOCTL, uintptr(event.Fd), uintptr(PERF_IOC_ENABLE_X86), uintptr(0), uintptr(0), uintptr(0), uintptr(0))
	if err != 0 {
		return -1
	}
	//fmt.Println(err)
	return 0
}

// DisableEvent disables an event
func (event *PerfEventInfo) DisableEvent() int {
	if event.Fd < 2 {
		fmt.Println("File descriptor is not set")
		return -1
	}
	_, _, err := syscall.Syscall6(syscall.SYS_IOCTL, uintptr(event.Fd), uintptr(PERF_IOC_DISABLE_X86), uintptr(0), uintptr(0), uintptr(0), uintptr(0))
	if err != 0 {
		return -1
	}
	//fmt.Println(err)
	return 0
}

// ReadEvent reads for an event
func (event *PerfEventInfo) ReadEvent() int {
	readBuf := make([]byte, 8, 10)
	_, err := syscall.Read(event.Fd, readBuf)
	if err != nil {
		//fmt.Print(err.Error(), "\n")
		return -1
	}
	data := binary.LittleEndian.Uint64(readBuf)
	event.Data = data
	return 0
}

func setBit(properties uint64, bitPos uint64) uint64 {
	properties |= (1 << bitPos)
	return properties
}
