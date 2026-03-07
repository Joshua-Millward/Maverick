package main

import (
	"encoding/hex"
	"errors"
	"fmt"
	"math/rand"
	"strings"

	ax "github.com/Adaptix-Framework/axc2"
)

func CreateAgent(initialData []byte) (ax.AgentData, ax.ExtenderAgent, error) {
	var (
		agent ax.AgentData
		mvcfg MaverickData
	)

	packer := CreatePacker(initialData)

	command := packer.ParseInt8()
	if command != 0xf1 {
		return agent, ModuleObject.ext, errors.New("error agent checkin data")
	}

	mvcfg.Session.AgentIdStr = string(packer.ParsePad(36))
	agentId := fmt.Sprintf("%08x", rand.Uint32())
	mvcfg.Session.AgentIdStr = agentId

	mvcfg.Machine.OsArch = packer.ParseInt8()
	mvcfg.Machine.Username = string(packer.ParseBytes())
	mvcfg.Machine.Computer = packer.ParseString()
	mvcfg.Machine.Domain = packer.ParseString()
	mvcfg.Machine.Netbios = packer.ParseString()
	mvcfg.Session.ProcessId = uint32(packer.ParseInt32())
	mvcfg.Session.ThreadId = uint32(packer.ParseInt32())
	mvcfg.Session.ImgPath = packer.ParseString()

	mvcfg.Session.Acp = uint32(packer.ParseInt32())
	mvcfg.Session.Oemcp = uint32(packer.ParseInt32())

	mvcfg.Session.SleepTime = uint32(packer.ParseInt32())
	mvcfg.Session.Jitter = uint32(packer.ParseInt32())

	mvcfg.Machine.Ipaddress = int32_to_ipv4(packer.ParseInt32())

	mvcfg.Machine.OsMajor = uint32(packer.ParseInt32())
	mvcfg.Machine.OsMinor = uint32(packer.ParseInt32())
	mvcfg.Machine.OsBuild = uint32(packer.ParseInt32())

	mvcfg.Session.Elevated = packer.ParseInt32() != 0

	key := packer.ParseBytes()

	process := ConvertCpToUTF8(mvcfg.Session.ImgPath, int(mvcfg.Session.Acp))
	if strings.Contains(process, "\\") {
		parts := strings.Split(process, "\\")
		process = parts[len(parts)-1]
	}
	mvcfg.Session.ImgName = process

	osDesc := GetWindowsVersionName(mvcfg.Machine.OsMajor, mvcfg.Machine.OsMinor, mvcfg.Machine.OsBuild)

	agent = ax.AgentData{
		Id:         agentId,
		SessionKey: key,
		OemCP:      int(mvcfg.Session.Oemcp),
		ACP:        int(mvcfg.Session.Acp),
		Sleep:      uint(mvcfg.Session.SleepTime / 1000),
		Jitter:     uint(mvcfg.Session.Jitter),
		Username:   ConvertCpToUTF8(mvcfg.Machine.Username, int(mvcfg.Session.Acp)),
		Computer:   ConvertCpToUTF8(mvcfg.Machine.Computer, int(mvcfg.Session.Acp)),
		Process:    process,
		Pid:        fmt.Sprintf("%v", mvcfg.Session.ProcessId),
		Tid:        fmt.Sprintf("%v", mvcfg.Session.ThreadId),
		Arch: func() string {
			if mvcfg.Machine.OsArch == 0x86 {
				return "x86"
			}
			return "x64"
		}(),
		Elevated:   mvcfg.Session.Elevated,
		Os:         OS_WINDOWS,
		OsDesc:     osDesc,
		InternalIP: mvcfg.Machine.Ipaddress,
		Domain:     mvcfg.Machine.Domain,
	}

	mvcfg.JsonMarshal(&agent, true)

	return agent, ModuleObject.ext, nil
}

func PackTasks(agentData ax.AgentData, tasksArray []ax.TaskData) ([]byte, error) {
	var (
		array []interface{}
		err   error
	)

	array = append(array, int8(0))
	array = append(array, len(tasksArray))

	for _, taskData := range tasksArray {
		randomId := make([]byte, 19)
		_, _ = rand.Read(randomId)
		taskUID := taskData.TaskId + hex.EncodeToString(randomId)

		array = append(array, taskUID)
		array = append(array, len(taskData.Data))
		array = append(array, taskData.Data)
	}

	packData, err := PackArray(array)
	if err != nil {
		return nil, err
	}

	return packData, nil
}

func CreateTask(ts Teamserver, agent ax.AgentData, args map[string]any) (ax.TaskData, ax.ConsoleMessageData, error) {
	var (
		taskData    ax.TaskData
		messageData ax.ConsoleMessageData
	)

	command, ok := args["command"].(string)
	if !ok {
		return taskData, messageData, errors.New("'command' must be set")
	}

	def, exists := commandRegistry[command]
	if !exists {
		return taskData, messageData, fmt.Errorf("unknown command: %s", command)
	}

	array, msg, err := def.Create(args)
	if err != nil {
		return taskData, messageData, err
	}

	taskData = ax.TaskData{
		Type: TYPE_TASK,
		Sync: true,
	}

	messageData = ax.ConsoleMessageData{
		Status:  MESSAGE_INFO,
		Message: msg,
	}
	if m, ok := args["message"].(string); ok {
		messageData.Message = m
	}

	if array != nil {
		taskData.Data, err = PackArray(array)
		if err != nil {
			return taskData, messageData, err
		}
	}

	return taskData, messageData, nil
}
