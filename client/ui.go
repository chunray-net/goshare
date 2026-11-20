package main

import (
	"fmt"
	"image/color"
	"log"
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"
)

type UI struct {
	app    fyne.App
	window fyne.Window
}

func NewUI(a fyne.App, w fyne.Window) *UI {
	return &UI{app: a, window: w}
}

func (ui *UI) Run() {
	a := ui.app
	w := ui.window

	preferences := a.Preferences()
	preferences.SetBool("first_run", false)

	serverEntry := widget.NewEntry()
	serverEntry.SetPlaceHolder("119.96.24.44:6000")
	serverEntry.SetText("119.96.24.44:6000")

	pathEntry := widget.NewEntry()
	pathEntry.SetPlaceHolder(".")
	pathEntry.SetText(".")

	dropHint := widget.NewLabel("拖拽文件到此窗口上传")
	dropHint.Alignment = fyne.TextAlignCenter
	dropHint.Wrapping = fyne.TextWrapWord

	statusLabel := widget.NewLabel("准备就绪")
	progressBar := widget.NewProgressBar()
	progressBar.Hide()

	listContainer := container.NewVBox()

	var refresh func()
	var processDropped func([]fyne.URI)

	refresh = func() {
		entries, err := fetchList(serverEntry.Text, pathEntry.Text)
		if err != nil {
			dialog.ShowError(err, w)
			statusLabel.SetText("刷新失败: " + err.Error())
			return
		}

		listContainer.Objects = nil
		for _, e := range entries {
			//kind := "文件"
			var textColor color.Color = color.Black
			if e.IsDir {
				//kind = "目录"
				textColor = color.RGBA{0, 0, 255, 255}
			}

			nameLabel := canvas.NewText(e.Name, textColor)
			//kindLabel := canvas.NewText(kind, textColor)
			sizeLabel := canvas.NewText(fmt.Sprintf("      %d KB", e.Size/1024), textColor)

			var action fyne.CanvasObject
			if e.IsDir {
				action = widget.NewButton("进入", func(name string) func() {
					return func() {
						pathEntry.SetText(remoteJoin(pathEntry.Text, name))
						refresh()
					}
				}(e.Name))
			} else {
				action = widget.NewButton("下载", func(name string) func() {
					return func() {
						saveDialog := dialog.NewFileSave(func(writer fyne.URIWriteCloser, err error) {
							if err != nil {
								dialog.ShowError(err, w)
								return
							}
							if writer == nil {
								return
							}
							localPath, pathErr := uriToLocalPath(writer.URI())
							writer.Close()
							if pathErr != nil {
								dialog.ShowError(pathErr, w)
								return
							}
							progressBar.Show()
							progressBar.SetValue(0)
							statusLabel.SetText("下载中: " + name)
							go func() {
								err := downloadFile(serverEntry.Text, pathEntry.Text, name, localPath, func(written, total int64) {
									if total > 0 {
										progressBar.SetValue(float64(written) / float64(total))
									}
								})
								if err != nil {
									dialog.ShowError(err, w)
									statusLabel.SetText("下载失败: " + err.Error())
								} else {
									statusLabel.SetText("下载完成: " + name)
								}
								progressBar.Hide()
							}()
						}, w)
						// 自动带入服务端文件名和后缀
						saveDialog.SetFileName(name)
						saveDialog.SetFilter(storage.NewExtensionFileFilter([]string{filepath.Ext(name)}))
						saveDialog.Show()
					}
				}(e.Name))
			}

			listContainer.Add(container.NewHBox(
				nameLabel,
				//kindLabel,
				sizeLabel,
				layout.NewSpacer(),
				action,
			))
		}
		listContainer.Refresh()
		statusLabel.SetText(fmt.Sprintf("已加载 %d 项", len(entries)))
	}

	processDropped = func(uris []fyne.URI) {
		if len(uris) == 0 {
			return
		}

		valid := make([]string, 0, len(uris))
		totalBytes := int64(0)
		for _, uri := range uris {
			if uri == nil {
				continue
			}
			localPath, err := uriToLocalPath(uri)
			if err != nil {
				log.Println("uri paths error", err)
				continue
			}
			info, err := os.Stat(localPath)
			if err != nil || info.IsDir() {
				continue
			}
			valid = append(valid, localPath)
			totalBytes += info.Size()
		}

		if len(valid) == 0 {
			statusLabel.SetText("无可上传文件")
			return
		}

		progressBar.Show()
		progressBar.SetValue(0)
		statusLabel.SetText("拖拽上传中...")

		go func() {
			uploadedBytes := int64(0)
			failed := make([]string, 0)

			for _, localPath := range valid {
				fileName := filepath.Base(localPath)
				f, err := os.Open(localPath)
				if err != nil {
					failed = append(failed, localPath)
					continue
				}

				info, _ := f.Stat()
				fileSize := info.Size()

				err = uploadFile(serverEntry.Text, pathEntry.Text, fileName, f, fileSize, func(written int64) {
					progressBar.SetValue(float64(uploadedBytes+written) / float64(totalBytes))
				})
				f.Close()

				if err != nil {
					failed = append(failed, localPath)
					continue
				}

				uploadedBytes += fileSize
				statusLabel.SetText(fmt.Sprintf("已完成 %d/%d 文件", len(valid)-len(failed), len(valid)))
			}

			progressBar.SetValue(1)

			if len(failed) > 0 {
				displayNames := make([]string, 0, len(failed))
				for _, p := range failed {
					displayNames = append(displayNames, filepath.Base(p))
				}
				msg := fmt.Sprintf("上传完成 %d/%d，失败: %s\n是否重试失败文件？", len(valid)-len(failed), len(valid), strings.Join(displayNames, ", "))
				dialog.ShowConfirm("上传结果", msg, func(ok bool) {
					if ok {
						continueURIs := make([]fyne.URI, 0, len(failed))
						for _, p := range failed {
							continueURIs = append(continueURIs, storage.NewFileURI(p))
						}
						processDropped(continueURIs)
					}
				}, w)
			} else {
				statusLabel.SetText(fmt.Sprintf("已完成 %d/%d 文件", len(valid), len(valid)))
			}

			progressBar.Hide()
			refresh()
		}()
	}

	w.SetOnDropped(func(_ fyne.Position, uris []fyne.URI) {
		processDropped(uris)
	})

	refreshBtn := widget.NewButton("刷新列表", refresh)
	upBtn := widget.NewButton("上级目录", func() {
		pathEntry.SetText(remoteParent(pathEntry.Text))
		refresh()
	})

	mkdirEntry := widget.NewEntry()
	mkdirEntry.SetPlaceHolder("新目录名称")
	mkdirBtn := widget.NewButton("创建目录", func() {
		if mkdirEntry.Text == "" {
			dialog.ShowError(fmt.Errorf("请输入目录名称"), w)
			return
		}
		fullPath := remoteJoin(pathEntry.Text, mkdirEntry.Text)
		if err := mkdir(serverEntry.Text, fullPath); err != nil {
			dialog.ShowError(err, w)
			statusLabel.SetText("mkdir 失败: " + err.Error())
			return
		}
		statusLabel.SetText("目录创建成功: " + fullPath)
		refresh()
	})

	uploadBtn := widget.NewButton("上传文件", func() {
		dialog.ShowFileOpen(func(rc fyne.URIReadCloser, err error) {
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
			if rc == nil {
				return
			}

			localPath, uriErr := uriToLocalPath(rc.URI())
			var totalSize int64
			if uriErr == nil {
				if fi, fiErr := os.Stat(localPath); fiErr == nil {
					totalSize = fi.Size()
				}
			}

			fileName := filepath.Base(rc.URI().Path())
			progressBar.Show()
			progressBar.SetValue(0)

			go func() {
				defer rc.Close()

				err := uploadFile(serverEntry.Text, pathEntry.Text, fileName, rc, totalSize, func(written int64) {
					if totalSize > 0 {
						progressBar.SetValue(float64(written) / float64(totalSize))
					}
				})
				progressBar.Hide()

				if err != nil {
					dialog.ShowError(err, w)
					statusLabel.SetText("上传失败: " + err.Error())
					return
				}
				statusLabel.SetText("上传成功: " + fileName)
				refresh()
			}()
		}, w)
	})

	controls := container.NewVBox(
		dropHint,
		progressBar,
		container.NewHBox(refreshBtn, upBtn, uploadBtn),
		container.NewGridWithColumns(2, mkdirEntry, mkdirBtn),
		statusLabel,
	)

	w.SetContent(container.NewBorder(controls, nil, nil, nil, container.NewVScroll(listContainer)))

	refresh()
	w.ShowAndRun()
}
