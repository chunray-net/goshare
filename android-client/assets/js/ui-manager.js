/**
 * GoShare 安卓客户端 - UI 管理模块
 * 
 * 这个模块负责管理用户界面的所有交互和显示逻辑
 */

/**
 * UI 管理器类
 */
class UIManager {
    constructor(fileManager) {
        this.fileManager = fileManager;
        this.currentPath = '/';
        
        // 绑定事件处理器
        this.fileManager.onFileListUpdate = this.renderFileList.bind(this);
        this.fileManager.onUploadProgress = this.updateUploadProgress.bind(this);
        this.fileManager.onDownloadProgress = this.updateDownloadProgress.bind(this);
        
        // 初始化 UI 事件监听器
        this.setupEventListeners();
    }

    /**
     * 设置事件监听器
     */
    setupEventListeners() {
        // 连接相关
        document.getElementById('connectBtn')?.addEventListener('click', () => {
            const address = document.getElementById('serverAddr').value.trim();
            if (address) {
                this.connectToServer(address);
            }
        });

        document.getElementById('disconnectBtn')?.addEventListener('click', () => {
            this.disconnectServer();
        });

        // 文件操作相关
        document.getElementById('refreshBtn')?.addEventListener('click', () => {
            this.refreshFileList();
        });

        document.getElementById('mkdirBtn')?.addEventListener('click', () => {
            this.showMkdirForm();
        });

        document.getElementById('uploadBtn')?.addEventListener('click', () => {
            this.showUploadForm();
        });

        // 表单相关
        document.getElementById('createDirBtn')?.addEventListener('click', () => {
            this.createDirectory();
        });

        document.getElementById('cancelDirBtn')?.addEventListener('click', () => {
            this.hideMkdirForm();
        });

        document.getElementById('uploadFileBtn')?.addEventListener('click', () => {
            this.uploadFile();
        });

        document.getElementById('cancelUploadBtn')?.addEventListener('click', () => {
            this.hideUploadForm();
        });

        // 键盘快捷键
        document.addEventListener('keydown', (e) => {
            if ((e.ctrlKey || e.metaKey) && e.key === 'r') {
                e.preventDefault();
                this.refreshFileList();
            } else if ((e.ctrlKey || e.metaKey) && e.key === 'n') {
                e.preventDefault();
                this.showMkdirForm();
            } else if ((e.ctrlKey || e.metaKey) && e.key === 'u') {
                e.preventDefault();
                this.showUploadForm();
            } else if (e.key === 'Escape') {
                this.hideMkdirForm();
                this.hideUploadForm();
            }
        });

        // 文件输入变化
        document.getElementById('fileInput')?.addEventListener('change', (e) => {
            const file = e.target.files[0];
            if (file) {
                this.validateAndShowFileInfo(file);
            }
        });
    }

    /**
     * 连接到服务器
     * @param {string} address - 服务器地址
     */
    async connectToServer(address) {
        try {
            this.showStatus('正在连接服务器...', false);
            await this.fileManager.client.connect(address, 9000);
            
            this.showStatus('连接服务器成功', true);
            this.updateConnectionStatus(true);
            this.showMainControls();
            this.refreshFileList();
            
            // 保存服务器地址
            localStorage.setItem('serverAddress', address);
            
        } catch (error) {
            this.showStatus('连接服务器失败: ' + error.message, false);
            this.updateConnectionStatus(false);
        }
    }

    /**
     * 断开服务器连接
     */
    disconnectServer() {
        this.fileManager.client.disconnect();
        this.updateConnectionStatus(false);
        this.hideMainControls();
        this.showStatus('已断开连接', true);
    }

    /**
     * 刷新文件列表
     */
    async refreshFileList() {
        try {
            this.showStatus('正在刷新文件列表...', false);
            await this.fileManager.listFiles(this.currentPath);
            this.showStatus('文件列表刷新成功', true);
        } catch (error) {
            this.showStatus('刷新文件列表失败: ' + error.message, false);
        }
    }

    /**
     * 显示创建文件夹表单
     */
    showMkdirForm() {
        const form = document.getElementById('mkdirForm');
        if (form) {
            form.classList.remove('hidden');
            document.getElementById('newDirName').focus();
        }
    }

