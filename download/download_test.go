package download

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"

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
	if err := Download(dir, items); err != nil {
		t.Fatal(err)
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
	if err := Download(dir, items[1:]); err != nil {
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
