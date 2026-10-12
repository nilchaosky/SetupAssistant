package env

import "path/filepath"

// MySQL 把服务器命令加进系统 PATH。mysql 在安装目录的 bin 下。
func MySQL(dest string) error {
	return addToPath(filepath.Join(dest, "bin"))
}