    /**
     * 隐藏创建文件夹表单
     */
    hideMkdirForm() {
        const form = document.getElementById('mkdirForm');
        if (form) {
            form.classList.add('hidden');
            document.getElementById('newDirName').value = '';
        }
    }

    /**
     * 创建目录
     */
    async createDirectory() {
        const name = document.getElementById('newDirName').value.trim();
        if (!name) {
            this.showStatus('请输入文件夹名称', false);
            return;
        }

        try {
            this.showStatus('正在创建文件夹...', false);
            await this.fileManager.createDirectory(name);
            this.hideMkdirForm();
            this.showStatus('文件夹创建成功', true);
        } catch (error) {
            this.showStatus('创建文件夹失败: ' + error.message, false);
        }
    }

    /**
     * 显示上传表单
     */
    showUploadForm() {
        const form = document.getElementById('uploadForm');
        if (form) {
            form.classList.remove('hidden');
        }
    }

    /**
     * 隐藏上传表单
     */
    hideUploadForm() {
        const form = document.getElementById('uploadForm');
        if (form) {
            form.classList.add('hidden');
            document.getElementById('fileInput').value = '';
            this.hideUploadProgress();
        }
    }

    /**
     * 上传文件
     */
    async uploadFile() {
        const fileInput = document.getElementById('fileInput');
        const file = fileInput.files[0];
        
        if (!file) {
            this.showStatus('请选择要上传的文件', false);
            return;
        }

        try {
            this.showStatus('正在上传文件...', false);
            this.showUploadProgress(true);
            
            await this.fileManager.uploadFile(file, (progress, uploaded, total) => {
                this.updateUploadProgress(progress, uploaded, total);
            });
            
            this.hideUploadForm();
            this.showStatus('文件上传成功', true);
            
        } catch (error) {
            this.showStatus('上传文件失败: ' + error.message, false);
            this.hideUploadForm();
        }
    }

    /**
     * 下载文件
     * @param {string} filename - 文件名
     */
    async downloadFile(filename) {
        try {
            this.showStatus('正在准备下载...', false);
            
            const fileData = await this.fileManager.downloadFile(filename, (progress, downloaded, total) => {
                this.updateDownloadProgress(progress, downloaded, total);
            });
            
            // 创建下载链接
            this.createDownloadLink(filename, fileData);
            this.showStatus('文件下载成功', true);
            
        } catch (error) {
            this.showStatus('下载文件失败: ' + error.message, false);
        }
    }

    /**
     * 删除文件
     * @param {string} filename - 文件名
     */
    async deleteFile(filename) {
        if (!confirm(`确定要删除文件 "${filename}" 吗？此操作不可恢复！`)) {
            return;
        }

        try {
            this.showStatus('正在删除文件...', false);
            await this.fileManager.deleteFile(filename);
            this.showStatus('文件删除成功', true);
        } catch (error) {
            this.showStatus('删除文件失败: ' + error.message, false);
        }
    }

    /**
     * 删除目录
     * @param {string} dirname - 目录名
     */
    async deleteDirectory(dirname) {
        if (!confirm(`确定要删除目录 "${dirname}" 吗？此操作不可恢复！`)) {
            return;
        }

        try {
            this.showStatus('正在删除目录...', false);
            await this.fileManager.deleteDirectory(dirname);
            this.showStatus('目录删除成功', true);
        } catch (error) {
            this.showStatus('删除目录失败: ' + error.message, false);
        }
    }

    /**
     * 打开目录
     * @param {string} dirname - 目录名
     */
    async openDirectory(dirname) {
        try {
            await this.fileManager.openDirectory(dirname);
            this.updatePathDisplay();
        } catch (error) {
            this.showStatus('打开目录失败: ' + error.message, false);
        }
    }

    /**
     * 返回上级目录
     */
    async goUpDirectory() {
        try {
            await this.fileManager.goUpDirectory();
            this.updatePathDisplay();
        } catch (error) {
            this.showStatus('返回上级目录失败: ' + error.message, false);
        }
    }

