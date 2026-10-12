// Package env 在安装完成后写入环境变量。路径按安装目录拼接。
// 只处理静默安装写不了环境变量的软件。JetBrains、VS Code、Python 的 PATH 由安装包自己的静默参数写入。
package env

// Apply 按软件名写入对应的环境变量。dest 是该软件的安装目录。
func Apply(name, dest string) error {
	switch name {
	case "MySQL":
		return MySQL(dest)
	case "PostgreSQL":
		return PostgreSQL(dest)
	case "Java":
		return Java(dest)
	default:
		return nil
	}
}
