package env

import "path/filepath"

// PostgreSQL 把客户端命令加进系统 PATH。psql 在安装目录的 bin 下。
func PostgreSQL(dest string) error {
	return addToPath(filepath.Join(dest, "bin"))
}
