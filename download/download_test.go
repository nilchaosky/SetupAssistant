package download

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/nilchaosky/SetupAssistant/terminal"
)

func TestDownloadWritesFileAndSkipsEmptyURL(t *testing.T) {
	payload := []byte("installer-bytes")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/start" {
			http.Redirect(w, r, "/c1e7ff53-9fbd-4824-9ee7-5d48dc43febf", http.StatusFound)
			return
		}
		w.Header().Set("Content-Length", strconv.Itoa(len(payload)))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(payload)
	}))
	defer server.Close()

	dir := t.TempDir()
	items := []terminal.PendingInstall{
		{Name: "Java", Version: "8"},
		{Name: "Notepad++", Version: "v8.9.8", Package: "npp.8.9.8.Installer.x64.exe", URL: server.URL + "/start"},
	}
	succeeded, failed, err := Download(dir, items)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(succeeded, []string{"Notepad++ v8.9.8"}) || len(failed) != 0 {
		t.Fatalf("succeeded = %v failed = %v", succeeded, failed)
	}

	dest := filepath.Join(dir, "npp.8.9.8.Installer.x64.exe")
	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != string(payload) {
		t.Fatalf("file = %q", data)
	}
	if _, err := os.Stat(dest + ".part"); !os.IsNotExist(err) {
		t.Fatal("partial file should be removed")
	}
	if _, err := os.Stat(filepath.Join(dir, "Java")); !os.IsNotExist(err) {
		t.Fatal("empty url should not create a file")
	}

	info, err := os.Stat(dest)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := Download(dir, items[1:]); err != nil {
		t.Fatal(err)
	}
	again, err := os.Stat(dest)
	if err != nil {
		t.Fatal(err)
	}
	if !again.ModTime().Equal(info.ModTime()) || again.Size() != info.Size() {
		t.Fatal("existing file with the same size should be left unchanged")
	}
}

func TestDownloadRunsOnePackageAtATime(t *testing.T) {
	payload := []byte("installer-bytes")
	var mu sync.Mutex
	active := 0
	maxActive := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		active++
		if active > maxActive {
			maxActive = active
		}
		mu.Unlock()
		time.Sleep(50 * time.Millisecond)
		mu.Lock()
		active--
		mu.Unlock()
		w.Header().Set("Content-Length", strconv.Itoa(len(payload)))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(payload)
	}))
	defer server.Close()

	dir := t.TempDir()
	items := []terminal.PendingInstall{
		{Name: "Notepad++", Version: "v8.9.8", Package: "npp.exe", URL: server.URL + "/npp"},
		{Name: "MySQL", Version: "8.0.46", Package: "mysql.msi", URL: server.URL + "/mysql"},
	}
	succeeded, failed, err := Download(dir, items)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"Notepad++ v8.9.8", "MySQL 8.0.46"}
	if !reflect.DeepEqual(succeeded, want) || len(failed) != 0 {
		t.Fatalf("succeeded = %v failed = %v", succeeded, failed)
	}
	if maxActive != 1 {
		t.Fatalf("max active downloads = %d", maxActive)
	}
}

func TestDownloadListsFailureBesideSuccess(t *testing.T) {
	payload := []byte("ok")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/bad" {
			http.Error(w, "missing", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Length", strconv.Itoa(len(payload)))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(payload)
	}))
	defer server.Close()

	dir := t.TempDir()
	items := []terminal.PendingInstall{
		{Name: "Java", URL: ""},
		{Name: "VS Code", Package: "code.exe", URL: server.URL + "/bad"},
		{Name: "Notepad++", Version: "v8.9.8", Package: "npp.exe", URL: server.URL + "/ok"},
	}
	var succeeded, failed []string
	var err error
	output := captureStdout(t, func() {
		succeeded, failed, err = Download(dir, items)
	})
	if err == nil || !reflect.DeepEqual(succeeded, []string{"Notepad++ v8.9.8"}) {
		t.Fatalf("succeeded = %v err = %v", succeeded, err)
	}
	if len(failed) != 1 || failed[0] != "VS Code：HTTP 404" {
		t.Fatalf("failed = %v", failed)
	}
	if !strings.Contains(output, "下载成功：\nNotepad++ v8.9.8\n") || !strings.Contains(output, "下载失败：\nVS Code：HTTP 404\n") {
		t.Fatalf("output = %q", output)
	}
	if _, statErr := os.Stat(filepath.Join(dir, "npp.exe")); statErr != nil {
		t.Fatal(statErr)
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = writer
	t.Cleanup(func() { os.Stdout = orig })

	fn()
	writer.Close()
	os.Stdout = orig

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, reader); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}
