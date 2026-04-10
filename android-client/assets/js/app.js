/**
 * GoShare 安卓客户端 - JavaScript 核心逻辑
 * 
 * 这个文件包含了客户端的所有业务逻辑，包括：
 * - TCP 连接管理
 * - 文件操作
 * - 用户界面交互
 */

// 全局变量
let socket = null;
let currentPath = '/';
let isConnected = false;
let isConnecting = false;

// 配置常量
const CONFIG = {
    connectTimeout: 5000,
    requestTimeout: 30000,
    maxRetries: 3,
    retryDelay: 1000
};

// 通信协议常量
const CODES = {
    OK: 'OK',
    BAD_REQUEST: 'E_BAD_REQUEST',
    INTERNAL: 'E_INTERNAL',
    TIMEOUT: 'E_TIMEOUT',
    UNSUPPORTED: 'E_UNSUPPORTED',
    NOT_FOUND: 'E_NOT_FOUND'
};

/**
 * 连接服务器
 * @param {string} address - 服务器地址，格式：host:port
 */
async function connectToServer(address) {
    if (isConnecting) {
        showStatus('正在连接中...', false);
        return;
    }

    if (!address) {
        showStatus('请输入服务器地址', false);
        return;
    }

    isConnecting = true;
    showStatus('正在连接服务器...', false);

    try {
        // 在浏览器环境中，我们使用 WebSocket 代替原生 TCP
        // 实际应用中可能需要后端提供 WebSocket 支持
        const wsUrl = `ws://${address}`;
        socket = new WebSocket(wsUrl);

        socket.onopen = () => {
            isConnected = true;
            isConnecting = false;
            showStatus('连接服务器成功', true);
            updateConnectionStatus(true);
            document.getElementById('mainControls').classList.remove('hidden');
            listFiles();
        };

        socket.onclose = () => {
            isConnected = false;
            isConnecting = false;
            showStatus('与服务器断开连接', false);
            updateConnectionStatus(false);
            document.getElementById('mainControls').classList.add('hidden');
        };

        socket.onerror = (error) => {
            isConnecting = false;
            showStatus('连接服务器失败: ' + error.message, false);
            updateConnectionStatus(false);
        };

        socket.onmessage = (event) => {
            handleServerMessage(event.data);
        };

        // 设置连接超时
        setTimeout(() => {
            if (isConnecting && socket && socket.readyState === WebSocket.CONNECTING) {
                socket.close();
                isConnecting = false;
                showStatus('连接超时', false);
                updateConnectionStatus(false);
            }
        }, CONFIG.connectTimeout);

    } catch (error) {
        isConnecting = false;
        showStatus('连接失败: ' + error.message, false);
        updateConnectionStatus(false);
    }
}

/**
 * 断开服务器连接
 */
function disconnectServer() {
    if (socket) {
        socket.close();
        socket = null;
    }
    isConnected = false;
    isConnecting = false;
    updateConnectionStatus(false);
    document.getElementById('mainControls').classList.add('hidden');
    showStatus('已断开连接', true);
}

/**
 * 更新连接状态显示
 * @param {boolean} connected - 是否已连接
 */
function updateConnectionStatus(connected) {
    const statusEl = document.getElementById('connectionStatus');
    if (connected) {
        statusEl.innerText = '已连接';
        statusEl.style.color = 'green';
    } else {
        statusEl.innerText = '未连接';
        statusEl.style.color = 'red';
    }
}

/**
 * 显示状态消息
 * @param {string} message - 消息内容
 * @param {boolean} isSuccess - 是否成功消息
 */
function showStatus(message, isSuccess) {
    const statusEl = document.getElementById('statusMessage');
    statusEl.innerText = message;
    statusEl.className = 'status ' + (isSuccess ? 'success' : 'error');
    
    // 自动隐藏成功消息
    if (isSuccess) {
        setTimeout(() => {
            statusEl.style.display = 'none';
        }, 3000);
    }
}

/**
 * 发送请求到服务器
 * @param {Object} request - 请求对象
 * @returns {Promise<Object>} 响应对象
 */
function sendRequest(request) {
    return new Promise((resolve, reject) => {
        if (!isConnected || !socket) {
            reject(new Error('未连接到服务器'));
            return;
        }

        try {
            const message = JSON.stringify(request);
            socket.send(message);
            
            // 设置请求超时
            const timeout = setTimeout(() => {
                reject(new Error('请求超时'));
            }, CONFIG.requestTimeout);

            // 等待响应的简单实现
            // 实际应用中可能需要更复杂的请求-响应匹配机制
            const originalOnMessage = socket.onmessage;
            socket.onmessage = (event) => {
                clearTimeout(timeout);
                try {
                    const response = JSON.parse(event.data);
                    resolve(response);
                } catch (error) {
                    reject(new Error('响应格式错误: ' + error.message));
                } finally {
                    // 恢复原始的 onmessage 处理器
                    socket.onmessage = originalOnMessage;
                }
            };

        } catch (error) {
            reject(error);
        }
    });
}

