package windows

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"
)

// JavaHome 返回已安装 JDK 的目录。major 是大版本，例如 8、21。
func JavaHome(major string) (string, bool, error) {
	for _, root := range []registry.Key{registry.LOCAL_MACHINE, registry.CURRENT_USER} {
		for _, base := range []string{`SOFTWARE\JavaSoft`, `SOFTWARE\WOW6432Node\JavaSoft`} {
			dir, found, err := javaHomeAt(root, base, major)
			if err != nil {
				return "", false, err
			}
			if found && fileExists(filepath.Join(dir, "bin", "java.exe")) {
				return dir, true, nil
			}
		}
	}
	return "", false, nil
}

func javaHomeAt(root registry.Key, base, major string) (string, bool, error) {
	if major == "8" {
		dir, found, err := registryString(root, base+`\Java Development Kit\1.8`, "JavaHome")
		return cleanDir(dir), found, err
	}
	key, found, err := openKey(root, base+`\JDK`)
	if err != nil || !found {
		return "", false, err
	}
	defer key.Close()
	names, err := key.ReadSubKeyNames(-1)
	if err != nil {
		return "", false, err
	}
	for _, name := range names {
		if name != major && !strings.HasPrefix(name, major+".") {
			continue
		}
		dir, found, err := registryString(root, base+`\JDK\`+name, "JavaHome")
		if err != nil || found {
			return cleanDir(dir), found, err
		}
	}
	return "", false, nil
}

// PythonHome 返回已安装 Python 的目录。series 是主次版本，例如 2.7、3.13。
func PythonHome(series string) (string, bool, error) {
	path := `SOFTWARE\Python\PythonCore\` + series + `\InstallPath`
	wow := `SOFTWARE\WOW6432Node\Python\PythonCore\` + series + `\InstallPath`
	for _, root := range []registry.Key{registry.LOCAL_MACHINE, registry.CURRENT_USER} {
		for _, keyPath := range []string{path, wow} {
			dir, found, err := registryString(root, keyPath, "")
			if err != nil {
				return "", false, err
			}
			dir = cleanDir(dir)
			if found && fileExists(filepath.Join(dir, "python.exe")) {
				return dir, true, nil
			}
		}
	}
	return "", false, nil
}

// AppDir 按显示名称查找软件，并且程序文件必须还在磁盘上。
// 只剩注册表记录、程序已经删掉时，不算已安装。
// 名称相同，或以这个名称加空格、括号继续，都算同一个产品。
func AppDir(prefix string) (string, bool, error) {
	roots := []struct {
		root registry.Key
		path string
	}{
		{registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`},
		{registry.LOCAL_MACHINE, `SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall`},
		{registry.CURRENT_USER, `SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`},
	}
	for _, item := range roots {
		dir, found, err := appDirIn(item.root, item.path, prefix)
		if err != nil || found {
			return dir, found, err
		}
	}
	return "", false, nil
}

