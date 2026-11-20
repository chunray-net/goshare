package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// webEntry 是 HTTP 页面中一张文件/目录卡片的数据。
type webEntry struct {
	Name    string
	IsDir   bool
	Icon    string
	Size    string
	ModTime string
	Link    string
}

type breadcrumb struct {
	Name string
	Path string
}

type pageData struct {
	Path       string
	Crumbs     []breadcrumb
	Entries    []webEntry
	DirCount   int
	FileCount  int
	TotalCount int
	HasParent  bool
	ParentPath string
	RootName   string
	ServerTime string
}

// startHTTPServer 启动 HTTP 文件浏览/下载服务，页面以卡片形式展示文件。
func startHTTPServer(addr, root string) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", webListHandler(root))
	mux.HandleFunc("/download", webDownloadHandler(root))
	mux.HandleFunc("/upload", webUploadHandler(root))

	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
	log.Printf("http file browser on %s, storage=%s", addr, root)
	if err := srv.ListenAndServe(); err != nil {
		log.Printf("http server stopped: %v", err)
	}
}

// cleanWebPath 规范化浏览器传入的 URL 路径，始终返回以 "/" 开头的路径。
func cleanWebPath(raw string) string {
	p := strings.TrimSpace(raw)
	p = strings.ReplaceAll(p, "\\", "/")
	p = path.Clean("/" + p)
	if p == "." || p == "" {
		return "/"
	}
	return p
}

// webRel 把 URL 路径转换为 absPath 需要的相对路径（根目录为空字符串）。
func webRel(p string) string {
	return strings.TrimPrefix(p, "/")
}

func joinWebPath(base, name string) string {
	return path.Join(cleanWebPath(base), name)
}

func buildCrumbs(p string) []breadcrumb {
	crumbs := []breadcrumb{{Name: "根目录", Path: "/"}}
	if p == "/" {
		return crumbs
	}
	current := ""
	for _, seg := range strings.Split(strings.Trim(p, "/"), "/") {
		if seg == "" {
			continue
		}
		current += "/" + seg
		crumbs = append(crumbs, breadcrumb{Name: seg, Path: current})
	}
	return crumbs
}

func humanSize(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for n >= div*unit && exp < 5 {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), "KMGTPE"[exp])
}

func fileIcon(name string) string {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".png", ".jpg", ".jpeg", ".gif", ".bmp", ".webp", ".svg", ".ico":
		return "🖼️"
	case ".mp4", ".mkv", ".avi", ".mov", ".flv", ".webm", ".ts":
		return "🎬"
	case ".mp3", ".wav", ".flac", ".aac", ".ogg", ".m4a":
		return "🎵"
	case ".zip", ".rar", ".7z", ".tar", ".gz", ".bz2", ".xz", ".iso":
		return "📦"
	case ".pdf":
		return "📕"
	case ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx":
		return "📊"
	case ".go", ".js", ".java", ".py", ".c", ".cpp", ".h", ".html", ".css", ".json", ".xml", ".sh", ".bat":
		return "💻"
	case ".txt", ".md", ".log", ".ini", ".conf", ".yaml", ".yml":
		return "📃"
	case ".exe", ".msi", ".apk", ".dmg", ".deb", ".rpm":
		return "⚙️"
	default:
		return "📄"
	}
}

