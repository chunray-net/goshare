# GoShare 安卓客户端

这是一个基于 JavaScript 的 GoShare 文件共享服务的安卓客户端实现。

## 功能特性

- 📁 文件浏览：查看服务器上的文件和目录结构
- 📂 目录管理：创建新目录
- 📤 文件上传：上传文件到服务器
- 📥 文件下载：从服务器下载文件
- 🌐 响应式设计：适配手机和平板设备
- 🔗 TCP 连接：与 GoShare 服务器进行通信

## 技术架构

### 前端技术栈
- **HTML5**: 页面结构
- **CSS3**: 样式和布局，使用 CSS Grid 和 Flexbox
- **JavaScript (ES6+)**: 业务逻辑和用户交互
- **WebSocket**: 实时通信（需要后端支持）

### 通信协议
基于 GoShare 服务器的 TCP 通信协议：
- JSON 格式的消息交换
- 支持的操作：list, mkdir, upload, download
- 错误处理和状态码

## 项目结构

```
android-client/
├── index.html          # 主页面
├── README.md           # 项目说明
└── assets/            # 静态资源（图片、图标等）
    ├── css/
    │   └── style.css   # 样式文件
    └── js/
        └── app.js      # 主要 JavaScript 逻辑
```

## 快速开始

### 1. 环境要求
- 现代浏览器（支持 ES6+）
- GoShare 服务器正在运行
- 网络连接到服务器

### 2. 使用方法

1. **打开客户端**
   ```bash
   # 直接在浏览器中打开 index.html 文件
   open android-client/index.html
   ```

2. **连接服务器**
   - 在地址栏输入服务器地址（例如：`192.168.1.100:9000`）
   - 点击"连接服务器"按钮

3. **使用功能**
   - **浏览文件**: 连接成功后自动显示根目录文件列表
   - **创建目录**: 点击"创建文件夹"按钮，输入名称后创建
   - **上传文件**: 点击"上传文件"按钮，选择文件后上传
   - **下载文件**: 点击文件旁的"下载"按钮下载文件

## API 接口说明

### 连接服务器
```javascript
// 连接到指定地址的服务器
connectToServer(address)
```

### 文件操作

#### 列出文件
```javascript
// 请求: {"op": "list", "path": "/path/to/directory"}
// 响应: {"ok": true, "entries": [...]}
listFiles(path)
```

#### 创建目录
```javascript
// 请求: {"op": "mkdir", "path": "/path/to/new/directory"}
// 响应: {"ok": true, "code": "OK"}
createDirectory(path, name)
```

#### 上传文件
```javascript
// 请求: {"op": "upload", "path": "/target/path", "name": "filename", "size": filesize}
// 数据: 文件二进制数据
uploadFile(path, filename, fileData)
```

#### 下载文件
```javascript
// 请求: {"op": "download", "path": "/path/to/file", "name": "filename"}
// 响应: {"ok": true, "size": filesize}
// 数据: 文件二进制数据
downloadFile(path, filename)
```

## 错误处理

客户端实现了完整的错误处理机制：

- **连接错误**: 服务器连接失败时的提示
- **请求错误**: API 请求失败时的错误信息显示
- **文件操作错误**: 上传/下载失败时的错误处理
- **网络超时**: 请求超时时的处理

## 响应式设计

客户端采用响应式设计，适配不同屏幕尺寸：

- **手机端**: 单列布局，大按钮，易于触摸操作
- **平板端**: 适当的间距和布局调整
- **桌面端**: 更大的界面和更丰富的交互

## 开发说明

### 修改样式
样式文件位于 `assets/css/style.css`，使用了现代 CSS 特性：

```css
/* 使用 CSS 变量进行主题管理 */
:root {
    --primary-color: #2196F3;
    --bg-color: #f5f5f5;
    /* ... */
}

/* 响应式设计 */
@media (max-width: 768px) {
    /* 手机端样式 */
}
```

### 添加新功能
在 `assets/js/app.js` 中添加新的功能：

```javascript
// 示例：添加搜索功能
function searchFiles(keyword) {
    // 实现搜索逻辑
}
```

## 已知限制

1. **浏览器限制**: 由于浏览器安全策略，无法直接创建 TCP 连接
   - 解决方案：需要后端提供 WebSocket 或 HTTP API 支持

2. **文件大小限制**: 浏览器对文件上传大小有限制
   - 解决方案：实现分块上传功能

3. **并发限制**: 浏览器对并发连接数有限制
   - 解决方案：使用连接池管理

## 后续改进计划

- [ ] 实现 WebSocket 连接支持
- [ ] 添加文件搜索功能
- [ ] 实现批量文件操作
- [ ] 添加文件预览功能（图片、文本等）
- [ ] 支持文件夹压缩和解压
- [ ] 添加用户认证功能
- [ ] 实现离线缓存
- [ ] 添加主题切换功能

## 贡献指南

欢迎提交 Issue 和 Pull Request！

### 提交 Issue
请详细描述问题，包括：
- 操作系统和浏览器版本
- 复现步骤
- 错误信息

### 提交代码
1. Fork 项目
2. 创建功能分支
3. 提交更改
4. 创建 Pull Request

## 许可证

本项目采用 MIT 许可证。详见 [LICENSE](LICENSE) 文件。

## 联系方式

如有问题或建议，请通过以下方式联系：

- 提交 Issue
- 发送邮件至项目维护者

---

**注意**: 这是一个演示项目，实际生产环境使用时需要考虑更多的安全性和性能优化。