func appDirIn(root registry.Key, path, prefix string) (string, bool, error) {
	key, found, err := openKey(root, path)
	if err != nil || !found {
		return "", false, err
	}
	defer key.Close()
	names, err := key.ReadSubKeyNames(-1)
	if err != nil {
		return "", false, err
	}
	for _, name := range names {
		sub, found, err := openKey(root, path+`\`+name)
		if err != nil || !found {
			if err != nil {
				return "", false, err
			}
			continue
		}
		display, _, _ := sub.GetStringValue("DisplayName")
		location, _, _ := sub.GetStringValue("InstallLocation")
		icon, _, _ := sub.GetStringValue("DisplayIcon")
		sub.Close()
		if !sameProduct(display, prefix) || !programExists(location, icon) {
			continue
		}
		return installDir(location, icon), true, nil
	}
	return "", false, nil
}

func sameProduct(name, prefix string) bool {
	name = strings.TrimSpace(name)
	prefix = strings.TrimSpace(prefix)
	if name == "" || prefix == "" {
		return false
	}
	if strings.EqualFold(name, prefix) {
		return true
	}
	lowerName := strings.ToLower(name)
	lowerPrefix := strings.ToLower(prefix)
	if !strings.HasPrefix(lowerName, lowerPrefix) {
		return false
	}
	rest := lowerName[len(lowerPrefix):]
	if rest == "" {
		return true
	}
	if rest[0] != ' ' && rest[0] != '(' {
		return false
	}
	return !strings.Contains(rest, "insiders")
}

// MySQLServerDir 返回已安装的 MySQL Server 目录。series 是主次版本，例如 8.0。
func MySQLServerDir(series string) (string, bool, error) {
	dir, found, err := registryString(registry.LOCAL_MACHINE, `SOFTWARE\MySQL AB\MySQL Server `+series, "Location")
	if err != nil {
		return "", false, err
	}
	dir = cleanDir(dir)
	if found && fileExists(filepath.Join(dir, "bin", "mysqld.exe")) {
		return dir, true, nil
	}
	return serviceInstallDir("MySQL" + strings.ReplaceAll(series, ".", ""))
}

// PostgreSQLDir 返回已安装的 PostgreSQL 目录。major 是主版本，例如 18。
func PostgreSQLDir(major string) (string, bool, error) {
	for _, arch := range []string{"x64", "x86"} {
		dir, found, err := registryString(registry.LOCAL_MACHINE, `SOFTWARE\PostgreSQL\Installations\postgresql-`+arch+`-`+major, "Base Directory")
		if err != nil {
			return "", false, err
		}
		dir = cleanDir(dir)
		if found && fileExists(filepath.Join(dir, "bin", "postgres.exe")) {
			return dir, true, nil
		}
	}
	return serviceInstallDir("postgresql-x64-" + major)
}

func serviceInstallDir(name string) (string, bool, error) {
	image, found, err := registryString(registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Services\`+name, "ImagePath")
	if err != nil || !found {
		return "", false, err
	}
	exe := imageExecutable(image)
	if !fileExists(exe) {
		return "", false, nil
	}
	return installDirFromImage(image), true, nil
}

func imageExecutable(image string) string {
	image = strings.TrimSpace(image)
	if strings.HasPrefix(image, `"`) {
		if end := strings.Index(image[1:], `"`); end >= 0 {
			return image[1 : 1+end]
		}
	} else if i := strings.IndexAny(image, " \t"); i > 0 {
		return image[:i]
	}
	return image
}

// programExists 确认注册表指向的程序文件还在。图标路径优先，否则看安装目录是否存在。
func programExists(location, icon string) bool {
	if file := iconFile(icon); file != "" {
		return fileExists(file)
	}
	return dirExists(cleanDir(location))
}

func iconFile(icon string) string {
	icon = strings.Trim(strings.TrimSpace(icon), `"`)
	if i := strings.LastIndex(icon, ","); i > 0 && iconIndex(icon[i+1:]) {
		icon = strings.TrimSpace(icon[:i])
	}
	if icon == "" || filepath.Ext(icon) == "" {
		return ""
	}
	return icon
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func dirExists(path string) bool {
	if path == "" {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func installDirFromImage(image string) string {
	image = strings.TrimSpace(image)
	if strings.HasPrefix(image, `"`) {
		if end := strings.Index(image[1:], `"`); end >= 0 {
			image = image[1 : 1+end]
		}
	} else if i := strings.IndexAny(image, " \t"); i > 0 {
		image = image[:i]
	}
	if filepath.Ext(image) != "" {
		image = filepath.Dir(image)
	}
	if strings.EqualFold(filepath.Base(image), "bin") {
		image = filepath.Dir(image)
	}
	return cleanDir(image)
}

func registryString(root registry.Key, path, name string) (string, bool, error) {
	key, found, err := openKey(root, path)
	if err != nil || !found {
		return "", false, err
	}
	defer key.Close()
	value, _, err := key.GetStringValue(name)
	if err != nil {
		if errors.Is(err, registry.ErrNotExist) {
			return "", false, nil
		}
		return "", false, err
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return "", false, nil
	}
	return value, true, nil
}

func openKey(root registry.Key, path string) (registry.Key, bool, error) {
	key, err := registry.OpenKey(root, path, registry.QUERY_VALUE|registry.ENUMERATE_SUB_KEYS)
	if err != nil {
		if errors.Is(err, registry.ErrNotExist) {
			return 0, false, nil
		}
		return 0, false, err
	}
	return key, true, nil
}

func cleanDir(dir string) string {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return ""
	}
	return strings.TrimRight(filepath.Clean(dir), `\/`)
}
