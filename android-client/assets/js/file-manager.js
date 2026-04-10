/**
 * GoShare 安卓客户端 - 文件管理模块
 * 
 * 这个模块负责处理所有与文件操作相关的功能，包括：
 * - 文件列表显示
 * - 文件上传和下载
 * - 目录管理
 * - 文件预览
 */

/**
 * 文件管理器类
 */
class FileManager {
    constructor(client) {
        this.client = client;
        this.currentPath = '/';
        this.fileList = [];
        this.uploadQueue = [];
        this.downloadQueue = [];
        
        // 事件回调
        this.onFileListUpdate = null;
        this.onUploadProgress = null;
        this.onDownloadProgress = null;
    }

    /**
     * 列出文件
     * @param {string} path - 路径
     * @returns {Promise<Array>} 文件列表
     */
    async listFiles(path = '/') {
        try {
            const response = await this.client.listFiles(path);
            
            if (response.ok) {
                this.currentPath = path;
                this.fileList = response.entries || [];
                
                if (this.onFileListUpdate) {
                    this.onFileListUpdate(this.fileList, path);
                }
                
                return this.fileList;
            } else {
                throw new Error(response.error || '获取文件列表失败');
            }
        } catch (error) {
            console.error('列出文件失败:', error);
            throw error;
        }
    }

    /**
     * 创建目录
     * @param {string} name - 目录名称
     * @returns {Promise<boolean>} 是否成功
     */
    async createDirectory(name) {
        try {
            // 验证目录名称
            if (!this.validateDirectoryName(name)) {
                throw new Error('目录名称包含非法字符');
            }

            const fullPath = this.currentPath === '/' ? name : `${this.currentPath}/${name}`;
            const response = await this.client.createDirectory(fullPath);
            
            if (response.ok) {
                // 刷新文件列表
                await this.listFiles(this.currentPath);
                return true;
            } else {
                throw new Error(response.error || '创建目录失败');
            }
        } catch (error) {
            console.error('创建目录失败:', error);
            throw error;
        }
    }

    /**
     * 上传文件
     * @param {File} file - 文件对象
     * @param {Function} onProgress - 进度回调函数
     * @returns {Promise<boolean>} 是否成功
     */
    async uploadFile(file, onProgress = null) {
        try {
            // 验证文件
            if (!this.validateFile(file)) {
                throw new Error('文件无效或过大');
            }

            // 创建上传任务
            const task = {
                id: Date.now(),
                file: file,
                progress: 0,
                status: 'pending'
            };

            this.uploadQueue.push(task);
            
            // 开始上传
            task.status = 'uploading';
            const result = await this.performUpload(task, onProgress);
            
            // 从队列中移除
            const index = this.uploadQueue.indexOf(task);
            if (index > -1) {
                this.uploadQueue.splice(index, 1);
            }
            
            return result;
        } catch (error) {
            console.error('上传文件失败:', error);
            throw error;
        }
    }

    /**
     * 执行文件上传
     * @param {Object} task - 上传任务
     * @param {Function} onProgress - 进度回调
     * @returns {Promise<boolean>} 是否成功
     */
    async performUpload(task, onProgress) {
        const { file } = task;
        
        try {
            // 1. 发送上传请求
            const response = await this.client.uploadFile(
                this.currentPath,
                file.name,
                file.size
            );

            if (!response.ok) {
                throw new Error(response.error || '上传请求失败');
            }

            // 2. 分块上传文件数据
            const chunkSize = 1024 * 1024; // 1MB 每块
            const totalChunks = Math.ceil(file.size / chunkSize);
            let uploadedChunks = 0;

            for (let i = 0; i < totalChunks; i++) {
                const start = i * chunkSize;
                const end = Math.min(start + chunkSize, file.size);
                const chunk = file.slice(start, end);

                // 这里需要实际的文件数据传输
                // 由于浏览器限制，这里只是模拟
                await this.sendFileChunk(chunk, i, totalChunks);
                
                uploadedChunks++;
                task.progress = (uploadedChunks / totalChunks) * 100;
                
                if (onProgress) {
                    onProgress(task.progress, uploadedChunks, totalChunks);
                }
            }

            // 3. 确认上传完成
            return true;

        } catch (error) {
            task.status = 'failed';
            throw error;
        }
    }