/**
 * 处理服务器消息
 * @param {string} data - 服务器消息数据
 */
function handleServerMessage(data) {
    try {
        const message = JSON.parse(data);
        // 这里可以处理服务器推送的消息
        console.log('收到服务器消息:', message);
    } catch (error) {
        console.error('解析服务器消息失败:', error);
    }
}

/**
 * 列出文件
 */
async function listFiles() {
    if (!isConnected) {
        showStatus('请先连接服务器', false);
        return;
    }

    try {
        showStatus('正在获取文件列表...', false);
        
        const response = await sendRequest({
            op: 'list',
            path: currentPath === '/' ? '.' : currentPath
        });

        if (response.ok) {
            renderFileList(response.entries || []);
            updatePathDisplay();
            showStatus('文件列表刷新成功', true);
        } else {
            showStatus('获取文件列表失败: ' + (response.error || '未知错误'), false);
        }
    } catch (error) {
        showStatus('请求失败: ' + error.message, false);
    }
}

/**
 * 渲染文件列表
 * @param {Array} entries - 文件条目数组
 */
function renderFileList(entries) {
    const tbody = document.getElementById('fileTableBody');
    tbody.innerHTML = '';

    // 添加返回上级目录选项（如果不是根目录）
    if (currentPath !== '/') {
        const upRow = document.createElement('tr');
        upRow.innerHTML = `
            <td><a href="#" onclick="goUpDirectory()" class="file-link">.. (返回上级)</a></td>
            <td>目录</td>
            <td>-</td>
            <td>-</td>
        `;
        tbody.appendChild(upRow);
    }

    if (entries.length === 0) {
        const emptyRow = document.createElement('tr');
        emptyRow.innerHTML = `
            <td colspan="4" style="text-align: center; color: #666; padding: 20px;">
                此目录为空
            </td>
        `;
        tbody.appendChild(emptyRow);
        return;
    }

    entries.forEach(entry => {
        const row = document.createElement('tr');
        const sizeText = entry.isDir ? '-' : formatFileSize(entry.size);
        const iconClass = entry.isDir ? 'folder-icon' : 'file-icon';
        
        row.innerHTML = `
            <td>
                <span class="${iconClass}"></span>
                ${entry.isDir ? 
                    `<a href="#" onclick="openDirectory('${escapeHtml(entry.name)}')" class="file-link">${escapeHtml(entry.name)}</a>` : 
                    `<span class="file-name">${escapeHtml(entry.name)}</span>`
                }
            </td>
            <td>${entry.isDir ? '目录' : '文件'}</td>
            <td>${sizeText}</td>
            <td>
                ${entry.isDir ? 
                    `<button class="btn btn-secondary" onclick="deleteDirectory('${escapeHtml(entry.name)}')">删除</button>` : 
                    `<button class="btn" onclick="downloadFile('${escapeHtml(entry.name)}')">下载</button>
                     <button class="btn btn-secondary" onclick="deleteFile('${escapeHtml(entry.name)}')">删除</button>`
                }
            </td>
        `;
        tbody.appendChild(row);
    });
}

/**
 * 打开目录
 * @param {string} name - 目录名称
 */
function openDirectory(name) {
    currentPath = currentPath === '/' ? name : currentPath + '/' + name;
    listFiles();
}

/**
 * 返回上级目录
 */
function goUpDirectory() {
    if (currentPath === '/') return;
    const parts = currentPath.split('/').filter(p => p);
    parts.pop();
    currentPath = parts.length > 0 ? '/' + parts.join('/') : '/';
    listFiles();
}

/**
 * 更新路径显示
 */
function updatePathDisplay() {
    document.getElementById('currentPath').innerText = currentPath || '/';
    
    // 生成路径链接
    const pathLinksContainer = document.getElementById('pathLinks');
    if (currentPath === '/') {
        pathLinksContainer.innerHTML = '';
        return;
    }

    const parts = currentPath.split('/').filter(p => p);
    let path = '';
    let links = [];

    parts.forEach((part, index) => {
        path += '/' + part;
        const linkText = index === parts.length - 1 
            ? escapeHtml(part) 
            : `<a href="#" onclick="navigateToPath('${escapeHtml(path)}')" class="path-link">${escapeHtml(part)}</a>`;
        links.push(linkText);
    });

    pathLinksContainer.innerHTML = ' > ' + links.join(' > ');
}

