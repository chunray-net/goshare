package main

import (
	"fmt"
	"net/url"
	"path"
	"runtime"
	"strings"

	"fyne.io/fyne/v2"
)

func remoteJoin(base, elem string) string {
	if base == "" || base == "." {
		return elem
	}
	return path.Join(base, elem)
}

func remoteParent(base string) string {
	if base == "" || base == "." {
		return "."
	}
	p := path.Dir(base)
	if p == "." || p == "/" {
		return "."
	}
	return p
}

func uriToLocalPath(uri fyne.URI) (string, error) {
	if uri == nil {
		return "", fmt.Errorf("uri为空")
	}

	u, err := url.Parse(uri.String())
	if err == nil && u.Scheme == "file" {
		p := u.Path
		if runtime.GOOS == "windows" && strings.HasPrefix(p, "/") && len(p) > 2 && p[2] == ':' {
			p = p[1:]
		}
		p, err = url.PathUnescape(p)
		if err != nil {
			return "", err
		}
		return p, nil
	}

	p := uri.Path()
	if strings.HasPrefix(p, "file://") {
		u2, err := url.Parse(p)
		if err == nil {
			p2 := u2.Path
			if runtime.GOOS == "windows" && strings.HasPrefix(p2, "/") && len(p2) > 2 && p2[2] == ':' {
				p2 = p2[1:]
			}
			p2, err = url.PathUnescape(p2)
			if err != nil {
				return "", err
			}
			return p2, nil
		}
	}

	return p, nil
}
