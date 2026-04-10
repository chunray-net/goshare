package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"
)

type DirEntry struct {
	Name  string `json:"name"`
	IsDir bool   `json:"isDir"`
	Size  int64  `json:"size"`
}

type tcpRequest struct {
	Op   string `json:"op"`
	Path string `json:"path,omitempty"`
	Name string `json:"name,omitempty"`
	Size int64  `json:"size,omitempty"`
}

type tcpResponse struct {
	Ok      bool       `json:"ok"`
	Code    string     `json:"code,omitempty"`
	Error   string     `json:"error,omitempty"`
	Entries []DirEntry `json:"entries,omitempty"`
	Size    int64      `json:"size,omitempty"`
}

const (
	connectTimeout  = 5 * time.Second
	requestTimeout  = 30 * time.Second
	transferTimeout = 300 * time.Second
	maxRetries      = 3

	CodeOK          = "OK"
	CodeBadRequest  = "E_BAD_REQUEST"
	CodeInternal    = "E_INTERNAL"
	CodeTimeout     = "E_TIMEOUT"
	CodeUnsupported = "E_UNSUPPORTED"
	CodeNotFound    = "E_NOT_FOUND"
)

type AppError struct {
	Code string
	Err  error
}

func (e AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %v", e.Code, e.Err)
	}
	return fmt.Sprintf("[%s]", e.Code)
}

func (e AppError) Unwrap() error {
	return e.Err
}

func wrapError(code string, err error) error {
	if err == nil {
		return nil
	}
	return AppError{Code: code, Err: err}
}

type progressReader struct {
	r       io.Reader
	Total   int64
	cb      func(written int64)
	written int64
}

func (p *progressReader) Read(buf []byte) (int, error) {
	n, err := p.r.Read(buf)
	if n > 0 {
		p.written += int64(n)
		if p.cb != nil {
			p.cb(p.written)
		}
	}
	return n, err
}

func isRetryable(err error) bool {
	if err == nil {
		return false
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return netErr.Timeout() || netErr.Temporary()
	}
	if errors.Is(err, io.EOF) {
		return true
	}
	var appErr AppError
	if errors.As(err, &appErr) {
		return appErr.Code == CodeTimeout || appErr.Code == CodeInternal
	}
	return false
}

func withRetries(operation func() error) error {
	var lastErr error
	for i := 0; i < maxRetries; i++ {
		lastErr = operation()
		if lastErr == nil {
			return nil
		}
		if !isRetryable(lastErr) {
			return lastErr
		}
		if i < maxRetries-1 {
			time.Sleep(time.Duration(i+1) * 200 * time.Millisecond)
		}
	}
	return lastErr
}

func normalizeAddr(server string) (string, error) {
	server = strings.TrimSpace(server)
	server = strings.TrimPrefix(server, "http://")
	server = strings.TrimPrefix(server, "https://")
	server = strings.TrimSuffix(server, "/")
	if server == "" {
		return "", fmt.Errorf("server address empty")
	}
	if !strings.Contains(server, ":") {
		server = server + ":9000"
	}
	return server, nil
}

func doRequest(server string, req tcpRequest) (*tcpResponse, *bufio.Reader, net.Conn, error) {
	addr, err := normalizeAddr(server)
	if err != nil {
		return nil, nil, nil, wrapError(CodeBadRequest, err)
	}
	dialer := net.Dialer{Timeout: connectTimeout}
	conn, err := dialer.Dial("tcp", addr)
	if err != nil {
		return nil, nil, nil, wrapError(CodeTimeout, err)
	}
	conn.SetDeadline(time.Now().Add(requestTimeout))

	if err := json.NewEncoder(conn).Encode(req); err != nil {
		conn.Close()
		return nil, nil, nil, wrapError(CodeInternal, err)
	}

	r := bufio.NewReader(conn)
	line, err := r.ReadBytes('\n')
	if err != nil {
		conn.Close()
		return nil, nil, nil, wrapError(CodeTimeout, err)
	}
	var resp tcpResponse
	if err := json.Unmarshal(line, &resp); err != nil {
		conn.Close()
		return nil, nil, nil, wrapError(CodeInternal, err)
	}
	if !resp.Ok {
		conn.Close()
		return nil, nil, nil, wrapError(resp.Code, fmt.Errorf(resp.Error))
	}
	return &resp, r, conn, nil
}

func fetchList(server string, remotePath string) ([]DirEntry, error) {
	var entries []DirEntry
	err := withRetries(func() error {
		resp, _, conn, err := doRequest(server, tcpRequest{Op: "list", Path: remotePath})
		if conn != nil {
			conn.Close()
		}
		if err != nil {
			return err
		}
		entries = resp.Entries
		return nil
	})
	return entries, err
}

