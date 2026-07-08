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

func TestMode(t *testing.T) {
	tests := []struct {
		mode int
		want int
	}{
		{mode: 0, want: 1},
		{mode: 1, want: 1},
		{mode: 2, want: 2},
		{mode: 3, want: 3},
		{mode: 4, want: 1},
	}

	for _, tt := range tests {
		Settings.Mode = tt.mode
		if got := Mode(); got != tt.want {
			t.Fatalf("mode %d: got %d, want %d", tt.mode, got, tt.want)
		}
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

func TestPasskeySettings(t *testing.T) {
	bytes, _ := os.ReadFile("../testdata/setting.json")

	f, _ := os.CreateTemp("", "PasskeySettings")
	filename := f.Name()
	f.Write(bytes)
	f.Close()
	t.Cleanup(func() { os.Remove(filename) })

	Read(filename)
	handle, e := EnsureUserHandle("test")
	if e != nil {
		t.Fatal(e)
	}
	if handle == "" {
		t.Fatal("handle empty")
	}

	passkey := PasskeyType{CredentialID: "cred", PublicKey: "key", SignCount: 1}
	if e := AddPasskey("test", passkey); e != nil {
		t.Fatal(e)
	}
	user, ok := FindUserByCredentialID("cred")
	if !ok || user.Id != "test" {
		t.Fatal("credential user notfound")
	}
	user, ok = FindUserByHandle(handle)
	if !ok || user.Id != "test" {
		t.Fatal("handle user notfound")
	}

	passkey.SignCount = 2
	if e := UpdatePasskey("cred", passkey); e != nil {
		t.Fatal(e)
	}
	user, _ = FindUserByID("test")
	if len(user.Passkeys) != 1 || user.Passkeys[0].SignCount != 2 {
		t.Fatal("passkey update failed")
	}
}

func TestCreateRandomString(t *testing.T) {
	r := createRandomString(20)
	if len(r) != 20 {
		t.Error("error")
	}
}
