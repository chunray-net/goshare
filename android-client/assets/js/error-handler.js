/**
 * GoShare 安卓客户端 - 错误处理模块
 * 
 * 这个模块负责处理客户端的所有错误情况，包括：
 * - 网络错误
 * - 服务器错误
 * - 客户端错误
 * - 用户操作错误
 */

/**
 * 错误处理器类
 */
class ErrorHandler {
    constructor() {
        this.errorHistory = [];
        this.maxHistorySize = 100;
        
        // 绑定全局错误处理器
        this.setupGlobalErrorHandling();
    }

    /**
     * 设置全局错误处理
     */
    setupGlobalErrorHandling() {
        // 捕获未处理的 Promise 拒绝
        window.addEventListener('unhandledrejection', (event) => {
            this.handleUnhandledRejection(event);
        });

        // 捕获全局 JavaScript 错误
        window.addEventListener('error', (event) => {
            this.handleGlobalError(event);
        });

        // 捕获资源加载错误
        window.addEventListener('securitypolicyviolation', (event) => {
            this.handleSecurityPolicyViolation(event);
        });
    }

    /**
     * 处理未处理的 Promise 拒绝
     * @param {PromiseRejectionEvent} event - Promise 拒绝事件
     */
    handleUnhandledRejection(event) {
        const error = new Error(`未处理的 Promise 拒绝: ${event.reason}`);
        this.logError(error, 'unhandledrejection');
        
        // 阻止默认行为（控制台错误输出）
        event.preventDefault();
    }

    /**
     * 处理全局 JavaScript 错误
     * @param {ErrorEvent} event - 错误事件
     */
    handleGlobalError(event) {
        const error = new Error(`JavaScript 错误: ${event.message}`);
        error.filename = event.filename;
        error.lineno = event.lineno;
        error.colno = event.colno;
        
        this.logError(error, 'javascript');
    }

    /**
     * 处理安全策略违规
     * @param {SecurityPolicyViolationEvent} event - 安全策略违规事件
     */
    handleSecurityPolicyViolation(event) {
        const error = new Error(`安全策略违规: ${event.violatedDirective}`);
        error.violatedDirective = event.violatedDirective;
        error.blockedURI = event.blockedURI;
        error.sourceFile = event.sourceFile;
        
        this.logError(error, 'security');
    }

    /**
     * 记录错误
     * @param {Error} error - 错误对象
     * @param {string} type - 错误类型
     * @param {Object} context - 上下文信息
     */
    logError(error, type = 'unknown', context = {}) {
        const errorInfo = {
            timestamp: new Date().toISOString(),
            type: type,
            message: error.message,
            stack: error.stack,
            context: context,
            userAgent: navigator.userAgent,
            url: window.location.href
        };

        // 添加到历史记录
        this.errorHistory.push(errorInfo);
        
        // 限制历史记录大小
        if (this.errorHistory.length > this.maxHistorySize) {
            this.errorHistory.shift();
        }

        // 输出到控制台（开发模式）
        if (this.isDevelopment()) {
            console.error('错误详情:', errorInfo);
        }

        // 发送错误报告（如果配置了错误报告服务）
        this.sendErrorReport(errorInfo);
    }

    /**
     * 处理网络错误
     * @param {Error} error - 错误对象
     * @param {Object} context - 上下文信息
     */
    handleNetworkError(error, context = {}) {
        const networkError = new Error(`网络错误: ${error.message}`);
        networkError.originalError = error;
        
        this.logError(networkError, 'network', context);
        
        // 显示用户友好的错误消息
        this.showUserMessage('网络连接失败，请检查网络连接后重试', 'error');
        
        return networkError;
    }

    /**
     * 处理服务器错误
     * @param {Object} response - 服务器响应
     * @param {Object} context - 上下文信息
     */
    handleServerError(response, context = {}) {
        const serverError = new Error(`服务器错误: ${response.error || '未知错误'}`);
        serverError.response = response;
        
        this.logError(serverError, 'server', context);
        
        // 根据错误类型显示不同的用户消息
        const userMessage = this.getServerErrorMessage(response.code);
        this.showUserMessage(userMessage, 'error');
        
        return serverError;
    }

