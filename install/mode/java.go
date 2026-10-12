package mode

import (
	"fmt"
	"regexp"
)

// Java 静默安装 Oracle JDK。同一大版本已经安装时只给出警告。
// JAVA_HOME 不在这里写入，这条安装成功后再配置。
// 返回值表示这次是否真正安装成功。
//
//	/s          静默安装
//	ADDLOCAL    只装开发工具和源码，不装独立的公共 JRE。JDK 自带的 JRE 仍在安装目录的 jre 下
//	INSTALLDIR  安装目录。路径有空格时要加引号
func Java(label, pkg, dest string) (bool, error) {
	major, err := javaMajor(pkg)
	if err != nil {
		return false, err
	}
	dir, found, err := FindJava(major)
	if skip, err := skipInstalled(label, dir, found, err); skip || err != nil {
		return false, err
	}
	args := `/s ADDLOCAL="ToolsFeature,SourceFeature" INSTALLDIR="` + dest + `"`
	if err := run(dest, pkg, args); err != nil {
		return false, err
	}
	succeed(dest)
	return true, nil
}

var javaMajorPattern = regexp.MustCompile(`(?i)jdk-(\d+)`)

func javaMajor(pkg string) (string, error) {
	match := javaMajorPattern.FindStringSubmatch(pkg)
	if match == nil {
		return "", fmt.Errorf("无法从安装包名得到 Java 版本: %s", pkg)
	}
	return match[1], nil
}