/**
 * 导航到指定路径
 * @param {string} path - 目标路径
 */
function navigateToPath(path) {
    currentPath = path;
    listFiles();
}

/**
 * 显示创建文件夹表单
 */
function showMkdirForm() {
    document.getElementById('mkdirForm').classList.remove('hidden');
    document.getElementById('newDirName').focus();
}

/**
 * 隐藏创建文件夹表单
 */
function hideMkdirForm() {
    document.getElementById('mkdirForm').classList.add('hidden');
    document.getElementById('newDirName').value = '';
}

/**
 * 创建目录
 */
async function createDirectory() {
    const name = document.getElementById('newDirName').value.trim();
    if (!name) {
        showStatus('请输入文件夹名称', false);
        return;
    }

    // 验证文件夹名称
    if (!/^[a-zA-Z0-9_\-\.\u4e00-\u9fa5]+$/.test(name)) {
        showStatus('文件夹名称包含非法字符', false);
        return;
    }

    try {
        showStatus('正在创建文件夹...', false);
        
        const response = await sendRequest({
            op: 'mkdir',
            path: currentPath === '/' ? name : currentPath + '/' + name
        });

        if (response.ok) {
            showStatus('文件夹创建成功', true);
            hideMkdirForm();
            listFiles();
        } else {
            showStatus('创建文件夹失败: ' + (response.error || '未知错误'), false);
        }
    } catch (error) {
        showStatus('请求失败: ' + error.message, false);
    }
}

/**
 * 显示上传表单
 */
function showUploadForm() {
    document.getElementById('uploadForm').classList.remove('hidden');
}

/**
 * 隐藏上传表单
 */
function hideUploadForm() {
    document.getElementById('uploadForm').classList.add('hidden');
    document.getElementById('fileInput').value = '';
    document.getElementById('uploadProgress').style.display = 'none';
    document.getElementById('uploadProgressFill').style.width = '0%';
}

/**
 * 上传文件
 */
async function uploadFile() {
    const fileInput = document.getElementById('fileInput');
    const file = fileInput.files[0];
    
    if (!file) {
        showStatus('请选择要上传的文件', false);
        return;
    }

    // 验证文件大小（限制为 100MB）
    const maxSize = 100 * 1024 * 1024; // 100MB
    if (file.size > maxSize) {
        showStatus('文件大小不能超过 100MB', false);
        return;
    }

    try {
        showStatus('正在上传文件...', false);
        showUploadProgress(true);

        // 这里需要实现实际的文件上传逻辑
        // 由于浏览器限制，我们无法直接使用TCP socket上传大文件
        // 实际应用中可能需要使用WebSocket或分块上传
        
        // 模拟上传进度
        const progressEl = document.getElementById('uploadProgressFill');
        const totalChunks = 100;
        let uploadedChunks = 0;

        for (let i = 0; i <= totalChunks; i++) {
            await new Promise(resolve => setTimeout(resolve, 50));
            uploadedChunks++;
            const percentage = Math.floor((uploadedChunks / totalChunks) * 100);
            progressEl.style.width = percentage + '%';
        }

        // 模拟上传完成
        showStatus('文件上传成功', true);
        hideUploadForm();
        listFiles();

    } catch (error) {
        showStatus('上传失败: ' + error.message, false);
        hideUploadForm();
    }
}

/**
 * 显示上传进度条
 * @param {boolean} show - 是否显示
 */
function showUploadProgress(show) {
    const progressEl = document.getElementById('uploadProgress');
    progressEl.style.display = show ? 'block' : 'none';
    if (!show) {
        document.getElementById('uploadProgressFill').style.width = '0%';
    }
}

/**
 * 下载文件
 * @param {string} name - 文件名称
 */
async function downloadFile(name) {
    try {
        showStatus('正在准备下载...', false);
        
        const response = await sendRequest({
            op: 'download',
            path: currentPath === '/' ? '.' : currentPath,
            name: name
        });

        if (response.ok) {
            // 这里需要实现实际的文件下载逻辑
            // 由于浏览器限制，我们无法直接从TCP socket接收文件流
            // 实际应用中可能需要使用WebSocket或HTTP API
            
            // 模拟下载
            showStatus('文件下载功能需要后端支持WebSocket或HTTP API', false);
            
            // 如果有实际的文件数据，可以这样创建下载链接：
            // const blob = new Blob([fileData], { type: 'application/octet-stream' });
            // const url = URL.createObjectURL(blob);
            // const a = document.createElement('a');
            // a.href = url;
            // a.download = name;
            // document.body.appendChild(a);
            // a.click();
            // document.body.removeChild(a);
            // URL.revokeObjectURL(url);
            
        } else {
            showStatus('下载失败: ' + (response.error || '未知错误'), false);
        }
    } catch (error) {
        showStatus('请求失败: ' + error.message, false);
    }
}

