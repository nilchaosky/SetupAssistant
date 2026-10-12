package env

import "testing"

func TestDatabaseBinComesFromInstallDir(t *testing.T) {
	var got []string
	orig := addToPath
	addToPath = func(dir string) error {
		got = append(got, dir)
		return nil
	}
	t.Cleanup(func() { addToPath = orig })

	if err := MySQL(`D:\Database\MySQL`); err != nil {
		t.Fatal(err)
	}
	if err := PostgreSQL(`E:\Database\PostgreSQL`); err != nil {
		t.Fatal(err)
	}
	if err := Apply("App", `D:\Tools\App`); err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != `D:\Database\MySQL\bin` || got[1] != `E:\Database\PostgreSQL\bin` {
		t.Fatalf("path = %v", got)
	}
}

func TestJavaSetsHomeAndBin(t *testing.T) {
	var home, bin string
	origHome := setSystemString
	origPath := addToPath
	setSystemString = func(name, value string) error {
		if name != "JAVA_HOME" {
			t.Fatalf("name = %s", name)
		}
		home = value
		return nil
	}
	addToPath = func(dir string) error {
		bin = dir
		return nil
	}
	t.Cleanup(func() {
		setSystemString = origHome
		addToPath = origPath
	})

	if err := Apply("Java", `E:\Language\Java`); err != nil {
		t.Fatal(err)
	}
	if home != `E:\Language\Java` || bin != `%JAVA_HOME%\bin` {
		t.Fatalf("home = %s bin = %s", home, bin)
	}
}
