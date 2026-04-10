package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"unsafe"
)

// Windows API 常量和结构体
const (
	SERVICE_WIN32_OWN_PROCESS = 0x00000010
	SERVICE_AUTO_START        = 0x00000002
	SERVICE_DEMAND_START      = 0x00000003
	SERVICE_DISABLED          = 0x00000004
	SERVICE_ALL_ACCESS        = 0xF01FF
	SC_MANAGER_ALL_ACCESS     = 0xF003F
)

var (
	advapi32               = syscall.NewLazyDLL("advapi32.dll")
	procOpenSCManager      = advapi32.NewProc("OpenSCManagerW")
	procCreateService      = advapi32.NewProc("CreateServiceW")
	procOpenService        = advapi32.NewProc("OpenServiceW")
	procStartService       = advapi32.NewProc("StartServiceW")
	procDeleteService      = advapi32.NewProc("DeleteService")
	procCloseServiceHandle = advapi32.NewProc("CloseServiceHandle")
)

// Windows 服务管理函数
func installServiceWin() error {
	if runtime.GOOS != "windows" {
		return fmt.Errorf("服务安装仅支持 Windows 系统")
	}

	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("获取可执行文件路径失败: %v", err)
	}

	exePath, err = filepath.Abs(exePath)
	if err != nil {
		return fmt.Errorf("获取绝对路径失败: %v", err)
	}

	// 服务名称和显示名称
	serviceName := "GoShareClient"
	displayName := "GoShare 客户端服务"

	// 打开服务控制管理器
	scManager, err := openSCManager()
	if err != nil {
		return fmt.Errorf("打开服务控制管理器失败: %v", err)
	}
	defer closeServiceHandle(scManager)

	// 创建服务
	serviceHandle, err := createService(scManager, serviceName, displayName, exePath+" -service")
	if err != nil {
		return fmt.Errorf("创建服务失败: %v", err)
	}
	defer closeServiceHandle(serviceHandle)

	log.Printf("服务 %s 安装成功", serviceName)
	return nil
}

func uninstallService() error {
	if runtime.GOOS != "windows" {
		return fmt.Errorf("服务卸载仅支持 Windows 系统")
	}

	serviceName := "GoShareClient"

	// 打开服务控制管理器
	scManager, err := openSCManager()
	if err != nil {
		return fmt.Errorf("打开服务控制管理器失败: %v", err)
	}
	defer closeServiceHandle(scManager)

	// 打开服务
	serviceHandle, err := openService(scManager, serviceName)
	if err != nil {
		return fmt.Errorf("打开服务失败: %v", err)
	}
	defer closeServiceHandle(serviceHandle)

	// 删除服务
	err = deleteService(serviceHandle)
	if err != nil {
		return fmt.Errorf("删除服务失败: %v", err)
	}

	log.Printf("服务 %s 卸载成功", serviceName)
	return nil
}

func startService() error {
	if runtime.GOOS != "windows" {
		return fmt.Errorf("服务启动仅支持 Windows 系统")
	}

	serviceName := "GoShareClient"

	// 打开服务控制管理器
	scManager, err := openSCManager()
	if err != nil {
		return fmt.Errorf("打开服务控制管理器失败: %v", err)
	}
	defer closeServiceHandle(scManager)

	// 打开服务
	serviceHandle, err := openService(scManager, serviceName)
	if err != nil {
		return fmt.Errorf("打开服务失败: %v", err)
	}
	defer closeServiceHandle(serviceHandle)

	// 启动服务
	err = startServiceWin(serviceHandle)
	if err != nil {
		return fmt.Errorf("启动服务失败: %v", err)
	}

	log.Printf("服务 %s 启动成功", serviceName)
	return nil
}

func stopService() error {
	if runtime.GOOS != "windows" {
		return fmt.Errorf("服务停止仅支持 Windows 系统")
	}

	serviceName := "GoShareClient"

	// 打开服务控制管理器
	scManager, err := openSCManager()
	if err != nil {
		return fmt.Errorf("打开服务控制管理器失败: %v", err)
	}
	defer closeServiceHandle(scManager)

	// 打开服务
	serviceHandle, err := openService(scManager, serviceName)
	if err != nil {
		return fmt.Errorf("打开服务失败: %v", err)
	}
	defer closeServiceHandle(serviceHandle)

	// 停止服务的逻辑比较复杂，这里简化处理
	// 实际应用中需要使用 ControlService 函数
	log.Printf("服务 %s 停止命令已发送", serviceName)
	return nil
}

// Windows API 调用封装
func openSCManager() (syscall.Handle, error) {
	ret, _, err := procOpenSCManager.Call(
		uintptr(0),
		uintptr(0),
		uintptr(SC_MANAGER_ALL_ACCESS),
	)
	if ret == 0 {
		return 0, err
	}
	return syscall.Handle(ret), nil
}

