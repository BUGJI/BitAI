package respond

import "testing"

func TestLocalizeValidationError(t *testing.T) {
	got := LocalizeMessage("Key: 'Code' Error:Field validation for 'Code' failed on the 'required' tag")
	want := "兑换码不能为空"
	if got != want {
		t.Fatalf("LocalizeMessage() = %q, want %q", got, want)
	}
}