func webListHandler(root string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		current := cleanWebPath(r.URL.Query().Get("path"))
		abs, err := absPath(root, webRel(current))
		if err != nil {
			http.Error(w, "invalid path: "+err.Error(), http.StatusBadRequest)
			return
		}

		infos, err := os.ReadDir(abs)
		if err != nil {
			if os.IsNotExist(err) {
				http.Error(w, "目录不存在", http.StatusNotFound)
				return
			}
			http.Error(w, "读取目录失败: "+err.Error(), http.StatusInternalServerError)
			return
		}

		entries := make([]webEntry, 0, len(infos))
		dirCount, fileCount := 0, 0
		for _, info := range infos {
			fi, statErr := info.Info()
			if statErr != nil {
				continue
			}
			name := info.Name()
			e := webEntry{
				Name:    name,
				IsDir:   info.IsDir(),
				ModTime: fi.ModTime().Format("2006-01-02 15:04"),
			}
			if e.IsDir {
				e.Icon = "📁"
				e.Size = "文件夹"
				e.Link = "/?path=" + url.QueryEscape(joinWebPath(current, name))
				dirCount++
			} else {
				e.Icon = fileIcon(name)
				e.Size = humanSize(fi.Size())
				e.Link = "/download?path=" + url.QueryEscape(current) + "&name=" + url.QueryEscape(name)
				fileCount++
			}
			entries = append(entries, e)
		}

		sort.SliceStable(entries, func(i, j int) bool {
			if entries[i].IsDir != entries[j].IsDir {
				return entries[i].IsDir
			}
			return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name)
		})

		data := pageData{
			Path:       current,
			Crumbs:     buildCrumbs(current),
			Entries:    entries,
			DirCount:   dirCount,
			FileCount:  fileCount,
			TotalCount: len(entries),
			ServerTime: time.Now().Format("2006-01-02 15:04:05"),
		}
		rootAbs, rootErr := absPath(root, "")
		if rootErr == nil {
			data.RootName = filepath.Base(rootAbs)
		}
		if data.RootName == "" || data.RootName == "." {
			data.RootName = "storage"
		}
		if current != "/" {
			parent := path.Dir(current)
			if parent == "." {
				parent = "/"
			}
			data.HasParent = true
			data.ParentPath = parent
		}

		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := pageTemplate.Execute(w, data); err != nil {
			log.Println("render page error:", err)
		}
	}
}

func webDownloadHandler(root string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		current := cleanWebPath(r.URL.Query().Get("path"))
		name := filepath.Base(strings.TrimSpace(r.URL.Query().Get("name")))
		if name == "" || name == "." || name == string(filepath.Separator) {
			http.Error(w, "缺少文件名", http.StatusBadRequest)
			return
		}

		absDir, err := absPath(root, webRel(current))
		if err != nil {
			http.Error(w, "invalid path: "+err.Error(), http.StatusBadRequest)
			return
		}
		filePath := filepath.Join(absDir, name)
		absFile, err := absPath(root, path.Join(webRel(current), name))
		if err != nil || absFile != filePath {
			http.Error(w, "invalid path", http.StatusBadRequest)
			return
		}

		fi, err := os.Stat(absFile)
		if err != nil {
			if os.IsNotExist(err) {
				http.Error(w, "文件不存在", http.StatusNotFound)
				return
			}
			http.Error(w, "读取文件失败: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if fi.IsDir() {
			http.Redirect(w, r, "/?path="+url.QueryEscape(joinWebPath(current, name)), http.StatusFound)
			return
		}

		if cd := mime.FormatMediaType("attachment", map[string]string{"filename": name}); cd != "" {
			w.Header().Set("Content-Disposition", cd)
		}
		http.ServeFile(w, r, absFile)
	}
}

// maxUploadBytes 是单次网页上传请求的总大小上限。
const maxUploadBytes int64 = 10 << 30 // 10 GiB

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Println("write json error:", err)
	}
}

func uploadError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]any{"ok": false, "error": msg})
}

