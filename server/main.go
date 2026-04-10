package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	connectTimeout = 5 * time.Second
	requestTimeout = 30 * time.Second

	CodeOK          = "OK"
	CodeBadRequest  = "E_BAD_REQUEST"
	CodeInternal    = "E_INTERNAL"
	CodeTimeout     = "E_TIMEOUT"
	CodeUnsupported = "E_UNSUPPORTED"
	CodeNotFound    = "E_NOT_FOUND"
)

type DirEntry struct {
	Name  string `json:"name"`
	IsDir bool   `json:"isDir"`
	Size  int64  `json:"size"`
}

type Request struct {
	Op   string `json:"op"`
	Path string `json:"path,omitempty"`
	Name string `json:"name,omitempty"`
	Size int64  `json:"size,omitempty"`
}

type Response struct {
	Ok      bool       `json:"ok"`
	Code    string     `json:"code,omitempty"`
	Error   string     `json:"error,omitempty"`
	Entries []DirEntry `json:"entries,omitempty"`
	Size    int64      `json:"size,omitempty"`
}

func main() {
	addr := flag.String("addr", ":9000", "server address")
	storage := flag.String("storage", "./storage", "storage root folder")
	flag.Parse()

	if err := os.MkdirAll(*storage, 0755); err != nil {
		log.Fatalf("create storage path: %v", err)
	}

	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatalf("listen failed: %v", err)
	}
	defer ln.Close()
	fmt.Printf("server on %s, storage=%s\n", *addr, *storage)

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println("accept error:", err)
			continue
		}
		go handleConn(conn, *storage)
	}
}

func absPath(root, rel string) (string, error) {
	cleaned := filepath.Clean("/" + rel)
	abs := filepath.Join(root, cleaned)
	abs, err := filepath.Abs(abs)
	if err != nil {
		return "", err
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(abs, rootAbs+string(os.PathSeparator)) && abs != rootAbs {
		return "", fmt.Errorf("invalid path")
	}
	return abs, nil
}

func handleConn(conn net.Conn, root string) {
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(requestTimeout))
	r := bufio.NewReader(conn)

	line, err := r.ReadBytes('\n')
	if err != nil {
		if err != io.EOF {
			log.Println("read request error:", err)
		}
		return
	}

	var req Request
	if err := json.Unmarshal(line, &req); err != nil {
		sendResponse(conn, Response{Ok: false, Code: CodeBadRequest, Error: "invalid request: " + err.Error()})
		return
	}

	switch req.Op {
	case "list":
		handleList(conn, root, req)
	case "mkdir":
		handleMkdir(conn, root, req)
	case "upload":
		conn.SetDeadline(time.Now().Add(120 * time.Second))
		handleUpload(conn, root, req, r)
	case "download":
		conn.SetDeadline(time.Now().Add(120 * time.Second))
		handleDownload(conn, root, req)
	default:
		sendResponse(conn, Response{Ok: false, Code: CodeUnsupported, Error: "unsupported op"})
	}
}

func sendResponse(w io.Writer, resp Response) {
	b, _ := json.Marshal(resp)
	w.Write(append(b, '\n'))
}

func handleList(conn net.Conn, root string, req Request) {
	path := req.Path
	if path == "" {
		path = "."
	}
	abs, err := absPath(root, path)
	if err != nil {
		sendResponse(conn, Response{Ok: false, Code: CodeBadRequest, Error: err.Error()})
		return
	}
	infos, err := os.ReadDir(abs)
	if err != nil {
		sendResponse(conn, Response{Ok: false, Code: CodeInternal, Error: err.Error()})
		return
	}
	entries := make([]DirEntry, 0, len(infos))
	for _, info := range infos {
		fi, _ := info.Info()
		entries = append(entries, DirEntry{Name: info.Name(), IsDir: info.IsDir(), Size: fi.Size()})
	}
	sendResponse(conn, Response{Ok: true, Code: CodeOK, Entries: entries})
}

func handleMkdir(conn net.Conn, root string, req Request) {
	if req.Path == "" {
		sendResponse(conn, Response{Ok: false, Code: CodeBadRequest, Error: "path required"})
		return
	}
	abs, err := absPath(root, req.Path)
	if err != nil {
		sendResponse(conn, Response{Ok: false, Code: CodeBadRequest, Error: err.Error()})
		return
	}
	if err := os.MkdirAll(abs, 0755); err != nil {
		sendResponse(conn, Response{Ok: false, Code: CodeInternal, Error: err.Error()})
		return
	}
	sendResponse(conn, Response{Ok: true, Code: CodeOK})
}

func handleUpload(conn net.Conn, root string, req Request, r *bufio.Reader) {
	if req.Path == "" || req.Name == "" || req.Size <= 0 {
		sendResponse(conn, Response{Ok: false, Code: CodeBadRequest, Error: "path/name/size required"})
		return
	}
	absdir, err := absPath(root, req.Path)
	if err != nil {
		sendResponse(conn, Response{Ok: false, Code: CodeBadRequest, Error: err.Error()})
		return
	}
	if err := os.MkdirAll(absdir, 0755); err != nil {
		sendResponse(conn, Response{Ok: false, Code: CodeInternal, Error: err.Error()})
		return
	}
	outfile := filepath.Join(absdir, filepath.Base(req.Name))
	f, err := os.Create(outfile)
	if err != nil {
		sendResponse(conn, Response{Ok: false, Code: CodeInternal, Error: err.Error()})
		return
	}
	defer f.Close()

	if _, err := io.CopyN(f, r, req.Size); err != nil {
		sendResponse(conn, Response{Ok: false, Code: CodeInternal, Error: "receive file error: " + err.Error()})
		return
	}

	sendResponse(conn, Response{Ok: true, Code: CodeOK})
}

func handleDownload(conn net.Conn, root string, req Request) {
	if req.Path == "" || req.Name == "" {
		sendResponse(conn, Response{Ok: false, Code: CodeBadRequest, Error: "path/name required"})
		return
	}
	absdir, err := absPath(root, req.Path)
	if err != nil {
		sendResponse(conn, Response{Ok: false, Code: CodeBadRequest, Error: err.Error()})
		return
	}
	filePath := filepath.Join(absdir, filepath.Base(req.Name))
	info, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			sendResponse(conn, Response{Ok: false, Code: CodeNotFound, Error: "file not found"})
		} else {
			sendResponse(conn, Response{Ok: false, Code: CodeInternal, Error: err.Error()})
		}
		return
	}
	f, err := os.Open(filePath)
	if err != nil {
		sendResponse(conn, Response{Ok: false, Code: CodeInternal, Error: err.Error()})
		return
	}
	defer f.Close()

	sendResponse(conn, Response{Ok: true, Code: CodeOK, Size: info.Size()})
	if _, err := io.Copy(conn, f); err != nil {
		log.Println("send file error:", err)
	}
}
