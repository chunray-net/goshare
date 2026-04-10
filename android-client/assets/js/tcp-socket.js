/**
 * GoShare 安卓客户端 - TCP Socket 连接实现
 * 
 * 由于浏览器安全限制，浏览器无法直接创建原生 TCP 连接。
 * 这个文件提供了几种可能的解决方案：
 * 1. WebSocket 连接（推荐）
 * 2. HTTP API 代理
 * 3. Native App 集成方案
 */

/**
 * TCP Socket 连接管理器
 */
class TcpSocketManager {
    constructor() {
        this.socket = null;
        this.isConnected = false;
        this.isConnecting = false;
        this.reconnectAttempts = 0;
        this.maxReconnectAttempts = 5;
        this.reconnectDelay = 1000;
        
        // 事件回调
        this.onOpen = null;
        this.onClose = null;
        this.onError = null;
        this.onMessage = null;
    }

    /**
     * 连接到服务器
     * @param {string} address - 服务器地址
     * @param {number} port - 端口号
     * @param {Object} options - 连接选项
     */
    async connect(address, port, options = {}) {
        if (this.isConnecting) {
            throw new Error('正在连接中...');
        }

        this.isConnecting = true;
        this.reconnectAttempts = 0;

        try {
            // 尝试不同的连接方式
            if (options.useWebSocket) {
                await this.connectWebSocket(address, port);
            } else if (options.useHttpProxy) {
                await this.connectHttpProxy(address, port);
            } else {
                // 默认使用 WebSocket
                await this.connectWebSocket(address, port);
            }

            this.isConnected = true;
            this.isConnecting = false;
            
            if (this.onOpen) {
                this.onOpen();
            }

        } catch (error) {
            this.isConnecting = false;
            this.isConnected = false;
            
            if (this.onError) {
                this.onError(error);
            }
            
            throw error;
        }
    }

    /**
     * 使用 WebSocket 连接
     * @param {string} address - 服务器地址
     * @param {number} port - 端口号
     */
    connectWebSocket(address, port) {
        return new Promise((resolve, reject) => {
            try {
                const wsUrl = `ws://${address}:${port}`;
                this.socket = new WebSocket(wsUrl);

                this.socket.onopen = () => {
                    resolve();
                };

                this.socket.onclose = (event) => {
                    this.isConnected = false;
                    if (this.onClose) {
                        this.onClose(event);
                    }
                    
                    // 自动重连
                    if (event.code !== 1000 && this.reconnectAttempts < this.maxReconnectAttempts) {
                        this.reconnectAttempts++;
                        setTimeout(() => {
                            this.connect(address, port, { useWebSocket: true });
                        }, this.reconnectDelay * this.reconnectAttempts);
                    }
                };

                this.socket.onerror = (error) => {
                    reject(new Error('WebSocket 连接错误: ' + error.message));
                };

                this.socket.onmessage = (event) => {
                    if (this.onMessage) {
                        this.onMessage(event.data);
                    }
                };

                // 设置连接超时
                setTimeout(() => {
                    if (this.socket.readyState === WebSocket.CONNECTING) {
                        this.socket.close();
                        reject(new Error('WebSocket 连接超时'));
                    }
                }, 10000);

            } catch (error) {
                reject(error);
            }
        });
    }

    /**
     * 使用 HTTP 代理连接
     * @param {string} address - 服务器地址
     * @param {number} port - 端口号
     */
    connectHttpProxy(address, port) {
        return new Promise((resolve, reject) => {
            // 这里需要后端提供 HTTP 代理服务
            // 将 TCP 请求转换为 HTTP 请求
            const proxyUrl = `http://${address}:${port}/proxy`;
            
            // 检查代理服务是否可用
            fetch(proxyUrl + '/ping', {
                method: 'GET',
                timeout: 5000
            })
            .then(response => {
                if (response.ok) {
                    this.socket = {
                        type: 'http-proxy',
                        baseUrl: proxyUrl
                    };
                    resolve();
                } else {
                    reject(new Error('HTTP 代理服务不可用'));
                }
            })
            .catch(error => {
                reject(new Error('HTTP 代理连接失败: ' + error.message));
            });
        });
    }

    /**
     * 发送数据
     * @param {string|Object} data - 要发送的数据
     */
    async send(data) {
        if (!this.isConnected || !this.socket) {
            throw new Error('未连接到服务器');
        }

        try {
            if (this.socket instanceof WebSocket) {
                const message = typeof data === 'string' ? data : JSON.stringify(data);
                this.socket.send(message);
            } else if (this.socket.type === 'http-proxy') {
                await this.sendViaHttpProxy(data);
            }
        } catch (error) {
            throw new Error('发送数据失败: ' + error.message);
        }
    }