// webUploadHandler 接收浏览器上传的文件，保存到当前浏览的目录。
// 采用流式解析，边接收边写入目标目录的临时文件，不会在系统临时目录再占用一份空间。
func webUploadHandler(root string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", "POST")
			uploadError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		current := cleanWebPath(r.URL.Query().Get("path"))
		absDir, err := absPath(root, webRel(current))
		if err != nil {
			uploadError(w, http.StatusBadRequest, "invalid path: "+err.Error())
			return
		}
		fi, err := os.Stat(absDir)
		if err != nil {
			if os.IsNotExist(err) {
				uploadError(w, http.StatusNotFound, "目录不存在")
				return
			}
			uploadError(w, http.StatusInternalServerError, "读取目录失败: "+err.Error())
			return
		}
		if !fi.IsDir() {
			uploadError(w, http.StatusBadRequest, "目标不是目录")
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)
		mr, err := r.MultipartReader()
		if err != nil {
			uploadError(w, http.StatusBadRequest, "需要 multipart/form-data 上传: "+err.Error())
			return
		}

		saved := make([]string, 0)
		for {
			part, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				if reportUploadLimit(w, err) {
					return
				}
				uploadError(w, http.StatusBadRequest, "读取上传数据失败: "+err.Error())
				return
			}

			name := sanitizeUploadName(part.FileName())
			if name == "" {
				part.Close()
				continue
			}
			dst, err := absPath(root, path.Join(webRel(current), name))
			if err != nil || filepath.Dir(dst) != absDir {
				part.Close()
				uploadError(w, http.StatusBadRequest, "非法文件名: "+part.FileName())
				return
			}
			dst = uniqueUploadPath(dst)
			copyErr := saveUploadedFile(dst, part)
			part.Close()
			if copyErr != nil {
				if reportUploadLimit(w, copyErr) {
					return
				}
				log.Printf("save upload %q failed: %v", part.FileName(), copyErr)
				uploadError(w, http.StatusInternalServerError, "保存 "+name+" 失败: "+copyErr.Error())
				return
			}
			saved = append(saved, filepath.Base(dst))
		}

		if len(saved) == 0 {
			uploadError(w, http.StatusBadRequest, "没有选择有效的文件")
			return
		}
		log.Printf("web upload to %s: %s", current, strings.Join(saved, ", "))
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "saved": saved})
	}
}

// reportUploadLimit 在超过大小上限时写出 413 响应并返回 true。
func reportUploadLimit(w http.ResponseWriter, err error) bool {
	var maxErr *http.MaxBytesError
	if !errors.As(err, &maxErr) {
		return false
	}
	uploadError(w, http.StatusRequestEntityTooLarge, "文件总大小超过限制（"+humanSize(maxUploadBytes)+"）")
	return true
}

// sanitizeUploadName 只保留纯文件名，过滤路径分隔符和 Windows 非法字符。
func sanitizeUploadName(raw string) string {
	name := strings.TrimSpace(raw)
	name = strings.ReplaceAll(name, "\\", "/")
	name = path.Base(name)
	name = strings.TrimSpace(name)
	if name == "" || name == "." || name == ".." || name == "/" {
		return ""
	}
	name = strings.Map(func(r rune) rune {
		if r < 32 || strings.ContainsRune(`<>:"/\|?*`, r) {
			return '_'
		}
		return r
	}, name)
	name = strings.TrimRight(name, " .")
	if name == "" || name == "." || name == ".." {
		return ""
	}
	return name
}

// uniqueUploadPath 为重名文件生成 "name (1).ext"，避免覆盖已有文件。
func uniqueUploadPath(dst string) string {
	if _, err := os.Lstat(dst); os.IsNotExist(err) {
		return dst
	}
	dir, file := filepath.Split(dst)
	ext := filepath.Ext(file)
	base := strings.TrimSuffix(file, ext)
	if base == "" {
		base, ext = file, ""
	}
	for i := 1; i < 10000; i++ {
		candidate := filepath.Join(dir, fmt.Sprintf("%s (%d)%s", base, i, ext))
		if _, err := os.Lstat(candidate); os.IsNotExist(err) {
			return candidate
		}
	}
	return dst
}

