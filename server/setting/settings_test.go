package setting

import (
	"os"
	"path"
	"testing"
)

func TestRead(t *testing.T) {
	e := Read("../testdata/setting.json")
	if e != nil {
		t.Error(e)
	}

	filename := path.Join(os.TempDir(), "notfound"+createRandomString(20))
	e = Read(filename)
	if e != nil {
		t.Error(e)
	}
}

func TestUpdatePassword(t *testing.T) {
	bytes, _ := os.ReadFile("../testdata/setting.json")

	f, _ := os.CreateTemp("", "UpdatePassword")
	filename := f.Name()
	f.Write(bytes)
	f.Close()

	Read(filename)
	if e := UpdatePassword("test", "test2"); e != nil {
		t.Error(e)
	}
	if e := UpdatePassword("notfound", "test2"); e == nil {
		t.Error("error")
	}

	settingsFile = os.TempDir()
	if e := UpdatePassword("test", "test3"); e == nil {
		t.Error("error")
	}
}

func TestCreateRandomString(t *testing.T) {
	r := createRandomString(20)
	if len(r) != 20 {
		t.Error("error")
	}
}
