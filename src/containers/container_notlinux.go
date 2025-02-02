// +build !linux

package containers

import (
	"log"
	"syscall"
	"utils"
	"net/url"
)

const BASH_PATH = "/bin/bash"

type container struct {
	// Name of the container as per command Name.
	Name string
}

func (c *container) AddContainerAttributes(containerAttribs *syscall.SysProcAttr, params url.Values) {
	log.Println("INFO: call to AddContainerAttributes")
	return
}

func (c *container) AddProcess(pid int) {
	log.Println("INFO: Waiting for command to finish...", pid)
}

func (c *container) IsProcess(pid int) bool {
	return false
}

func (c *container) Delete() {
	log.Println("container deleted for : ", c.Name)
}

func (c *container) AddProcesstoNewSubCgroup(pid int, iscompiled bool) {
	log.Println("AddProcesstoNewSubCgroup called for : ", c.Name, pid, iscompiled)
}

func (c *container) DeleteProcessFromSubCgroup(pid int) {
	log.Println("DeleteProcessFromSubCgroup called for : ", c.Name, pid)
}

func NewContainer(name string, memlimit int64) (*container, error) {
	var containerObj container
	containerObj.Name = name
	log.Println("INFO: New Container created for : ", name)
	return &containerObj, nil
}

func EnableNetworking(pid int) {
    log.Println("Network enabled for pid: ", pid)
}

func GetCommandArgs(command string, argv []string, ppid int, params map[string][]string) (commandArgs []string) {
	var commandlist []string
	var err error
	if utils.Iscompiled(params) {
		commandpath := BASH_PATH
		commandlist = append(commandlist, commandpath)
	} else {
		commandlist = append(commandlist, command)
	}
	commandlist = append(commandlist, argv...)
	if utils.Iscompiled(params) {
		filename := utils.GetIdeFileName(params)
		var arg0 string = utils.GetIdeContent(params)
		var otherargs string = utils.GetCompilerFlags(params)
		err = utils.SaveIdeContentToFile(params, filename)
		if err == nil {
			// file saved successfully now i can pass filename as arg0
			arg0 = filename
		}
		//this is a compilation request
		compilerOptions := []string {"-c", utils.GetCompilationScript(command), arg0, otherargs}
		commandlist = append(commandlist, compilerOptions...)
	}
	commandArgs = append(commandArgs, commandlist...)
	log.Println("commands Args: ", commandArgs)
	return commandArgs
}

// get working directory for a running process, pid
func GetWorkingDir(pid int) string {
	return ""
}


func GetHomeDirFromEnv(pid int) string {
	return ""
}