    /**
     * 处理客户端错误
     * @param {Error} error - 错误对象
     * @param {Object} context - 上下文信息
     */
    handleClientError(error, context = {}) {
        const clientError = new Error(`客户端错误: ${error.message}`);
        clientError.originalError = error;
        
        this.logError(clientError, 'client', context);
        
        // 显示用户友好的错误消息
        const userMessage = this.getClientErrorMessage(error);
        this.showUserMessage(userMessage, 'error');
        
        return clientError;
    }

    /**
     * 处理用户操作错误
     * @param {string} message - 错误消息
     * @param {Object} context - 上下文信息
     */
    handleUserError(message, context = {}) {
        const userError = new Error(`用户操作错误: ${message}`);
        
        this.logError(userError, 'user', context);
        
        // 显示用户友好的错误消息
        this.showUserMessage(message, 'warning');
        
        return userError;
    }

    /**
     * 获取服务器错误的用户友好消息
     * @param {string} code - 错误代码
     * @returns {string} 用户友好消息
     */
    getServerErrorMessage(code) {
        const errorMessages = {
            'E_BAD_REQUEST': '请求格式错误，请检查输入参数',
            'E_INTERNAL': '服务器内部错误，请稍后重试',
            'E_TIMEOUT': '请求超时，请检查网络连接',
            'E_UNSUPPORTED': '服务器不支持此操作',
            'E_NOT_FOUND': '请求的资源不存在',
            'E_PERMISSION_DENIED': '权限不足，无法执行此操作',
            'E_QUOTA_EXCEEDED': '存储空间不足',
            'E_FILE_TOO_LARGE': '文件大小超出限制'
        };

        return errorMessages[code] || '服务器错误，请稍后重试';
    }

    /**
     * 获取客户端错误的用户友好消息
     * @param {Error} error - 错误对象
     * @returns {string} 用户友好消息
     */
    getClientErrorMessage(error) {
        const message = error.message.toLowerCase();
        
        if (message.includes('timeout')) {
            return '操作超时，请稍后重试';
        } else if (message.includes('network')) {
            return '网络连接失败，请检查网络连接';
        } else if (message.includes('invalid')) {
            return '输入参数无效，请检查后重试';
        } else if (message.includes('permission')) {
            return '权限不足，无法执行此操作';
        } else {
            return '操作失败，请稍后重试';
        }
    }

    /**
     * 显示用户消息
     * @param {string} message - 消息内容
     * @param {string} type - 消息类型 (success, error, warning, info)
     */
    showUserMessage(message, type = 'info') {
        // 查找状态消息元素
        const statusEl = document.getElementById('statusMessage');
        if (statusEl) {
            statusEl.innerText = message;
            statusEl.className = `status ${type} show`;
            
            // 自动隐藏消息
            if (type === 'success' || type === 'info') {
                setTimeout(() => {
                    statusEl.style.display = 'none';
                }, 5000);
            }
        } else {
            // 如果没有状态元素，使用 alert（仅在开发模式）
            if (this.isDevelopment()) {
                alert(`${type.toUpperCase()}: ${message}`);
            }
        }
    }