    /**
     * 渲染文件列表
     * @param {Array} fileList - 文件列表
     * @param {string} path - 当前路径
     */
    renderFileList(fileList, path) {
        this.currentPath = path;
        const tbody = document.getElementById('fileTableBody');
        if (!tbody) return;

        tbody.innerHTML = '';

        // 添加返回上级目录选项（如果不是根目录）
        if (path !== '/') {
            const upRow = document.createElement('tr');
            upRow.innerHTML = `
                <td>
                    <span class="folder-icon"></span>
                    <a href="#" onclick="uiManager.goUpDirectory()" class="file-link">.. (返回上级)</a>
                </td>
                <td>目录</td>
                <td>-</td>
                <td>-</td>
            `;
            tbody.appendChild(upRow);
        }

        if (fileList.length === 0) {
            const emptyRow = document.createElement('tr');
            emptyRow.innerHTML = `
                <td colspan="4" class="empty-state">
                    <div>
                        <h3>此目录为空</h3>
                        <p>可以上传文件或创建文件夹</p>
                    </div>
                </td>
            `;
            tbody.appendChild(emptyRow);
            return;
        }

        fileList.forEach(entry => {
            const row = document.createElement('tr');
            const sizeText = entry.isDir ? '-' : this.fileManager.formatFileSize(entry.size);
            const iconClass = this.fileManager.getFileIconClass(entry.name, entry.isDir);
            const typeDesc = this.fileManager.getFileTypeDescription(entry.name, entry.isDir);
            
            row.innerHTML = `
                <td>
                    <span class="${iconClass}"></span>
                    ${entry.isDir ? 
                        `<a href="#" onclick="uiManager.openDirectory('${this.escapeHtml(entry.name)}')" class="file-link">${this.escapeHtml(entry.name)}</a>` : 
                        `<span class="file-name">${this.escapeHtml(entry.name)}</span>`
                    }
                </td>
                <td>${typeDesc}</td>
                <td>${sizeText}</td>
                <td>
                    ${entry.isDir ? 
                        `<button class="btn btn-danger" onclick="uiManager.deleteDirectory('${this.escapeHtml(entry.name)}')">删除</button>` : 
                        `<button class="btn btn-primary" onclick="uiManager.downloadFile('${this.escapeHtml(entry.name)}')">下载</button>
                         <button class="btn btn-danger" onclick="uiManager.deleteFile('${this.escapeHtml(entry.name)}')">删除</button>`
                    }
                </td>
            `;
            tbody.appendChild(row);
        });

        this.updatePathDisplay();
    }

    /**
     * 更新路径显示
     */
    updatePathDisplay() {
        const pathEl = document.getElementById('currentPath');
        const pathLinksEl = document.getElementById('pathLinks');
        
        if (pathEl) {
            pathEl.innerText = this.currentPath || '/';
        }
        
        if (pathLinksEl) {
            if (this.currentPath === '/') {
                pathLinksEl.innerHTML = '';
                return;
            }

            const parts = this.currentPath.split('/').filter(p => p);
            let path = '';
            let links = [];

            parts.forEach((part, index) => {
                path += '/' + part;
                const linkText = index === parts.length - 1 
                    ? this.escapeHtml(part) 
                    : `<a href="#" onclick="uiManager.navigateToPath('${this.escapeHtml(path)}')" class="path-link">${this.escapeHtml(part)}</a>`;
                links.push(linkText);
            });

            pathLinksEl.innerHTML = ' > ' + links.join(' > ');
        }
    }

    /**
     * 导航到指定路径
     * @param {string} path - 目标路径
     */
    async navigateToPath(path) {
        try {
            await this.fileManager.listFiles(path);
            this.updatePathDisplay();
        } catch (error) {
            this.showStatus('导航失败: ' + error.message, false);
        }
    }

    /**
     * 更新上传进度
     * @param {number} progress - 进度百分比
     * @param {number} uploaded - 已上传块数
     * @param {number} total - 总块数
     */
    updateUploadProgress(progress, uploaded, total) {
        const progressEl = document.getElementById('uploadProgressFill');
        if (progressEl) {
            progressEl.style.width = progress + '%';
        }
        
        this.showStatus(`上传进度: ${uploaded}/${total} (${Math.round(progress)}%)`, false);
    }