    /**
     * 发送文件块（模拟）
     * @param {Blob} chunk - 文件块
     * @param {number} index - 块索引
     * @param {number} total - 总块数
     */
    async sendFileChunk(chunk, index, total) {
        // 模拟网络延迟
        await new Promise(resolve => setTimeout(resolve, 100));
        
        // 这里应该实际发送文件数据到服务器
        console.log(`发送文件块 ${index + 1}/${total}`);
    }

    /**
     * 下载文件
     * @param {string} filename - 文件名
     * @param {Function} onProgress - 进度回调函数
     * @returns {Promise<Blob>} 文件数据
     */
    async downloadFile(filename, onProgress = null) {
        try {
            // 发送下载请求
            const response = await this.client.downloadFile(this.currentPath, filename);
            
            if (!response.ok) {
                throw new Error(response.error || '下载请求失败');
            }

            // 创建下载任务
            const task = {
                id: Date.now(),
                filename: filename,
                size: response.size,
                downloaded: 0,
                status: 'downloading'
            };

            this.downloadQueue.push(task);

            try {
                // 接收文件数据
                const fileData = await this.receiveFileData(task, onProgress);
                
                // 从队列中移除
                const index = this.downloadQueue.indexOf(task);
                if (index > -1) {
                    this.downloadQueue.splice(index, 1);
                }
                
                return fileData;
            } catch (error) {
                task.status = 'failed';
                throw error;
            }

        } catch (error) {
            console.error('下载文件失败:', error);
            throw error;
        }
    }

    /**
     * 接收文件数据
     * @param {Object} task - 下载任务
     * @param {Function} onProgress - 进度回调
     * @returns {Promise<Blob>} 文件数据
     */
    async receiveFileData(task, onProgress) {
        const chunks = [];
        const chunkSize = 1024 * 1024; // 1MB 每块
        let totalReceived = 0;

        while (totalReceived < task.size) {
            // 这里需要实际接收文件数据
            // 由于浏览器限制，这里只是模拟
            const chunk = await this.receiveFileChunk(chunkSize);
            chunks.push(chunk);
            
            totalReceived += chunk.byteLength;
            task.downloaded = totalReceived;
            
            const progress = (totalReceived / task.size) * 100;
            
            if (onProgress) {
                onProgress(progress, totalReceived, task.size);
            }
        }

        // 合并所有数据块
        return new Blob(chunks);
    }

    /**
     * 接收文件块（模拟）
     * @param {number} size - 块大小
     * @returns {Promise<ArrayBuffer>} 文件块数据
     */
    async receiveFileChunk(size) {
        // 模拟网络延迟
        await new Promise(resolve => setTimeout(resolve, 100));
        
        // 创建模拟数据
        const buffer = new ArrayBuffer(size);
        const view = new Uint8Array(buffer);
        for (let i = 0; i < size; i++) {
            view[i] = Math.floor(Math.random() * 256);
        }
        
        return buffer;
    }

    /**
     * 删除文件（模拟）
     * @param {string} filename - 文件名
     * @returns {Promise<boolean>} 是否成功
     */
    async deleteFile(filename) {
        try {
            // 注意：服务端API没有提供删除文件的接口
            // 这里只是模拟删除操作
            console.log(`模拟删除文件: ${filename}`);
            
            // 刷新文件列表
            await this.listFiles(this.currentPath);
            return true;
        } catch (error) {
            console.error('删除文件失败:', error);
            throw error;
        }
    }

    /**
     * 删除目录（模拟）
     * @param {string} dirname - 目录名
     * @returns {Promise<boolean>} 是否成功
     */
    async deleteDirectory(dirname) {
        try {
            // 注意：服务端API没有提供删除目录的接口
            // 这里只是模拟删除操作
            console.log(`模拟删除目录: ${dirname}`);
            
            // 刷新文件列表
            await this.listFiles(this.currentPath);
            return true;
        } catch (error) {
            console.error('删除目录失败:', error);
            throw error;
        }
    }

    /**
     * 打开目录
     * @param {string} dirname - 目录名
     * @returns {Promise<Array>} 文件列表
     */
    async openDirectory(dirname) {
        const newPath = this.currentPath === '/' ? dirname : `${this.currentPath}/${dirname}`;
        return this.listFiles(newPath);
    }

