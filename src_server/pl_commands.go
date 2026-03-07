package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	ax "github.com/Adaptix-Framework/axc2"
)

type CommandDef struct {
	ID      uint16
	Create  func(args map[string]any) ([]interface{}, string, error)
	Process func(resultData []byte, acp int) (message string, clearText string, msgType int)
}

var commandRegistry = map[string]CommandDef{}
var commandByID = map[uint16]*CommandDef{}

func registerCommand(name string, def CommandDef) {
	commandRegistry[name] = def
	d := commandRegistry[name]
	commandByID[def.ID] = &d
}

func init() {
	registerCommand("exit", CommandDef{
		ID: 0x10,
		Create: func(args map[string]any) ([]interface{}, string, error) {
			sub, _ := args["subcommand"].(string)
			if sub == "thread" {
				return []interface{}{0x10, 1}, "", nil
			} else if sub == "process" {
				return []interface{}{0x10, 2}, "", nil
			}
			return nil, "", errors.New("subcommand must be 'thread' or 'process'")
		},
		Process: func(resultData []byte, acp int) (string, string, int) {
			if len(resultData) >= 4 {
				method := binary.LittleEndian.Uint32(resultData[:4])
				if method == 1 {
					return "Agent terminated (thread exit)", "", MESSAGE_SUCCESS
				}
				return "Agent terminated (process exit)", "", MESSAGE_SUCCESS
			}
			return "Agent terminated", "", MESSAGE_SUCCESS
		},
	})

	registerCommand("sleep", CommandDef{
		ID: 0x20,
		Create: func(args map[string]any) ([]interface{}, string, error) {
			val, ok := args["val"].(string)
			if !ok {
				return nil, "", errors.New("parameter 'val' must be set")
			}
			sleepSec, err := strconv.Atoi(strings.TrimSuffix(val, "s"))
			if err != nil {
				return nil, "", fmt.Errorf("invalid sleep value: %s", val)
			}
			return []interface{}{0x20, sleepSec}, "", nil
		},
		Process: func(resultData []byte, acp int) (string, string, int) {
			return "Sleep updated", "", MESSAGE_SUCCESS
		},
	})

	registerCommand("whoami", CommandDef{
		ID: 0x30,
		Create: func(args map[string]any) ([]interface{}, string, error) {
			return []interface{}{0x30}, "", nil
		},
		Process: func(resultData []byte, acp int) (string, string, int) {
			if len(resultData) == 0 {
				return "No output", "", MESSAGE_SUCCESS
			}
			msg := ConvertCpToUTF8(string(resultData), acp)
			return msg, "", MESSAGE_SUCCESS
		},
	})
}

func ProcessTasksResult(ts Teamserver, agentData ax.AgentData, baseTask ax.TaskData, data []byte) []ax.TaskData {
	var resultTasks []ax.TaskData

	if len(data) < 38 {
		return resultTasks
	}

	packer := CreatePacker(data)

	for packer.Size() > 38 {
		taskUUID := string(packer.ParsePad(36))
		commandID := packer.ParseInt16()
		resultData := packer.ParseBytes()

		task := baseTask
		task.TaskId = taskUUID[:8]
		task.FinishDate = time.Now().Unix()
		task.Completed = true

		if def, ok := commandByID[commandID]; ok {
			task.Message, task.ClearText, task.MessageType = def.Process(resultData, agentData.ACP)
		} else {
			task.Message = fmt.Sprintf("Unknown command result: 0x%x", commandID)
			task.ClearText = string(resultData)
			task.MessageType = MESSAGE_INFO
		}

		if commandID == 0x10 {
			_ = ts.TsAgentTerminate(agentData.Id, task.TaskId)
		}

		resultTasks = append(resultTasks, task)
	}

	return resultTasks
}
