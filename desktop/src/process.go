package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"
)

func mustEnv(cmd *exec.Cmd, extra map[string]string) {
	cmd.Env = os.Environ()
	for key, value := range extra {
		cmd.Env = append(cmd.Env, key+"="+value)
	}
}

func startOctop(root string, port int) (*exec.Cmd, error) {
	py := pythonExe(root)
	launch := filepath.Join(root, "launch.py")
	cmd := exec.Command(py, launch, "run", "--host", "127.0.0.1", "--port", strconv.Itoa(port))
	cmd.Dir = root
	mustEnv(cmd, map[string]string{
		"OCTOP_HOME":           octopHome(),
		"OCTOP_GREEN_PACKAGES": filepath.Join(root, "packages"),
		"PYTHONNOUSERSITE":     "1",
		"PYTHONPATH":           "",
	})
	configureProcGroup(cmd)
	// 服务端输出一律落盘（按天滚动）。事故 2026-09-28：Windows 生产模式下输出被
	// 隐式丢弃，服务端首启故障完全不可诊断。日志位置：OCTOP_HOME/logs/server-<日期>.log
	if logw, err := serverLogFile(); err == nil {
		cmd.Stdout = logw
		cmd.Stderr = logw
	} else {
		cmd.Stdout = io.Discard
		cmd.Stderr = io.Discard
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return cmd, nil
}

func stopOctop(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	killProcessTree(cmd)
}

func dashboardURL(port int) string {
	return fmt.Sprintf("http://127.0.0.1:%d/", port)
}

// serverLogFile 返回按天滚动的服务端日志文件（追加模式）。
// 日志目录：OCTOP_HOME/logs/，文件名 server-YYYYMMDD.log。
func serverLogFile() (*os.File, error) {
	dir := filepath.Join(octopHome(), "logs")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return os.OpenFile(filepath.Join(dir, "server-"+time.Now().Format("20060102")+".log"),
		os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
}
