package env

import (
	"testing"

	"golang.org/x/sys/windows/registry"
)

func TestAppendPathEntry(t *testing.T) {
	updated, changed := appendPathEntry(`%SystemRoot%\system32`, `D:\Database\MySQL\bin`)
	if !changed || updated != `%SystemRoot%\system32;D:\Database\MySQL\bin` {
		t.Fatalf("updated = %q changed = %v", updated, changed)
	}

	again, changed := appendPathEntry(updated, `d:\database\mysql\bin\`)
	if changed || again != updated {
		t.Fatalf("duplicate updated = %q changed = %v", again, changed)
	}

	only, changed := appendPathEntry("  ", `D:\Database\PostgreSQL\bin`)
	if !changed || only != `D:\Database\PostgreSQL\bin` {
		t.Fatalf("empty updated = %q changed = %v", only, changed)
	}

	withHome, changed := appendPathEntry(only, `%JAVA_HOME%\bin`)
	if !changed || withHome != `D:\Database\PostgreSQL\bin;%JAVA_HOME%\bin` {
		t.Fatalf("java home updated = %q changed = %v", withHome, changed)
	}
}

func TestAddPathValueKeepsUnexpandedPath(t *testing.T) {
	const keyPath = `Software\SetupAssistantPathTest`
	key, _, err := registry.CreateKey(registry.CURRENT_USER, keyPath, registry.SET_VALUE|registry.QUERY_VALUE)
	if err != nil {
		t.Fatal(err)
	}
	if err := key.SetExpandStringValue("Path", `%SystemRoot%\system32`); err != nil {
		key.Close()
		t.Fatal(err)
	}
	key.Close()
	t.Cleanup(func() {
		_ = registry.DeleteKey(registry.CURRENT_USER, keyPath)
	})

	changed, err := addPathValue(registry.CURRENT_USER, keyPath, `D:\Database\MySQL\bin`)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected path to change")
	}
	changed, err = addPathValue(registry.CURRENT_USER, keyPath, `D:\Database\MySQL\bin`)
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Fatal("expected existing path to stay unchanged")
	}

	key, err = registry.OpenKey(registry.CURRENT_USER, keyPath, registry.QUERY_VALUE)
	if err != nil {
		t.Fatal(err)
	}
	defer key.Close()
	got, typ, err := key.GetStringValue("Path")
	if err != nil {
		t.Fatal(err)
	}
	if typ != registry.EXPAND_SZ {
		t.Fatalf("type = %d", typ)
	}
	want := `%SystemRoot%\system32;D:\Database\MySQL\bin`
	if got != want {
		t.Fatalf("path = %s", got)
	}
}

func TestSetStringValueReplacesOnlyWhenChanged(t *testing.T) {
	const keyPath = `Software\SetupAssistantEnvTest`
	key, _, err := registry.CreateKey(registry.CURRENT_USER, keyPath, registry.SET_VALUE|registry.QUERY_VALUE)
	if err != nil {
		t.Fatal(err)
	}
	key.Close()
	t.Cleanup(func() {
		_ = registry.DeleteKey(registry.CURRENT_USER, keyPath)
	})

	changed, err := setStringValue(registry.CURRENT_USER, keyPath, "JAVA_HOME", `E:\Language\Java`)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected JAVA_HOME to be created")
	}
	changed, err = setStringValue(registry.CURRENT_USER, keyPath, "JAVA_HOME", `E:\Language\Java`)
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Fatal("expected the same JAVA_HOME to stay unchanged")
	}

	key, err = registry.OpenKey(registry.CURRENT_USER, keyPath, registry.QUERY_VALUE)
	if err != nil {
		t.Fatal(err)
	}
	defer key.Close()
	got, typ, err := key.GetStringValue("JAVA_HOME")
	if err != nil {
		t.Fatal(err)
	}
	if typ != registry.SZ || got != `E:\Language\Java` {
		t.Fatalf("JAVA_HOME = %s type = %d", got, typ)
	}
}
