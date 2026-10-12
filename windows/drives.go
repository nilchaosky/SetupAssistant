package windows

import (
	"fmt"

	"golang.org/x/sys/windows"
)

// Drive 是一块固定磁盘。
type Drive struct {
	Path string // 盘符根路径，例如 D:\
	Name string // 卷标。没有则为空
}

func (d Drive) Label() string {
	if d.Name == "" {
		return d.Path
	}
	return d.Path + "  " + d.Name
}

// ListDrives 返回本机固定磁盘，不包含光驱和可移动磁盘。
func ListDrives() ([]Drive, error) {
	mask, err := windows.GetLogicalDrives()
	if err != nil {
		return nil, err
	}
	var drives []Drive
	for letter := 'A'; letter <= 'Z'; letter++ {
		if mask&(1<<uint(letter-'A')) == 0 {
			continue
		}
		root := fmt.Sprintf("%c:\\", letter)
		path, err := windows.UTF16PtrFromString(root)
		if err != nil {
			continue
		}
		if windows.GetDriveType(path) != windows.DRIVE_FIXED {
			continue
		}
		drives = append(drives, Drive{Path: root, Name: volumeLabel(path)})
	}
	return drives, nil
}

func volumeLabel(root *uint16) string {
	name := make([]uint16, 256)
	err := windows.GetVolumeInformation(root, &name[0], uint32(len(name)), nil, nil, nil, nil, 0)
	if err != nil {
		return ""
	}
	return windows.UTF16ToString(name)
}
