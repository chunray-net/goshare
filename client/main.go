package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"github.com/getlantern/systray"
)

var (
	// 命令行参数
	serviceMode = flag.Bool("service", false, "运行为后台服务")
	install     = flag.Bool("install", false, "安装为系统服务")
	uninstall   = flag.Bool("uninstall", false, "卸载系统服务")
	start       = flag.Bool("start", false, "启动服务")
	stop        = flag.Bool("stop", false, "停止服务")
)

func main() {
	flag.Parse()

	// 检查服务管理命令
	if *install {
		if err := installServiceWin(); err != nil {
			log.Fatal("安装服务失败:", err)
		}
		fmt.Println("服务安装成功")
		return
	}

	if *uninstall {
		if err := uninstallService(); err != nil {
			log.Fatal("卸载服务失败:", err)
		}
		fmt.Println("服务卸载成功")
		return
	}

	if *start {
		if err := startService(); err != nil {
			log.Fatal("启动服务失败:", err)
		}
		fmt.Println("服务启动成功")
		return
	}

	if *stop {
		if err := stopService(); err != nil {
			log.Fatal("停止服务失败:", err)
		}
		fmt.Println("服务停止成功")
		return
	}

	// 检查是否已在运行
	if isAlreadyRunning() {
		fmt.Println("GoShare 客户端已在运行")
		return
	}

	// 如果是服务模式，以守护进程方式运行
	if *serviceMode {
		runAsService()
		return
	}

	// GUI 模式
	runGUI()
}

func runGUI() {
	a := app.NewWithID("chunray.net")
	w := a.NewWindow("GoShare Fyne 客户端")
	w.Resize(fyne.NewSize(360, 480))

	setupSystray(w, a)

	ui := NewUI(a, w)
	ui.Run()
}

func runAsService() {
	// 后台服务模式
	log.Println("GoShare 服务启动")

	// 创建系统托盘图标
	setupServiceSystray()

	// 保持服务运行
	select {}
}

func setupServiceSystray() {
	go func() {
		systray.Run(func() {
			iconBytes := iconData()
			if len(iconBytes) == 0 {
				log.Println("systray warning: icon bytes empty, skip icon")
			} else {
				systray.SetIcon(iconBytes)
			}

			mShow := systray.AddMenuItem("显示客户端", "显示并激活 GoShare 客户端")
			mExit := systray.AddMenuItem("退出服务", "停止并退出服务")

			go func() {
				for {
					select {
					case <-mShow.ClickedCh:
						// 启动 GUI 模式
						cmd := exec.Command(os.Args[0])
						cmd.Start()
					case <-mExit.ClickedCh:
						systray.Quit()
						os.Exit(0)
					}
				}
			}()
		}, func() {
			// onExit
			log.Println("GoShare 服务退出")
		})
	}()
}

func isAlreadyRunning() bool {
	// 检查是否有同名进程在运行
	// 这里简化处理，实际可以使用进程锁或检查特定端口
	return false
}

func installService() error {
	// 安装服务的逻辑
	// 在 Windows 上，这通常涉及创建注册表项
	return nil
}

func uninstallService() error {
	// 卸载服务的逻辑
	return nil
}

func startService() error {
	// 启动服务
	cmd := exec.Command(os.Args[0], "-service")
	return cmd.Start()
}

func stopService() error {
	// 停止服务的逻辑
	return nil
}
