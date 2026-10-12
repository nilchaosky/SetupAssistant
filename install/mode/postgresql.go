package mode

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
)

// PostgreSQL 用 EDB 安装包静默安装。bin 的 PATH 在这条安装成功后写入。
//
//	--mode unattended        不显示安装向导
//	--unattendedmodeui none  不显示进度窗口
//	--prefix                 安装目录。路径有空格时要加引号
//	--superpassword          postgres 密码
//	--serverport             监听端口
//	--disable-components     不安装列出的组件。stackbuilder 会再下载其他组件
func PostgreSQL(label, pkg, dest string) (bool, error) {
	major, err := postgresMajor(pkg)
	if err != nil {
		return false, err
	}
	dir, found, err := FindPostgreSQL(major)
	if skip, err := skipInstalled(label, dir, found, err); skip || err != nil {
		return false, err
	}
	password, port, err := postgresAccount()
	if err != nil {
		return false, err
	}
	if err := requireFreePort(port); err != nil {
		return false, err
	}
	args := `--mode unattended --unattendedmodeui none --prefix "` + dest +
		`" --superpassword "` + password + `" --serverport ` + port + ` --disable-components stackbuilder`
	if err := run(dest, pkg, args); err != nil {
		return false, err
	}
	succeed(dest)
	return true, nil
}

func postgresAccount() (string, string, error) {
	settings, err := readSettings("postgresql-silent.config")
	if err != nil {
		return "", "", err
	}
	password := settings["superpassword"]
	if password == "" {
		return "", "", errors.New("config/postgresql-silent.config 未设置 superpassword")
	}
	port := settings["serverport"]
	if port == "" {
		port = "5432"
	}
	return password, port, nil
}

var postgresMajorPattern = regexp.MustCompile(`(?i)postgresql-(\d+)`)

func postgresMajor(pkg string) (string, error) {
	match := postgresMajorPattern.FindStringSubmatch(filepath.Base(pkg))
	if match == nil {
		return "", fmt.Errorf("无法从安装包名得到 PostgreSQL 版本: %s", pkg)
	}
	return match[1], nil
}
