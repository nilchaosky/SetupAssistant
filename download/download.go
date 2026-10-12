// Package download 按清单把安装包下载到 ./temp。
// 一次只下载一个安装包。地址为空则跳过，不计入成功或失败。
// 目标文件已存在且大小与 Content-Length 一致时也跳过，并算作下载成功。
// 数据先写入 .part，完成后再改成 package 指定的文件名。
package download

import (
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-resty/resty/v2"
	"github.com/nilchaosky/SetupAssistant/progress"
	"github.com/nilchaosky/SetupAssistant/terminal"
)

// Run 把 items 下载到 ./temp，并返回成功和失败的软件列表。
func Run(items []terminal.PendingInstall) (succeeded, failed []string, err error) {
	return Download(Dir(), items)
}

// Dir 返回下载目录 ./temp。
func Dir() string {
	return filepath.Join(".", "temp")
}

// Remove 删除下载目录。安装成功后不再需要这些安装包。
func Remove() error {
	return os.RemoveAll(Dir())
}

// Download 按清单顺序逐个下载到 dir。
// 返回的两个列表保持原清单顺序。有失败时 err 一并返回这些错误。
func Download(dir string, items []terminal.PendingInstall) (succeeded, failed []string, err error) {
	var failedErrs []error
	for _, item := range items {
		if strings.TrimSpace(item.URL) == "" {
			continue
		}
		if oneErr := One(dir, item); oneErr != nil {
			failed = append(failed, fmt.Sprintf("%s：%v", label(item), oneErr))
			failedErrs = append(failedErrs, fmt.Errorf("%s: %w", label(item), oneErr))
			continue
		}
		succeeded = append(succeeded, label(item))
	}
	printLists(succeeded, failed)
	return succeeded, failed, errors.Join(failedErrs...)
}

// One 下载一个安装包。地址为空时直接返回。
func One(dir string, item terminal.PendingInstall) error {
	if strings.TrimSpace(item.URL) == "" {
		return nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	client := resty.New().SetHeader("User-Agent", "SetupAssistant")
	return downloadOne(client, dir, item)
}

// RemoveFile 删除 temp 里的这个安装包。resource 目录中的原文件不动。
func RemoveFile(pkg string) error {
	name := filepath.Base(filepath.FromSlash(strings.TrimSpace(pkg)))
	if name == "." || name == "" || name == string(filepath.Separator) || name == "/" {
		return nil
	}
	err := os.Remove(filepath.Join(Dir(), name))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func downloadOne(client *resty.Client, dir string, item terminal.PendingInstall) error {
	resp, err := client.R().SetDoNotParseResponse(true).Get(item.URL)
	if err != nil {
		return err
	}
	body := resp.RawBody()
	if body != nil {
		defer body.Close()
	}
	if resp.StatusCode() < 200 || resp.StatusCode() >= 300 {
		return fmt.Errorf("HTTP %d", resp.StatusCode())
	}
	if body == nil || resp.RawResponse == nil {
		return errors.New("响应为空")
	}

	name := fileName(resp.RawResponse.Request.URL.String(), item)
	dest := filepath.Join(dir, name)
	total := resp.RawResponse.ContentLength
	if info, statErr := os.Stat(dest); statErr == nil && total > 0 && info.Size() == total {
		return nil
	}

	fmt.Printf("下载 %s\n", label(item))
	part := dest + ".part"
	bar := progress.New(total)
	if err := writeFile(bar.NewProxyReader(body), part, func() { bar.Finish() }); err != nil {
		os.Remove(part)
		return err
	}
	os.Remove(dest)
	if err := os.Rename(part, dest); err != nil {
		os.Remove(part)
		return err
	}
	return nil
}

func writeFile(body io.Reader, part string, finish func()) error {
	file, err := os.Create(part)
	if err != nil {
		return err
	}
	_, err = io.Copy(file, body)
	if finish != nil {
		finish()
	}
	closeErr := file.Close()
	if err != nil {
		return err
	}
	return closeErr
}

func printLists(succeeded, failed []string) {
	fmt.Println("下载成功：")
	printNames(succeeded)
	fmt.Println("下载失败：")
	printNames(failed)
}

func printNames(names []string) {
	if len(names) == 0 {
		fmt.Println("无")
		return
	}
	for _, name := range names {
		fmt.Println(name)
	}
}

func fileName(rawURL string, item terminal.PendingInstall) string {
	if name := filepath.Base(strings.TrimSpace(item.Package)); name != "." && name != "/" && name != string(filepath.Separator) && name != "" {
		return name
	}
	parsed, err := url.Parse(rawURL)
	if err == nil {
		name := filepath.Base(parsed.Path)
		if name != "." && name != "/" && name != string(filepath.Separator) {
			return name
		}
	}
	name := item.Name
	if item.Version != "" {
		name += "-" + item.Version
	}
	return name
}

func label(item terminal.PendingInstall) string {
	if item.Version == "" {
		return item.Name
	}
	return item.Name + " " + item.Version
}