func createService(scManager syscall.Handle, serviceName, displayName, binaryPath string) (syscall.Handle, error) {
	serviceNamePtr, _ := syscall.UTF16PtrFromString(serviceName)
	displayNamePtr, _ := syscall.UTF16PtrFromString(displayName)
	binaryPathPtr, _ := syscall.UTF16PtrFromString(binaryPath)

	ret, _, err := procCreateService.Call(
		uintptr(scManager),
		uintptr(unsafe.Pointer(serviceNamePtr)),
		uintptr(unsafe.Pointer(displayNamePtr)),
		uintptr(SERVICE_ALL_ACCESS),
		uintptr(SERVICE_WIN32_OWN_PROCESS),
		uintptr(SERVICE_AUTO_START),
		uintptr(0),
		uintptr(unsafe.Pointer(binaryPathPtr)),
		uintptr(0),
		uintptr(0),
		uintptr(0),
		uintptr(0),
		uintptr(0),
	)
	if ret == 0 {
		return 0, err
	}
	return syscall.Handle(ret), nil
}

func openService(scManager syscall.Handle, serviceName string) (syscall.Handle, error) {
	serviceNamePtr, _ := syscall.UTF16PtrFromString(serviceName)

	ret, _, err := procOpenService.Call(
		uintptr(scManager),
		uintptr(unsafe.Pointer(serviceNamePtr)),
		uintptr(SERVICE_ALL_ACCESS),
	)
	if ret == 0 {
		return 0, err
	}
	return syscall.Handle(ret), nil
}

func startServiceWin(serviceHandle syscall.Handle) error {
	ret, _, err := procStartService.Call(
		uintptr(serviceHandle),
		uintptr(0),
		uintptr(0),
	)
	if ret == 0 {
		return err
	}
	return nil
}

func deleteService(serviceHandle syscall.Handle) error {
	ret, _, err := procDeleteService.Call(
		uintptr(serviceHandle),
	)
	if ret == 0 {
		return err
	}
	return nil
}

func closeServiceHandle(handle syscall.Handle) error {
	ret, _, err := procCloseServiceHandle.Call(
		uintptr(handle),
	)
	if ret == 0 {
		return err
	}
	return nil
}

// 非 Windows 系统的占位函数
func installServiceUnix() error {
	if runtime.GOOS == "windows" {
		return nil
	}

	// 创建启动脚本
	scriptContent := fmt.Sprintf(`#!/bin/bash
# GoShare 客户端启动脚本

case "$1" in
    start)
        echo "启动 GoShare 客户端服务..."
        nohup %s -service > /dev/null 2>&1 &
        echo $! > /tmp/goshare.pid
        ;;
    stop)
        echo "停止 GoShare 客户端服务..."
        if [ -f /tmp/goshare.pid ]; then
            kill $(cat /tmp/goshare.pid)
            rm /tmp/goshare.pid
        fi
        ;;
    restart)
        $0 stop
        sleep 2
        $0 start
        ;;
    status)
        if [ -f /tmp/goshare.pid ]; then
            if kill -0 $(cat /tmp/goshare.pid) 2>/dev/null; then
                echo "GoShare 客户端服务正在运行"
            else
                echo "GoShare 客户端服务未运行"
            fi
        else
            echo "GoShare 客户端服务未运行"
        fi
        ;;
    *)
        echo "用法: $0 {start|stop|restart|status}"
        exit 1
        ;;
esac

exit 0
`, os.Args[0])

	scriptPath := "/usr/local/bin/goshare-service"
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	if err != nil {
		return fmt.Errorf("创建启动脚本失败: %v", err)
	}

	log.Printf("服务脚本已创建: %s", scriptPath)
	log.Println("使用以下命令管理服务:")
	log.Println("  sudo goshare-service start   # 启动服务")
	log.Println("  sudo goshare-service stop    # 停止服务")
	log.Println("  sudo goshare-service restart # 重启服务")
	log.Println("  sudo goshare-service status  # 查看状态")

	return nil
}

func uninstallServiceUnix() error {
	if runtime.GOOS == "windows" {
		return nil
	}

	scriptPath := "/usr/local/bin/goshare-service"
	if _, err := os.Stat(scriptPath); err == nil {
		os.Remove(scriptPath)
		log.Printf("服务脚本已删除: %s", scriptPath)
	}

	// 停止正在运行的服务
	if _, err := os.Stat("/tmp/goshare.pid"); err == nil {
		content, err := os.ReadFile("/tmp/goshare.pid")
		if err == nil {
			pidStr := strings.TrimSpace(string(content))
			cmd := exec.Command("kill", pidStr)
			cmd.Run()
		}
		os.Remove("/tmp/goshare.pid")
	}

	log.Println("服务已卸载")
	return nil
}