func mkdir(server string, remotePath string) error {
	return withRetries(func() error {
		_, _, conn, err := doRequest(server, tcpRequest{Op: "mkdir", Path: remotePath})
		if conn != nil {
			conn.Close()
		}
		return err
	})
}

func uploadFile(server string, destPath string, fileName string, fileReader io.Reader, totalBytes int64, progress func(written int64)) error {
	attempt := func() error {
		return uploadFileOnce(server, destPath, fileName, fileReader, totalBytes, progress)
	}

	if _, ok := fileReader.(io.ReadSeeker); ok {
		return withRetries(func() error {
			if seeker, _ := fileReader.(io.ReadSeeker); seeker != nil {
				_, _ = seeker.Seek(0, io.SeekStart)
			}
			return attempt()
		})
	}

	return attempt()
}

func uploadFileOnce(server string, destPath string, fileName string, fileReader io.Reader, totalBytes int64, progress func(written int64)) error {
	addr, err := normalizeAddr(server)
	if err != nil {
		return wrapError(CodeBadRequest, err)
	}
	dialer := net.Dialer{Timeout: connectTimeout}
	conn, err := dialer.Dial("tcp", addr)
	if err != nil {
		return wrapError(CodeTimeout, err)
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(requestTimeout))

	req := tcpRequest{Op: "upload", Path: destPath, Name: fileName, Size: totalBytes}
	if err := json.NewEncoder(conn).Encode(req); err != nil {
		return wrapError(CodeInternal, err)
	}

	uploadReader := fileReader
	if progress != nil {
		uploadReader = &progressReader{r: fileReader, Total: totalBytes, cb: progress}
	}

	conn.SetWriteDeadline(time.Now().Add(transferTimeout))
	if _, err := io.CopyN(conn, uploadReader, totalBytes); err != nil {
		return wrapError(CodeTimeout, err)
	}

	r := bufio.NewReader(conn)
	line, err := r.ReadBytes('\n')
	if err != nil {
		return wrapError(CodeTimeout, err)
	}
	var resp tcpResponse
	if err := json.Unmarshal(line, &resp); err != nil {
		return wrapError(CodeInternal, err)
	}
	if !resp.Ok {
		return wrapError(resp.Code, fmt.Errorf(resp.Error))
	}
	return nil
}

func downloadFile(server string, remotePath string, fileName string, localTarget string, progress func(written int64, total int64)) error {
	return withRetries(func() error {
		return downloadFileOnce(server, remotePath, fileName, localTarget, progress)
	})
}

func downloadFileOnce(server string, remotePath string, fileName string, localTarget string, progress func(written int64, total int64)) error {
	addr, err := normalizeAddr(server)
	if err != nil {
		return wrapError(CodeBadRequest, err)
	}
	dialer := net.Dialer{Timeout: connectTimeout}
	conn, err := dialer.Dial("tcp", addr)
	if err != nil {
		return wrapError(CodeTimeout, err)
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(requestTimeout))

	req := tcpRequest{Op: "download", Path: remotePath, Name: fileName}
	if err := json.NewEncoder(conn).Encode(req); err != nil {
		return wrapError(CodeInternal, err)
	}

	r := bufio.NewReader(conn)
	line, err := r.ReadBytes('\n')
	if err != nil {
		return wrapError(CodeTimeout, err)
	}
	var resp tcpResponse
	if err := json.Unmarshal(line, &resp); err != nil {
		return wrapError(CodeInternal, err)
	}
	if !resp.Ok {
		return wrapError(resp.Code, fmt.Errorf(resp.Error))
	}

	f, err := os.Create(localTarget)
	if err != nil {
		return wrapError(CodeInternal, err)
	}
	defer f.Close()

	conn.SetReadDeadline(time.Now().Add(transferTimeout))
	var reader io.Reader = r
	if progress != nil {
		var written int64
		reader = io.TeeReader(r, &countWriter{cb: func(n int64) {
			written += n
			progress(written, resp.Size)
		}})
	}

	if _, err := io.CopyN(f, reader, resp.Size); err != nil {
		return wrapError(CodeTimeout, err)
	}
	return nil
}

type countWriter struct {
	cb func(n int64)
}

func (w *countWriter) Write(p []byte) (int, error) {
	n := len(p)
	w.cb(int64(n))
	return n, nil
}

func buildDownloadURL(server, remotePath, fileName string) (string, error) {
	return "", fmt.Errorf("buildDownloadURL is not supported in TCP mode")
}
