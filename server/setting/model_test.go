package setting

import "testing"

func TestCheck(t *testing.T) {
	u := UserType{Id: "test", Password: "test"}
	if e := u.Check(); e != nil {
		t.Error(e)
	}

	u = UserType{Id: "", Password: "test"}
	if e := u.Check(); e == nil {
		t.Error(e)
	}

	u = UserType{Id: "test", Password: ""}
	if e := u.Check(); e == nil {
		t.Error(e)
	}
}

func TestCheckUser(t *testing.T) {
	Read("../testdata/setting.json")
	u := UserType{Id: "test", Password: "test"}
	if e := u.CheckUser(); e != nil {
		t.Error(e)
	}
	u = UserType{Id: "test", Password: "test2"}
	if e := u.CheckUser(); e == nil {
		t.Error(e)
	}
}