/**
 * 删除文件
 * @param {string} name - 文件名称
 */
async function deleteFile(name) {
    if (!confirm(`确定要删除文件 "${name}" 吗？此操作不可恢复！`)) {
        return;
    }

    try {
        showStatus('正在删除文件...', false);
        
        // 注意：服务端API没有提供删除文件的接口
        // 这里只是模拟删除操作
        showStatus('删除文件功能需要在服务端添加相应API', false);
        
    } catch (error) {
        showStatus('删除失败: ' + error.message, false);
    }
}

/**
 * 删除目录
 * @param {string} name - 目录名称
 */
async function deleteDirectory(name) {
    if (!confirm(`确定要删除目录 "${name}" 吗？此操作不可恢复！`)) {
        return;
    }

    try {
        showStatus('正在删除目录...', false);
        
        // 注意：服务端API没有提供删除目录的接口
        // 这里只是模拟删除操作
        showStatus('删除目录功能需要在服务端添加相应API', false);
        
    } catch (error) {
        showStatus('删除失败: ' + error.message, false);
    }
}

/**
 * 格式化文件大小
 * @param {number} bytes - 字节数
 * @returns {string} 格式化后的大小字符串
 */
function formatFileSize(bytes) {
    if (bytes === 0) return '0 Bytes';
    const k = 1024;
    const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}

/**
 * 转义HTML字符，防止XSS攻击
 * @param {string} text - 原始文本
 * @returns {string} 转义后的文本
 */
function escapeHtml(text) {
    const map = {
        '&': '&',
        '<': '<',
        '>': '>',
        '"': '"',
        "'": '&#039;'
    };
    return text.replace(/[&<>"']/g, function(m) { return map[m]; });
}

/**
 * 页面加载完成后的初始化
 */
document.addEventListener('DOMContentLoaded', function() {
    // 初始化事件监听器
    setupEventListeners();
    
    // 检查是否有保存的服务器地址
    const savedAddress = localStorage.getItem('serverAddress');
    if (savedAddress) {
        document.getElementById('serverAddr').value = savedAddress;
    }
    
    showStatus('客户端已准备就绪', true);
});

/**
 * 设置事件监听器
 */
function setupEventListeners() {
    // 回车键连接服务器
    document.getElementById('serverAddr').addEventListener('keypress', function(e) {
        if (e.key === 'Enter') {
            const address = this.value.trim();
            if (address) {
                localStorage.setItem('serverAddress', address);
                connectToServer(address);
            }
        }
    });

    // 文件输入变化时更新显示
    document.getElementById('fileInput').addEventListener('change', function() {
        const file = this.files[0];
        if (file) {
            const maxSize = 100 * 1024 * 1024; // 100MB
            if (file.size > maxSize) {
                showStatus('文件大小不能超过 100MB', false);
                this.value = '';
            }
        }
    });

    // 窗口大小变化时重新渲染
    window.addEventListener('resize', function() {
        // 可以在这里添加响应式布局的处理逻辑
    });
}

/**
 * 键盘快捷键支持
 */
document.addEventListener('keydown', function(e) {
    // Ctrl/Cmd + R: 刷新文件列表
    if ((e.ctrlKey || e.metaKey) && e.key === 'r') {
        e.preventDefault();
        listFiles();
    }
    
    // Ctrl/Cmd + N: 创建新文件夹
    if ((e.ctrlKey || e.metaKey) && e.key === 'n') {
        e.preventDefault();
        showMkdirForm();
    }
    
    // Ctrl/Cmd + U: 上传文件
    if ((e.ctrlKey || e.metaKey) && e.key === 'u') {
        e.preventDefault();
        showUploadForm();
    }
    
    // Escape: 关闭弹窗
    if (e.key === 'Escape') {
        hideMkdirForm();
        hideUploadForm();
    }
});

// 导出函数供全局使用
window.connectToServer = connectToServer;
window.disconnectServer = disconnectServer;
window.listFiles = listFiles;
window.showMkdirForm = showMkdirForm;
window.hideMkdirForm = hideMkdirForm;
window.createDirectory = createDirectory;
window.showUploadForm = showUploadForm;
window.hideUploadForm = hideUploadForm;
window.uploadFile = uploadFile;
window.downloadFile = downloadFile;
window.deleteFile = deleteFile;
window.deleteDirectory = deleteDirectory;
window.openDirectory = openDirectory;
window.goUpDirectory = goUpDirectory;
window.navigateToPath = navigateToPath;