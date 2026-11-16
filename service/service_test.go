package service

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	lf "github.com/sirupsen/logrus"
)

const probeRegex = "nohup\\.out(\\.[0-9]+)?"

func testLog() *lf.Entry {
	return lf.WithField("service", "test")
}

func writeFile(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := ioutil.WriteFile(path, []byte("line\n"), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestFindLogFilesMissingDir(t *testing.T) {
	root := filepath.Join(t.TempDir(), "missing")
	files, err := findLogFiles(root, probeRegex, testLog())
	if err == nil {
		t.Fatal("expected an error for a missing directory")
	}
	if !strings.Contains(err.Error(), root) {
		t.Fatalf("error should name the directory, got: %v", err)
	}
	if len(files) != 0 {
		t.Fatalf("expected no files, got %v", files)
	}
}

func TestFindLogFilesRootIsFile(t *testing.T) {
	root := filepath.Join(t.TempDir(), "nohup.out")
	writeFile(t, root)
	if _, err := findLogFiles(root, probeRegex, testLog()); err == nil {
		t.Fatal("expected an error when logrootdir is a file")
	}
}

func TestFindLogFilesInvalidRegex(t *testing.T) {
	if _, err := findLogFiles(t.TempDir(), "nohup(", testLog()); err == nil {
		t.Fatal("expected an error for an invalid regex")
	}
}

func TestFindLogFiles(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "nohup.out"))
	writeFile(t, filepath.Join(root, "nohup.out.1"))
	writeFile(t, filepath.Join(root, "other.log"))
	// file in a subdirectory is returned with its own path, not as root/name
	writeFile(t, filepath.Join(root, "old", "nohup.out.2"))
	// directory with a matching name is not a log file
	if err := os.MkdirAll(filepath.Join(root, "nohup.out.3"), 0755); err != nil {
		t.Fatal(err)
	}
	// link to a file that does not exist, as left behind by a rotation
	if err := os.Symlink(filepath.Join(root, "gone"), filepath.Join(root, "nohup.out.4")); err != nil {
		t.Fatal(err)
	}

	files, err := findLogFiles(root, probeRegex, testLog())
	if err != nil {
		t.Fatal(err)
	}

	got := []string{}
	for _, f := range files {
		if f.inode == 0 {
			t.Fatalf("no inode for %s", f.path)
		}
		if _, err := os.Stat(f.path); err != nil {
			t.Fatalf("returned path can't be read: %v", err)
		}
		rel, _ := filepath.Rel(root, f.path)
		got = append(got, rel)
	}
	sort.Strings(got)
	want := []string{"nohup.out", "nohup.out.1", filepath.Join("old", "nohup.out.2")}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("got %v, want %v", got, want)
	}
}