// saveUploadedFile 先写入同目录的临时文件，成功后再重命名，避免留下半截文件。
func saveUploadedFile(dst string, src io.Reader) (err error) {
	tmp, err := os.CreateTemp(filepath.Dir(dst), ".goshare-upload-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() {
		tmp.Close()
		if err != nil {
			os.Remove(tmpName)
		}
	}()

	if _, err = io.Copy(tmp, src); err != nil {
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	if err = os.Rename(tmpName, dst); err != nil {
		return err
	}
	return nil
}

var pageTemplate = template.Must(template.New("index").Parse(`<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>GoShare 文件管理</title>
<style>
* { box-sizing: border-box; }
body {
  margin: 0;
  min-height: 100vh;
  font-family: "PingFang SC", "Microsoft YaHei", "Helvetica Neue", Arial, sans-serif;
  color: #1e293b;
  background: linear-gradient(135deg, #eef2ff 0%, #f8fafc 45%, #ecfeff 100%);
}
.wrap { max-width: 1200px; margin: 0 auto; padding: 28px 20px 48px; }
header { margin-bottom: 22px; }
.title-row { display: flex; align-items: center; justify-content: space-between; gap: 16px; flex-wrap: wrap; }
h1 { margin: 0; font-size: 24px; letter-spacing: 1px; }
.subtitle { margin: 6px 0 0; color: #64748b; font-size: 13px; }
.crumbs { display: flex; align-items: center; flex-wrap: wrap; gap: 6px; margin-top: 16px; font-size: 14px; }
.crumbs a { color: #2563eb; text-decoration: none; padding: 4px 10px; border-radius: 999px; background: rgba(37,99,235,.08); }
.crumbs a:hover { background: rgba(37,99,235,.16); }
.crumbs .sep { color: #94a3b8; }
.crumbs .current { color: #475569; padding: 4px 10px; border-radius: 999px; background: rgba(148,163,184,.16); }
.stats { display: flex; gap: 10px; flex-wrap: wrap; margin: 18px 0 22px; }
.badge { font-size: 12px; color: #475569; background: #fff; border: 1px solid rgba(148,163,184,.3); padding: 6px 12px; border-radius: 999px; }
.toolbar { display: flex; align-items: center; gap: 10px; flex-wrap: wrap; }
.tool {
  font: inherit; font-size: 13px; font-weight: 600; line-height: 1;
  color: #1d4ed8; background: #fff; border: 1px solid rgba(37,99,235,.35);
  padding: 10px 16px; border-radius: 10px; cursor: pointer; text-decoration: none;
  transition: background .18s ease, color .18s ease, border-color .18s ease;
}
.tool:hover { background: #2563eb; border-color: #2563eb; color: #fff; }
.auto { display: inline-flex; align-items: center; gap: 6px; font-size: 13px; color: #475569; background: #fff; border: 1px solid rgba(148,163,184,.35); padding: 9px 12px; border-radius: 10px; cursor: pointer; user-select: none; }
.auto input { accent-color: #2563eb; cursor: pointer; }
.uploader { margin: 0 0 18px; background: #fff; border: 1px solid rgba(148,163,184,.22); border-radius: 14px; box-shadow: 0 6px 18px rgba(15,23,42,.06); overflow: hidden; }
.uploader summary { display: flex; align-items: center; gap: 8px; padding: 14px 18px; font-size: 14px; font-weight: 600; color: #1d4ed8; cursor: pointer; user-select: none; list-style: none; }
.uploader summary::-webkit-details-marker { display: none; }
.uploader summary::before { content: "▸"; color: #94a3b8; transition: transform .15s ease; }
.uploader[open] summary::before { transform: rotate(90deg); }
.uploader summary:hover { background: #f8fafc; }
.upload-body { border-top: 1px solid #eef2f7; padding: 16px 18px 18px; }
.dropzone { border: 2px dashed rgba(37,99,235,.35); border-radius: 12px; padding: 22px 16px; text-align: center; color: #475569; background: #f8fbff; transition: background .15s ease, border-color .15s ease; }
.dropzone.dragover { background: #eff6ff; border-color: #2563eb; }
.dropzone p { margin: 0; font-size: 14px; }
.dropzone .hint { margin-top: 8px; font-size: 12px; color: #94a3b8; }
.link-btn { border: none; background: none; padding: 0; font: inherit; color: #2563eb; font-weight: 600; cursor: pointer; text-decoration: underline; }
.selected { list-style: none; margin: 12px 0 0; padding: 0; display: flex; flex-direction: column; gap: 6px; }
.selected:empty { display: none; }
.selected li { display: flex; align-items: center; justify-content: space-between; gap: 12px; font-size: 13px; color: #475569; background: #f8fafc; border-radius: 8px; padding: 7px 12px; }
.selected li .fname { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.selected li .fmeta { flex: none; display: flex; align-items: center; gap: 10px; color: #94a3b8; }
.upload-actions { display: flex; align-items: center; gap: 10px; flex-wrap: wrap; margin-top: 14px; }
.tool.primary { background: #2563eb; border-color: #2563eb; color: #fff; }
.tool.primary:hover { background: #1d4ed8; }
.tool:disabled { opacity: .55; cursor: not-allowed; }
.upload-status { font-size: 13px; color: #475569; }
.progress { height: 6px; margin-top: 12px; background: #e2e8f0; border-radius: 999px; overflow: hidden; }
.progress-bar { width: 0; height: 100%; background: linear-gradient(90deg, #2563eb, #0ea5e9); transition: width .15s linear; }
.list { background: #fff; border: 1px solid rgba(148,163,184,.22); border-radius: 14px; box-shadow: 0 6px 18px rgba(15,23,42,.06); overflow: hidden; }
table { width: 100%; border-collapse: collapse; }
thead th { font-size: 12px; font-weight: 600; color: #64748b; text-align: left; padding: 12px 18px; background: #f8fafc; border-bottom: 1px solid #e2e8f0; white-space: nowrap; }
tbody td { padding: 12px 18px; border-bottom: 1px solid #eef2f7; font-size: 14px; vertical-align: middle; }
tbody tr:last-child td { border-bottom: none; }
tbody tr:hover { background: #f0f7ff; }
.name-cell { display: flex; align-items: center; gap: 10px; min-width: 0; }
.icon { flex: none; font-size: 18px; line-height: 1; }
.file-name { color: #1e293b; font-weight: 500; text-decoration: none; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.file-name:hover { color: #2563eb; text-decoration: underline; }
.size, .time { color: #64748b; font-size: 13px; white-space: nowrap; }
.col-size, .col-time { width: 130px; }
.col-action { width: 110px; text-align: right; }
.btn { display: inline-block; text-decoration: none; font-size: 12px; font-weight: 600; padding: 6px 14px; border-radius: 8px; color: #1d4ed8; background: rgba(37,99,235,.10); transition: background .18s ease, color .18s ease; }
.btn:hover { background: #2563eb; color: #fff; }
.btn.dir { color: #0d9488; background: rgba(13,148,136,.12); }
.btn.dir:hover { background: #0d9488; color: #fff; }
.empty { text-align: center; color: #94a3b8; padding: 56px 20px; }
footer { margin-top: 32px; text-align: center; font-size: 12px; color: #94a3b8; }
@media (max-width: 640px) {
  .wrap { padding: 18px 14px 32px; }
  h1 { font-size: 20px; }
  thead th, tbody td { padding: 10px 12px; }
  .col-time, .time { display: none; }
  .col-size, .col-action { width: 80px; }
}
</style>
</head>
<body>
<div class="wrap">
  <header>
    <div class="title-row">
      <div>
        <h1>GoShare 文件管理</h1>
        <p class="subtitle">根目录：{{.RootName}} · 当前目录：{{.Path}}</p>
      </div>
      <div class="toolbar">
        {{if .HasParent}}<a class="tool" href="/?path={{urlquery .ParentPath}}">↑ 上级目录</a>{{end}}
        <button class="tool" type="button" onclick="goshareReload()">⟳ 刷新</button>
        <label class="auto"><input type="checkbox" id="autoRefresh"> 自动刷新 10s</label>
      </div>
    </div>
    <nav class="crumbs">
      {{range $i, $c := .Crumbs}}
        {{if $i}}<span class="sep">/</span>{{end}}
        {{if eq $c.Path $.Path}}<span class="current">{{$c.Name}}</span>{{else}}<a href="/?path={{urlquery $c.Path}}">{{$c.Name}}</a>{{end}}
      {{end}}
    </nav>
    <div class="stats">
      <span class="badge">目录 {{.DirCount}}</span>
      <span class="badge">文件 {{.FileCount}}</span>
      <span class="badge">共 {{.TotalCount}} 项</span>
    </div>
  </header>
  <details class="uploader" id="uploadPanel" data-path="{{.Path}}">
    <summary>⬆ 上传文件到当前目录</summary>
    <div class="upload-body">
      <div class="dropzone" id="dropZone">
        <p>把文件拖到这里，或 <button type="button" class="link-btn" id="pickBtn">点击选择文件</button></p>
        <p class="hint">支持多选，单次总大小不超过 10 GB；同名文件会自动改名，不会覆盖已有文件。</p>
        <input type="file" id="fileInput" multiple hidden>
      </div>
      <ul class="selected" id="fileList"></ul>
      <div class="upload-actions">
        <button type="button" class="tool primary" id="uploadBtn" disabled>开始上传</button>
        <button type="button" class="tool" id="clearBtn">清空列表</button>
        <span class="upload-status" id="uploadStatus"></span>
      </div>
      <div class="progress"><div class="progress-bar" id="progressBar"></div></div>
    </div>
  </details>
  <main class="list">
    <table>
      <thead>
        <tr>
          <th>名称</th>
          <th class="col-size">大小</th>
          <th class="col-time">修改时间</th>
          <th class="col-action">操作</th>
        </tr>
      </thead>
      <tbody>
        {{range .Entries}}
        <tr>
          <td>
            <div class="name-cell">
              <span class="icon">{{.Icon}}</span>
              <a class="file-name" href="{{.Link}}" title="{{.Name}}">{{.Name}}</a>
            </div>
          </td>
          <td class="size">{{.Size}}</td>
          <td class="time">{{.ModTime}}</td>
          <td class="col-action">{{if .IsDir}}<a class="btn dir" href="{{.Link}}">进入</a>{{else}}<a class="btn" href="{{.Link}}">下载</a>{{end}}</td>
        </tr>
        {{else}}
        <tr><td class="empty" colspan="4">这个目录还是空的，可以在上方上传文件，或用客户端上传。</td></tr>
        {{end}}
      </tbody>
    </table>
  </main>
  <footer>GoShare · 服务时间 {{.ServerTime}}</footer>
</div>
<script>
(function () {
  var uploading = false;
  var KEY = 'goshare.autoRefresh';
  var box = document.getElementById('autoRefresh');

  function reloadURL() {
    var base = location.pathname + location.search;
    return base + (location.search ? '&' : '?') + '_t=' + Date.now();
  }
  window.goshareReload = function () {
    location.replace(reloadURL());
  };

  var panel = document.getElementById('uploadPanel');
  if (panel) {
    var dropZone = document.getElementById('dropZone');
    var input = document.getElementById('fileInput');
    var pickBtn = document.getElementById('pickBtn');
    var uploadBtn = document.getElementById('uploadBtn');
    var clearBtn = document.getElementById('clearBtn');
    var fileList = document.getElementById('fileList');
    var statusEl = document.getElementById('uploadStatus');
    var progressBar = document.getElementById('progressBar');
    var currentPath = panel.getAttribute('data-path') || '/';
    var selected = [];

    function sizeText(n) {
      if (n < 1024) { return n + ' B'; }
      var units = ['KB', 'MB', 'GB', 'TB'];
      var v = n / 1024, i = 0;
      while (v >= 1024 && i < units.length - 1) { v /= 1024; i++; }
      return v.toFixed(1) + ' ' + units[i];
    }

    function renderFiles() {
      fileList.innerHTML = '';
      selected.forEach(function (f, idx) {
        var li = document.createElement('li');
        var name = document.createElement('span');
        name.className = 'fname';
        name.textContent = f.name;
        name.title = f.name;
        var meta = document.createElement('span');
        meta.className = 'fmeta';
        var size = document.createElement('span');
        size.textContent = sizeText(f.size);
        var del = document.createElement('button');
        del.type = 'button';
        del.className = 'link-btn';
        del.textContent = '移除';
        del.addEventListener('click', function () {
          selected.splice(idx, 1);
          renderFiles();
        });
        meta.appendChild(size);
        meta.appendChild(del);
        li.appendChild(name);
        li.appendChild(meta);
        fileList.appendChild(li);
      });
      uploadBtn.disabled = uploading || selected.length === 0;
      clearBtn.disabled = uploading || selected.length === 0;
    }

    function addFiles(list) {
      if (!list || !list.length) { return; }
      for (var i = 0; i < list.length; i++) { selected.push(list[i]); }
      statusEl.textContent = '已选择 ' + selected.length + ' 个文件';
      renderFiles();
    }

    pickBtn.addEventListener('click', function (e) {
      e.stopPropagation();
      input.click();
    });
    dropZone.addEventListener('click', function (e) {
      if (e.target !== pickBtn) { input.click(); }
    });
    input.addEventListener('change', function () {
      addFiles(input.files);
      input.value = '';
    });

    ['dragenter', 'dragover'].forEach(function (evt) {
      dropZone.addEventListener(evt, function (e) {
        e.preventDefault();
        dropZone.classList.add('dragover');
      });
    });
    ['dragleave', 'dragend'].forEach(function (evt) {
      dropZone.addEventListener(evt, function () {
        dropZone.classList.remove('dragover');
      });
    });
    dropZone.addEventListener('drop', function (e) {
      e.preventDefault();
      dropZone.classList.remove('dragover');
      addFiles(e.dataTransfer ? e.dataTransfer.files : null);
    });

    clearBtn.addEventListener('click', function () {
      selected = [];
      statusEl.textContent = '';
      progressBar.style.width = '0%';
      renderFiles();
    });

    uploadBtn.addEventListener('click', function () {
      if (!selected.length || uploading) { return; }
      var fd = new FormData();
      selected.forEach(function (f) { fd.append('files', f, f.name); });

      uploading = true;
      renderFiles();
      uploadBtn.disabled = true;
      progressBar.style.width = '0%';
      statusEl.textContent = '正在上传…';

      var xhr = new XMLHttpRequest();
      xhr.open('POST', '/upload?path=' + encodeURIComponent(currentPath));
      xhr.upload.onprogress = function (e) {
        if (!e.lengthComputable) { return; }
        var pct = Math.round(e.loaded * 100 / e.total);
        progressBar.style.width = pct + '%';
        statusEl.textContent = '正在上传 ' + pct + '%';
      };
      xhr.onload = function () {
        uploading = false;
        var res = null;
        try { res = JSON.parse(xhr.responseText); } catch (err) {}
        if (xhr.status === 200 && res && res.ok) {
          progressBar.style.width = '100%';
          statusEl.textContent = '上传完成：' + res.saved.join('、');
          selected = [];
          renderFiles();
          setTimeout(function () { window.goshareReload(); }, 800);
          return;
        }
        renderFiles();
        statusEl.textContent = '上传失败：' + ((res && res.error) || ('HTTP ' + xhr.status));
      };
      xhr.onerror = function () {
        uploading = false;
        renderFiles();
        statusEl.textContent = '上传失败：网络错误';
      };
      xhr.send(fd);
    });

    renderFiles();
  }

  if (!box) { return; }

  try { box.checked = localStorage.getItem(KEY) === '1'; } catch (e) {}

  var timer = null;
  function schedule() {
    if (timer) { clearTimeout(timer); timer = null; }
    if (!box.checked) { return; }
    timer = setTimeout(function () {
      if (uploading) { schedule(); return; }
      if (document.hidden) { schedule(); return; }
      window.goshareReload();
    }, 10000);
  }

  box.addEventListener('change', function () {
    try { localStorage.setItem(KEY, box.checked ? '1' : '0'); } catch (e) {}
    schedule();
  });

  document.addEventListener('visibilitychange', function () {
    if (!document.hidden && box.checked) { schedule(); }
  });

  schedule();
})();
</script>
</body>
</html>
`))