    /**
     * 通过 HTTP 代理发送数据
     * @param {string|Object} data - 要发送的数据
     */
    async sendViaHttpProxy(data) {
        const message = typeof data === 'string' ? data : JSON.stringify(data);
        
        const response = await fetch(this.socket.baseUrl + '/send', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                message: message
            })
        });

        if (!response.ok) {
            throw new Error('HTTP 代理发送失败: ' + response.statusText);
        }

        return response.json();
    }

    /**
     * 接收数据
     * @returns {Promise} 接收数据的 Promise
     */
    async receive() {
        return new Promise((resolve, reject) => {
            if (!this.isConnected || !this.socket) {
                reject(new Error('未连接到服务器'));
                return;
            }

            // WebSocket 模式下，数据通过 onmessage 事件接收
            if (this.socket instanceof WebSocket) {
                const originalOnMessage = this.socket.onmessage;
                this.socket.onmessage = (event) => {
                    this.socket.onmessage = originalOnMessage;
                    resolve(event.data);
                };
            } else if (this.socket.type === 'http-proxy') {
                this.receiveViaHttpProxy().then(resolve).catch(reject);
            }
        });
    }

    /**
     * 通过 HTTP 代理接收数据
     */
    async receiveViaHttpProxy() {
        const response = await fetch(this.socket.baseUrl + '/receive', {
            method: 'GET'
        });

        if (!response.ok) {
            throw new Error('HTTP 代理接收失败: ' + response.statusText);
        }

        return response.json();
    }

    /**
     * 断开连接
     */
    disconnect() {
        if (this.socket) {
            if (this.socket instanceof WebSocket) {
                this.socket.close();
            }
            this.socket = null;
        }
        this.isConnected = false;
        this.isConnecting = false;
    }

    /**
     * 重连
     * @param {string} address - 服务器地址
     * @param {number} port - 端口号
     * @param {Object} options - 连接选项
     */
    async reconnect(address, port, options = {}) {
        this.disconnect();
        await new Promise(resolve => setTimeout(resolve, 1000));
        return this.connect(address, port, options);
    }
}

/**
 * GoShare 协议客户端
 */
class GoShareClient {
    constructor() {
        this.socketManager = new TcpSocketManager();
        this.requestId = 0;
        this.pendingRequests = new Map();
        
        // 绑定事件处理器
        this.socketManager.onOpen = this.onSocketOpen.bind(this);
        this.socketManager.onClose = this.onSocketClose.bind(this);
        this.socketManager.onError = this.onSocketError.bind(this);
        this.socketManager.onMessage = this.onSocketMessage.bind(this);
    }

    /**
     * 连接到 GoShare 服务器
     * @param {string} address - 服务器地址
     * @param {number} port - 端口号
     * @param {Object} options - 连接选项
     */
    async connect(address, port, options = {}) {
        try {
            await this.socketManager.connect(address, port, options);
            console.log('GoShare 客户端连接成功');
        } catch (error) {
            console.error('GoShare 客户端连接失败:', error);
            throw error;
        }
    }

    /**
     * 断开连接
     */
    disconnect() {
        this.socketManager.disconnect();
        console.log('GoShare 客户端已断开连接');
    }

    /**
     * 发送请求
     * @param {Object} request - 请求对象
     * @returns {Promise} 响应 Promise
     */
    async sendRequest(request) {
        try {
            // 添加请求 ID 用于匹配响应
            const requestId = ++this.requestId;
            request.id = requestId;

            // 创建 Promise 用于等待响应
            const promise = new Promise((resolve, reject) => {
                this.pendingRequests.set(requestId, { resolve, reject });
                
                // 设置超时
                setTimeout(() => {
                    const pending = this.pendingRequests.get(requestId);
                    if (pending) {
                        this.pendingRequests.delete(requestId);
                        reject(new Error('请求超时'));
                    }
                }, 30000);
            });

            // 发送请求
            await this.socketManager.send(request);
            
            // 等待响应
            const response = await promise;
            return response;

        } catch (error) {
            throw new Error('发送请求失败: ' + error.message);
        }
    }

    /**
     * 列出文件
     * @param {string} path - 路径
     * @returns {Promise} 响应 Promise
     */
    async listFiles(path = '/') {
        return this.sendRequest({
            op: 'list',
            path: path === '/' ? '.' : path
        });
    }

    /**
     * 创建目录
     * @param {string} path - 目录路径
     * @returns {Promise} 响应 Promise
     */
    async createDirectory(path) {
        return this.sendRequest({
            op: 'mkdir',
            path: path
        });
    }

    /**
     * 上传文件
     * @param {string} path - 目标路径
     * @param {string} filename - 文件名
     * @param {Blob|ArrayBuffer} fileData - 文件数据
     * @returns {Promise} 响应 Promise
     */
    async uploadFile(path, filename, fileData) {
        // 注意：由于浏览器限制，大文件上传需要特殊处理
        // 这里只是一个框架，实际实现需要考虑分块上传
        return this.sendRequest({
            op: 'upload',
            path: path,
            name: filename,
            size: fileData.size || fileData.byteLength
        });
    }

    /**
     * 下载文件
     * @param {string} path - 文件路径
     * @param {string} filename - 文件名
     * @returns {Promise} 响应 Promise
     */
    async downloadFile(path, filename) {
        return this.sendRequest({
            op: 'download',
            path: path,
            name: filename
        });
    }

    /**
     * Socket 打开事件处理器
     */
    onSocketOpen() {
        console.log('Socket 连接已打开');
    }

    /**
     * Socket 关闭事件处理器
     */
    onSocketClose(event) {
        console.log('Socket 连接已关闭:', event.code, event.reason);
    }

    /**
     * Socket 错误事件处理器
     */
    onSocketError(error) {
        console.error('Socket 连接错误:', error);
    }

    /**
     * Socket 消息事件处理器
     */
    onSocketMessage(data) {
        try {
            const response = JSON.parse(data);
            
            // 如果有请求 ID，匹配对应的 Promise
            if (response.id) {
                const pending = this.pendingRequests.get(response.id);
                if (pending) {
                    this.pendingRequests.delete(response.id);
                    if (response.ok) {
                        pending.resolve(response);
                    } else {
                        pending.reject(new Error(response.error || '请求失败'));
                    }
                }
            } else {
                // 处理服务器推送的消息
                console.log('收到服务器推送消息:', response);
            }
        } catch (error) {
            console.error('解析服务器消息失败:', error);
        }
    }
}

/**
 * 导出模块
 */
if (typeof module !== 'undefined' && module.exports) {
    module.exports = {
        TcpSocketManager,
        GoShareClient
    };
} else {
    window.TcpSocketManager = TcpSocketManager;
    window.GoShareClient = GoShareClient;
}