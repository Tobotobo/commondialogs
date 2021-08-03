// commondialogs.go
// Copyright (c) 2021 Tobotobo
// This software is released under the MIT License.
// http://opensource.org/licenses/mit-license.php

// Copyright 2010 The Walk Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
// https://github.com/lxn/walk/blob/master/LICENSE
// https://github.com/lxn/walk/blob/master/commondialogs.go

// +build windows

//lint:file-ignore SA1019 syscall.StringToUTF16 と syscall.StringToUTF16Ptr を使用します

package openfiledialog

import (
	"errors"
	"fmt"
	"path/filepath"
	"syscall"
	"unsafe"

	"github.com/lxn/win"
)

type FileDialog struct {
	Title          string
	FilePath       string
	FilePaths      []string
	InitialDirPath string
	Filter         string
	FilterIndex    int
	Flags          uint32
	ShowReadOnlyCB bool
}

func newError(message string) error {
	return errors.New(message)
}

func (dlg *FileDialog) show(owner win.HWND, fun func(ofn *win.OPENFILENAME) bool, flags uint32) (accepted bool, err error) {
	ofn := new(win.OPENFILENAME)

	ofn.LStructSize = uint32(unsafe.Sizeof(*ofn))
	ofn.HwndOwner = owner

	filter := make([]uint16, len(dlg.Filter)+2)
	copy(filter, syscall.StringToUTF16(dlg.Filter))
	// Replace '|' with the expected '\0'.
	for i, c := range filter {
		if byte(c) == '|' {
			filter[i] = uint16(0)
		}
	}
	ofn.LpstrFilter = &filter[0]
	ofn.NFilterIndex = uint32(dlg.FilterIndex)

	ofn.LpstrInitialDir = syscall.StringToUTF16Ptr(dlg.InitialDirPath)
	ofn.LpstrTitle = syscall.StringToUTF16Ptr(dlg.Title)
	ofn.Flags = win.OFN_FILEMUSTEXIST | flags | dlg.Flags

	if !dlg.ShowReadOnlyCB {
		ofn.Flags |= win.OFN_HIDEREADONLY
	}

	var fileBuf []uint16
	if flags&win.OFN_ALLOWMULTISELECT > 0 {
		fileBuf = make([]uint16, 65536)
	} else {
		fileBuf = make([]uint16, 1024)
		copy(fileBuf, syscall.StringToUTF16(dlg.FilePath))
	}
	ofn.LpstrFile = &fileBuf[0]
	ofn.NMaxFile = uint32(len(fileBuf))

	if !fun(ofn) {
		errno := win.CommDlgExtendedError()
		if errno != 0 {
			err = newError(fmt.Sprintf("Error %d", errno))
		}
		return
	}

	dlg.FilterIndex = int(ofn.NFilterIndex)

	if flags&win.OFN_ALLOWMULTISELECT > 0 {
		split := func() [][]uint16 {
			var parts [][]uint16

			from := 0
			for i, c := range fileBuf {
				if c == 0 {
					if i == from {
						return parts
					}

					parts = append(parts, fileBuf[from:i])
					from = i + 1
				}
			}

			return parts
		}

		parts := split()

		if len(parts) == 1 {
			dlg.FilePaths = []string{syscall.UTF16ToString(parts[0])}
		} else {
			dirPath := syscall.UTF16ToString(parts[0])
			dlg.FilePaths = make([]string, len(parts)-1)

			for i, fp := range parts[1:] {
				dlg.FilePaths[i] = filepath.Join(dirPath, syscall.UTF16ToString(fp))
			}
		}
	} else {
		dlg.FilePath = syscall.UTF16ToString(fileBuf)
	}

	accepted = true

	return
}

func (dlg *FileDialog) ShowOpen(owner win.HWND) (accepted bool, err error) {
	return dlg.show(owner, win.GetOpenFileName, win.OFN_NOCHANGEDIR)
}

func (dlg *FileDialog) ShowOpenMultiple(owner win.HWND) (accepted bool, err error) {
	return dlg.show(owner, win.GetOpenFileName, win.OFN_ALLOWMULTISELECT|win.OFN_EXPLORER|win.OFN_NOCHANGEDIR)
}

func (dlg *FileDialog) ShowSave(owner win.HWND) (accepted bool, err error) {
	return dlg.show(owner, win.GetSaveFileName, win.OFN_NOCHANGEDIR)
}