    /**
     * 更新下载进度
     * @param {number} progress - 进度百分比
     * @param {number} downloaded - 已下载字节数
     * @param {number} total - 总字节数
     */
    updateDownloadProgress(progress, downloaded, total) {
        this.showStatus(`下载进度: ${this.formatBytes(downloaded)}/${this.formatBytes(total)} (${Math.round(progress)}%)`, false);
    }

    /**
     * 显示上传进度条
     * @param {boolean} show - 是否显示
     */
    showUploadProgress(show) {
        const progressEl = document.getElementById('uploadProgress');
        if (progressEl) {
            progressEl.style.display = show ? 'block' : 'none';
            if (!show) {
                document.getElementById('uploadProgressFill').style.width = '0%';
            }
        }
    }

    /**
     * 隐藏上传进度条
     */
    hideUploadProgress() {
        this.showUploadProgress(false);
    }

    /**
     * 显示主控制区域
     */
    showMainControls() {
        const mainControls = document.getElementById('mainControls');
        if (mainControls) {
            mainControls.classList.remove('hidden');
        }
    }

    /**
     * 隐藏主控制区域
     */
    hideMainControls() {
        const mainControls = document.getElementById('mainControls');
        if (mainControls) {
            mainControls.classList.add('hidden');
        }
    }

    /**
     * 更新连接状态显示
     * @param {boolean} connected - 是否已连接
     */
    updateConnectionStatus(connected) {
        const statusEl = document.getElementById('connectionStatus');
        const statusDot = document.getElementById('statusDot');
        
        if (statusEl) {
            statusEl.innerText = connected ? '已连接' : '未连接';
            statusEl.style.color = connected ? 'green' : 'red';
        }
        
        if (statusDot) {
            statusDot.className = connected ? 'status-dot connected' : 'status-dot disconnected';
        }
    }

    /**
     * 显示状态消息
     * @param {string} message - 消息内容
     * @param {boolean} isSuccess - 是否成功消息
     */
    showStatus(message, isSuccess) {
        const statusEl = document.getElementById('statusMessage');
        if (!statusEl) return;

        statusEl.innerText = message;
        statusEl.className = 'status ' + (isSuccess ? 'success' : 'error');
        
        // 自动隐藏成功消息
        if (isSuccess) {
            setTimeout(() => {
                statusEl.style.display = 'none';
            }, 3000);
        } else {
            statusEl.style.display = 'block';
        }
    }

    /**
     * 创建下载链接
     * @param {string} filename - 文件名
     * @param {Blob} fileData - 文件数据
     */
    createDownloadLink(filename, fileData) {
        const url = URL.createObjectURL(fileData);
        const a = document.createElement('a');
        a.href = url;
        a.download = filename;
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        URL.revokeObjectURL(url);
    }

    /**
     * 验证并显示文件信息
     * @param {File} file - 文件对象
     */
    validateAndShowFileInfo(file) {
        const maxSize = 100 * 1024 * 1024; // 100MB
        
        if (file.size > maxSize) {
            this.showStatus('文件大小不能超过 100MB', false);
            document.getElementById('fileInput').value = '';
            return;
        }

        this.showStatus(`准备上传: ${file.name} (${this.formatBytes(file.size)})`, false);
    }

    /**
     * 格式化字节大小
     * @param {number} bytes - 字节数
     * @returns {string} 格式化后的大小字符串
     */
    formatBytes(bytes) {
        return this.fileManager.formatFileSize(bytes);
    }

    /**
     * 转义HTML字符
     * @param {string} text - 原始文本
     * @returns {string} 转义后的文本
     */
    escapeHtml(text) {
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
     * 初始化 UI
     */
    init() {
        // 检查是否有保存的服务器地址
        const savedAddress = localStorage.getItem('serverAddress');
        if (savedAddress) {
            document.getElementById('serverAddr').value = savedAddress;
        }
        
        this.showStatus('客户端已准备就绪', true);
    }
}

/**
 * 导出模块
 */
if (typeof module !== 'undefined' && module.exports) {
    module.exports = UIManager;
} else {
    window.UIManager = UIManager;
}