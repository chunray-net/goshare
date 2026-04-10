#!/bin/bash

echo "GoShare 客户端构建脚本"
echo "======================"

# 检查 Go 是否已安装
if ! command -v go &> /dev/null; then
    echo "错误: 未找到 Go 编译器，请先安装 Go"
    exit 1
fi

echo "正在构建 GoShare 客户端..."

# 创建输出目录
mkdir -p dist

# 构建 Linux 版本
echo "构建 Linux 版本..."
go build -o dist/goshare-client -ldflags "-s -w" client/main.go

if [ $? -ne 0 ]; then
    echo "构建失败"
    exit 1
fi

echo "构建成功！"
echo "可执行文件已生成到 dist/goshare-client"

# 创建服务管理脚本
echo "创建服务管理脚本..."
cat > dist/goshare-service << 'EOF'
#!/bin/bash

# GoShare 客户端服务管理脚本
# ============================

SERVICE_NAME="goshare-client"

show_usage() {
    echo "用法: $0 {install|uninstall|start|stop|restart|status}"
    echo ""
    echo "可用命令:"
    echo "  install   - 安装服务"
    echo "  uninstall - 卸载服务"
    echo "  start     - 启动服务"
    echo "  stop      - 停止服务"
    echo "  restart   - 重启服务"
    echo "  status    - 查看服务状态"
}

install_service() {
    echo "正在安装服务..."
    
    # 创建服务脚本
    sudo tee /etc/init.d/$SERVICE_NAME > /dev/null << 'SCRIPT'
#!/bin/bash
### BEGIN INIT INFO
# Provides:          goshare-client
# Required-Start:    $remote_fs $syslog
# Required-Stop:     $remote_fs $syslog
# Default-Start:     2 3 4 5
# Default-Stop:      0 1 6
# Short-Description: GoShare Client Service
# Description:       GoShare Client background service
### END INIT INFO

case "$1" in
    start)
        echo "Starting GoShare Client..."
        nohup /usr/local/bin/goshare-client -service > /dev/null 2>&1 &
        echo $! > /var/run/goshare-client.pid
        ;;
    stop)
        echo "Stopping GoShare Client..."
        if [ -f /var/run/goshare-client.pid ]; then
            kill $(cat /var/run/goshare-client.pid)
            rm /var/run/goshare-client.pid
        fi
        ;;
    restart)
        $0 stop
        sleep 2
        $0 start
        ;;
    status)
        if [ -f /var/run/goshare-client.pid ]; then
            if kill -0 $(cat /var/run/goshare-client.pid) 2>/dev/null; then
                echo "GoShare Client is running"
            else
                echo "GoShare Client is not running"
            fi
        else
            echo "GoShare Client is not running"
        fi
        ;;
    *)
        echo "Usage: $0 {start|stop|restart|status}"
        exit 1
        ;;
esac

exit 0
SCRIPT

    # 设置权限
    sudo chmod +x /etc/init.d/$SERVICE_NAME
    
    # 复制可执行文件
    sudo cp goshare-client /usr/local/bin/
    sudo chmod +x /usr/local/bin/goshare-client
    
    # 注册服务
    sudo update-rc.d $SERVICE_NAME defaults
    
    echo "服务安装完成"
}

uninstall_service() {
    echo "正在卸载服务..."
    
    # 停止服务
    sudo service $SERVICE_NAME stop 2>/dev/null
    
    # 卸载服务
    sudo update-rc.d -f $SERVICE_NAME remove 2>/dev/null
    
    # 删除文件
    sudo rm -f /etc/init.d/$SERVICE_NAME
    sudo rm -f /usr/local/bin/goshare-client
    sudo rm -f /var/run/goshare-client.pid 2>/dev/null
    
    echo "服务卸载完成"
}

start_service() {
    echo "正在启动服务..."
    sudo service $SERVICE_NAME start
}

stop_service() {
    echo "正在停止服务..."
    sudo service $SERVICE_NAME stop
}

restart_service() {
    echo "正在重启服务..."
    sudo service $SERVICE_NAME restart
}

status_service() {
    echo "检查服务状态..."
    sudo service $SERVICE_NAME status
}

# 主逻辑
case "$1" in
    install)
        install_service
        ;;
    uninstall)
        uninstall_service
        ;;
    start)
        start_service
        ;;
    stop)
        stop_service
        ;;
    restart)
        restart_service
        ;;
    status)
        status_service
        ;;
    *)
        show_usage
        exit 1
        ;;
esac

echo "操作完成"
EOF

chmod +x dist/goshare-service

echo ""
echo "构建完成！"
echo ""
echo "使用说明:"
echo "1. 运行客户端: ./dist/goshare-client"
echo "2. 使用 ./dist/goshare-service 管理服务"
echo "   - 安装服务: sudo ./dist/goshare-service install"
echo "   - 启动服务: sudo ./dist/goshare-service start"
echo "   - 停止服务: sudo ./dist/goshare-service stop"
echo "   - 卸载服务: sudo ./dist/goshare-service uninstall"
echo ""