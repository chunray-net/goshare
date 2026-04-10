@echo off
echo GoShare 客户端构建脚本
echo ======================

REM 检查 Go 是否已安装
go version >nul 2>&1
if errorlevel 1 (
    echo 错误: 未找到 Go 编译器，请先安装 Go
    pause
    exit /b 1
)

echo 正在构建 GoShare 客户端...

REM 创建输出目录
if not exist "dist" mkdir "dist"

REM 构建 Windows 版本
echo 构建 Windows 版本...
go build -o dist/goshare-client.exe -ldflags "-s -w" client/main.go

if errorlevel 1 (
    echo 构建失败
    pause
    exit /b 1
)

echo 构建成功！
echo 可执行文件已生成到 dist/goshare-client.exe

REM 创建服务管理脚本
echo 创建服务管理脚本...
(
echo @echo off
echo echo GoShare 客户端服务管理脚本
echo echo ============================
echo echo.
echo echo 用法: %%~n0 {install^|uninstall^|start^|stop^|restart^|status}
echo echo.
echo if "%%1"=="" (
echo     echo 请指定操作命令
echo     echo 可用命令:
echo     echo   install   - 安装服务
echo     echo   uninstall - 卸载服务
echo     echo   start     - 启动服务
echo     echo   stop      - 停止服务
echo     echo   restart   - 重启服务
echo     echo   status    - 查看服务状态
echo     exit /b 1
echo )
echo.
echo if "%%1"=="install" (
echo     echo 正在安装服务...
echo     dist\goshare-client.exe -install
echo     goto :end
echo )
echo.
echo if "%%1"=="uninstall" (
echo     echo 正在卸载服务...
echo     dist\goshare-client.exe -uninstall
echo     goto :end
echo )
echo.
echo if "%%1"=="start" (
echo     echo 正在启动服务...
echo     dist\goshare-client.exe -start
echo     goto :end
echo )
echo.
echo if "%%1"=="stop" (
echo     echo 正在停止服务...
echo     dist\goshare-client.exe -stop
echo     goto :end
echo )
echo.
echo if "%%1"=="restart" (
echo     echo 正在重启服务...
echo     dist\goshare-client.exe -stop
echo     timeout /t 2 /nobreak >nul
echo     dist\goshare-client.exe -start
echo     goto :end
echo )
echo.
echo if "%%1"=="status" (
echo     echo 检查服务状态...
echo     sc query GoShareClient
echo     goto :end
echo )
echo.
echo echo 未知命令: %%1
echo exit /b 1
echo.
echo :end
echo echo 操作完成
) > dist\goshare-service.bat

echo.
echo 构建完成！
echo.
echo 使用说明:
echo 1. 双击 dist\goshare-client.exe 运行客户端
echo 2. 使用 dist\goshare-service.bat 管理服务
echo    - 安装服务: goshare-service.bat install
echo    - 启动服务: goshare-service.bat start
echo    - 停止服务: goshare-service.bat stop
echo    - 卸载服务: goshare-service.bat uninstall
echo.
pause