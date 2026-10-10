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

func Run(items []terminal.PendingInstall) error {
	return Download(Dir(), items)
}

func Dir() string {
	return filepath.Join(".", "temp")
}

func Download(dir string, items []terminal.PendingInstall) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	client := resty.New().SetHeader("User-Agent", "SetupAssistant")
	var failed []error
	for _, item := range items {
		if strings.TrimSpace(item.URL) == "" {
			fmt.Printf("跳过 %s：缺少下载地址\n", label(item))
			continue
		}
		if err := downloadOne(client, dir, item); err != nil {
			fmt.Printf("下载失败 %s：%v\n", label(item), err)
			failed = append(failed, fmt.Errorf("%s: %w", label(item), err))
		}
	}
	return errors.Join(failed...)
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
		fmt.Printf("已存在，跳过：%s\n", dest)
		return nil
	}

	fmt.Printf("下载 %s\n", label(item))
	part := dest + ".part"
	if err := writeFile(body, part, total); err != nil {
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

func writeFile(body io.Reader, part string, total int64) error {
	file, err := os.Create(part)
	if err != nil {
		return err
	}

	bar := progress.New(total)
	_, err = io.Copy(file, bar.NewProxyReader(body))
	bar.Finish()
	closeErr := file.Close()
	if err != nil {
		return err
	}
	return closeErr
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