    /**
     * 发送错误报告
     * @param {Object} errorInfo - 错误信息
     */
    sendErrorReport(errorInfo) {
        // 这里可以实现错误报告发送逻辑
        // 例如发送到错误跟踪服务（如 Sentry、LogRocket 等）
        
        // 检查是否有配置的错误报告端点
        const errorReportEndpoint = this.getConfiguration('errorReportEndpoint');
        if (errorReportEndpoint) {
            fetch(errorReportEndpoint, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(errorInfo)
            }).catch(reportError => {
                console.warn('发送错误报告失败:', reportError);
            });
        }
    }

    /**
     * 获取配置值
     * @param {string} key - 配置键
     * @returns {any} 配置值
     */
    getConfiguration(key) {
        // 这里可以从全局配置对象或 localStorage 获取配置
        if (window.APP_CONFIG && window.APP_CONFIG[key]) {
            return window.APP_CONFIG[key];
        }
        
        return localStorage.getItem(`config_${key}`);
    }

    /**
     * 检查是否为开发模式
     * @returns {boolean} 是否为开发模式
     */
    isDevelopment() {
        return window.location.hostname === 'localhost' || 
               window.location.hostname === '127.0.0.1' ||
               this.getConfiguration('environment') === 'development';
    }

    /**
     * 获取错误历史
     * @returns {Array} 错误历史记录
     */
    getErrorHistory() {
        return this.errorHistory.slice(); // 返回副本
    }

    /**
     * 清除错误历史
     */
    clearErrorHistory() {
        this.errorHistory = [];
    }

    /**
     * 重试操作
     * @param {Function} operation - 要重试的操作
     * @param {number} maxRetries - 最大重试次数
     * @param {number} delay - 重试延迟（毫秒）
     * @returns {Promise} 操作结果
     */
    async retryOperation(operation, maxRetries = 3, delay = 1000) {
        let lastError;
        
        for (let i = 0; i <= maxRetries; i++) {
            try {
                return await operation();
            } catch (error) {
                lastError = error;
                
                if (i === maxRetries) {
                    // 最后一次尝试失败
                    this.logError(error, 'retry_failed', {
                        maxRetries: maxRetries,
                        attempts: i + 1
                    });
                    throw error;
                }
                
                // 等待后重试
                await new Promise(resolve => setTimeout(resolve, delay * Math.pow(2, i)));
            }
        }
        
        throw lastError;
    }

    /**
     * 创建错误边界包装器
     * @param {Function} component - 组件函数
     * @returns {Function} 包装后的组件
     */
    createErrorBoundary(component) {
        return function(...args) {
            try {
                return component.apply(this, args);
            } catch (error) {
                const errorHandler = new ErrorHandler();
                errorHandler.handleClientError(error, {
                    component: component.name || 'anonymous',
                    arguments: args
                });
                
                // 返回错误状态的 UI
                return `<div class="error-boundary">
                    <h3>组件加载失败</h3>
                    <p>请刷新页面或联系技术支持</p>
                </div>`;
            }
        };
    }

    /**
     * 验证输入数据
     * @param {any} data - 要验证的数据
     * @param {string} type - 数据类型
     * @param {Object} options - 验证选项
     * @returns {boolean} 是否有效
     */
    validateInput(data, type, options = {}) {
        try {
            switch (type) {
                case 'string':
                    if (typeof data !== 'string') {
                        throw new Error('数据类型必须为字符串');
                    }
                    if (options.minLength && data.length < options.minLength) {
                        throw new Error(`字符串长度不能少于 ${options.minLength} 个字符`);
                    }
                    if (options.maxLength && data.length > options.maxLength) {
                        throw new Error(`字符串长度不能超过 ${options.maxLength} 个字符`);
                    }
                    break;
                    
                case 'number':
                    if (typeof data !== 'number' || isNaN(data)) {
                        throw new Error('数据类型必须为数字');
                    }
                    if (options.min && data < options.min) {
                        throw new Error(`数字不能小于 ${options.min}`);
                    }
                    if (options.max && data > options.max) {
                        throw new Error(`数字不能大于 ${options.max}`);
                    }
                    break;
                    
                case 'file':
                    if (!(data instanceof File)) {
                        throw new Error('数据类型必须为文件');
                    }
                    if (options.maxSize && data.size > options.maxSize) {
                        throw new Error(`文件大小不能超过 ${this.formatBytes(options.maxSize)}`);
                    }
                    break;
                    
                default:
                    throw new Error(`不支持的数据类型: ${type}`);
            }
            
            return true;
        } catch (error) {
            this.handleUserError(error.message, {
                dataType: type,
                data: data,
                options: options
            });
            return false;
        }
    }

    /**
     * 格式化字节大小
     * @param {number} bytes - 字节数
     * @returns {string} 格式化后的大小字符串
     */
    formatBytes(bytes) {
        if (bytes === 0) return '0 Bytes';
        const k = 1024;
        const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB'];
        const i = Math.floor(Math.log(bytes) / Math.log(k));
        return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
    }
}

/**
 * 导出模块
 */
if (typeof module !== 'undefined' && module.exports) {
    module.exports = ErrorHandler;
} else {
    window.ErrorHandler = ErrorHandler;
}