    /**
     * 返回上级目录
     * @returns {Promise<Array>} 文件列表
     */
    async goUpDirectory() {
        if (this.currentPath === '/') {
            return this.fileList;
        }

        const parts = this.currentPath.split('/').filter(p => p);
        parts.pop();
        const newPath = parts.length > 0 ? '/' + parts.join('/') : '/';
        
        return this.listFiles(newPath);
    }

    /**
     * 获取当前路径
     * @returns {string} 当前路径
     */
    getCurrentPath() {
        return this.currentPath;
    }

    /**
     * 获取文件列表
     * @returns {Array} 文件列表
     */
    getFileList() {
        return this.fileList;
    }

    /**
     * 获取上传队列
     * @returns {Array} 上传队列
     */
    getUploadQueue() {
        return this.uploadQueue;
    }

    /**
     * 获取下载队列
     * @returns {Array} 下载队列
     */
    getDownloadQueue() {
        return this.downloadQueue;
    }

    /**
     * 验证目录名称
     * @param {string} name - 目录名称
     * @returns {boolean} 是否有效
     */
    validateDirectoryName(name) {
        // 只允许字母、数字、下划线、连字符、点和中文字符
        const pattern = /^[a-zA-Z0-9_\-\.\u4e00-\u9fa5]+$/;
        return pattern.test(name) && name.length > 0 && name.length <= 255;
    }

    /**
     * 验证文件
     * @param {File} file - 文件对象
     * @returns {boolean} 是否有效
     */
    validateFile(file) {
        if (!file) return false;
        
        // 检查文件大小（限制为 100MB）
        const maxSize = 100 * 1024 * 1024; // 100MB
        if (file.size > maxSize) {
            return false;
        }
        
        // 检查文件名
        if (!file.name || file.name.length === 0) {
            return false;
        }
        
        return true;
    }

    /**
     * 格式化文件大小
     * @param {number} bytes - 字节数
     * @returns {string} 格式化后的大小字符串
     */
    formatFileSize(bytes) {
        if (bytes === 0) return '0 Bytes';
        const k = 1024;
        const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB'];
        const i = Math.floor(Math.log(bytes) / Math.log(k));
        return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
    }

    /**
     * 获取文件图标类名
     * @param {string} filename - 文件名
     * @param {boolean} isDir - 是否为目录
     * @returns {string} 图标类名
     */
    getFileIconClass(filename, isDir) {
        if (isDir) {
            return 'folder-icon';
        }

        const extension = filename.split('.').pop().toLowerCase();
        const iconMap = {
            'txt': 'file-text-icon',
            'pdf': 'file-pdf-icon',
            'doc': 'file-doc-icon',
            'docx': 'file-doc-icon',
            'xls': 'file-xls-icon',
            'xlsx': 'file-xls-icon',
            'ppt': 'file-ppt-icon',
            'pptx': 'file-ppt-icon',
            'jpg': 'file-image-icon',
            'jpeg': 'file-image-icon',
            'png': 'file-image-icon',
            'gif': 'file-image-icon',
            'mp3': 'file-audio-icon',
            'wav': 'file-audio-icon',
            'mp4': 'file-video-icon',
            'avi': 'file-video-icon',
            'zip': 'file-archive-icon',
            'rar': 'file-archive-icon'
        };

        return iconMap[extension] || 'file-icon';
    }

    /**
     * 获取文件类型描述
     * @param {string} filename - 文件名
     * @param {boolean} isDir - 是否为目录
     * @returns {string} 类型描述
     */
    getFileTypeDescription(filename, isDir) {
        if (isDir) {
            return '文件夹';
        }

        const extension = filename.split('.').pop().toLowerCase();
        const typeMap = {
            'txt': '文本文档',
            'pdf': 'PDF文档',
            'doc': 'Word文档',
            'docx': 'Word文档',
            'xls': 'Excel文档',
            'xlsx': 'Excel文档',
            'ppt': 'PowerPoint文档',
            'pptx': 'PowerPoint文档',
            'jpg': '图片文件',
            'jpeg': '图片文件',
            'png': '图片文件',
            'gif': '图片文件',
            'mp3': '音频文件',
            'wav': '音频文件',
            'mp4': '视频文件',
            'avi': '视频文件',
            'zip': '压缩文件',
            'rar': '压缩文件'
        };

        return typeMap[extension] || '文件';
    }
}

/**
 * 导出模块
 */
if (typeof module !== 'undefined' && module.exports) {
    module.exports = FileManager;
} else {
    window.FileManager = FileManager;
}