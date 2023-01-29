// +build linux

package containers

import (
	"github.com/containerd/cgroups"
	specs "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/pkg/errors"
	"bytes"
	"log"
	"math"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"syscall"
	"utils"
)

const BASH_PATH = "/bin/bash"
var CPUshares = uint64(1024)

type container struct {
	// Name of the container as per command Name.
	Name string
	// Control holds container specific control/limits.
	Control cgroups.Cgroup

	// SysProcAttr holds optional, operating system-specific attributes to containerize the Process.
	SysProcAttr *syscall.SysProcAttr

	SubCgroups map[int]cgroups.Cgroup // map of pid to cgroup control of subcgroups
	mu         sync.Mutex
}

func (c *container) AddContainerAttributes(containerAttribs *syscall.SysProcAttr) {
	containerAttribs.Cloneflags = c.SysProcAttr.Cloneflags
	containerAttribs.UidMappings = c.SysProcAttr.UidMappings
	containerAttribs.GidMappings = c.SysProcAttr.GidMappings
}

func (c *container) AddProcess(pid int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.Control.Add(cgroups.Process{Pid: pid}); err != nil {
		log.Printf("ERROR: Error while adding process : %d to cgroups: %s\n", pid, err.Error())
	}
	log.Println("INFO: Waiting for command to finish...", pid)
}

func (c *container) IsProcess(pid int) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, ok := c.SubCgroups[pid]
	return ok
}

func (c *container) Delete() {
	c.Control.Delete()
	log.Println("controller deleted for : ", c.Name)
	// TBD : delete sub cgroup MAP
}

func (c *container) AddProcesstoNewSubCgroup(pid int, iscompiled bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	pidstr := strconv.Itoa(pid)
	memlimit := Commands2memLimitMap[c.Name] * MB // mem limit in MBs
	if iscompiled {
		memlimit = int64(math.Floor(2.0*float64(memlimit)))
		log.Println("effective memlimit: ",memlimit)
	}
	control, err := c.Control.New(pidstr, &specs.LinuxResources{
		/*CPU: &specs.LinuxCPU{
		        Shares: &CPUshares,
		        Cpus:   "0",
		        Mems:   "0",
		},*/
		Memory: &specs.LinuxMemory{
			Limit: &memlimit,
		},
	})
	if err != nil {
		log.Println("ERROR: unable to create sub cgroup for : " + c.Name + " pid: " + pidstr + " error: " + err.Error())
		return
	}
	err = control.Add(cgroups.Process{Pid: pid})
	if err != nil {
		log.Println("ERROR: Unable to add process to sub cgroup: " + c.Name + " pid: " + pidstr + " error: " + err.Error())
		control.Delete()
		return
	}
	c.SubCgroups[pid] = control
}

func (c *container) DeleteProcessFromSubCgroup(pid int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	cgroup, ok := c.SubCgroups[pid]
	if ok {
		err := cgroup.Delete()
		if err != nil {
			log.Println("ERROR: Unable to delete cgroup for : " + c.Name + " pid: " + strconv.Itoa(pid) + " ERROR: " + err.Error())
		}
		delete(c.SubCgroups, pid)
		log.Println("container deleted for "+c.Name+" pid: ", pid)
	}
}

func NewContainer(name string, memlimit int64) (*container, error) {
	var containerObj container
	var err error
	containerObj.Name = name
	// Load and clean up the containers if already exists
	ctrl, err := cgroups.Load(cgroups.V1, cgroups.StaticPath("/"+name+"_container"))
	if err == nil {
		log.Println("INFO: cgroup already exists, deleting: " + name + "_container")
		ctrl.Delete()
	}
	containerObj.Control, err = cgroups.New(cgroups.V1, cgroups.StaticPath("/"+name+"_container"), &specs.LinuxResources{
		/*CPU: &specs.LinuxCPU{
		        Shares: &CPUshares,
		        Cpus:   "0",
		        Mems:   "0",
		},*/
		Memory: &specs.LinuxMemory{
			Limit: &memlimit,
		},
	})
	if err != nil {
		return &containerObj, errors.New("Unable to create Container: " + err.Error())
	}

	containerObj.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags:   syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS | syscall.CLONE_NEWNET | syscall.CLONE_NEWUSER,
		Unshareflags: syscall.CLONE_NEWNS | syscall.CLONE_NEWNET,
		UidMappings: []syscall.SysProcIDMap{
			{
				ContainerID: 0,
				HostID:      os.Getuid(),
				Size:        1,
			},
		},
		GidMappings: []syscall.SysProcIDMap{
			{
				ContainerID: 0,
				HostID:      os.Getgid(),
				Size:        1,
			},
		},
	}
	containerObj.SubCgroups = make(map[int]cgroups.Cgroup)
	log.Println("INFO: New Container created for : ", name)
	return &containerObj, nil
}


func EnableNetworking(pid int) {
	//log.Println("euid: ",syscall.Getuid(), syscall.Geteuid())
	var b bytes.Buffer
	cmd := exec.Command("/usr/bin/nsenter", "-n", "-t"+strconv.Itoa(pid), "ifconfig", "lo", "up")
    cmd.Stdout = &b
    cmd.Stderr = &b
    err := cmd.Start()
    if err != nil {
       log.Println("ERROR: error enabling network for pid: ", pid, "error: ", err.Error(), string(b.Bytes()))
    }
}

func GetCommandArgs(command string, argv []string, ppid int, params map[string][]string) (commandArgs []string) {
	var commandlist []string
	var commandpath string
	var err error
	if utils.Iscompiled(params) {
		commandpath = BASH_PATH
	} else {
		commandpath, err = exec.LookPath(command)		// lookup path for absolutepath
		if err != nil {
			commandpath = command
		}
		prefix:= utils.GetPrefix(command)
		if prefix != "" {
			commandlist = append(commandlist, prefix)	// add prefix before calling the command
		}
	}
	commandlist = append(commandlist, commandpath)
	commandlist = append(commandlist, argv...)
	if utils.Iscompiled(params) {
		//this is a compilation request
		compilerOptions := []string {"-c", utils.GetCompilationScript(command), utils.GetIdeContent(params)}
		commandlist = append(commandlist, compilerOptions...)
	}
	if ppid == -1 {
		commandArgs = append(commandArgs, commandlist...)
	} else {
		// this process will run in namespace of ppid (forked namespace)
		// syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS | syscall.CLONE_NEWNET | syscall.CLONE_NEWUSER
		nsenterArgs := []string{"/usr/bin/nsenter", "-u", "-p", "-n", "-U", "-t"+strconv.Itoa(ppid)} 
		commandArgs = append(commandArgs, nsenterArgs...)
		commandArgs = append(commandArgs, commandlist...)
	}
	log.Println("commands Args: ", commandArgs)
	return commandArgs
}

// get working directory for a running process, pid
func GetWorkingDir(pid int) string {
	path := "/proc/" + strconv.Itoa(pid) + "/cwd"
	wd, err := os.Readlink(path)
	if err != nil {
		log.Println(err)
		return ""
	}
	return wd
}
