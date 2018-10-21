package googledrive2hugo

import (
	"testing"
)

// quick test to make sure it works and there are no Hugo regressions
func TestURLize(t *testing.T) {
	if URLize("hello world") != "hello-world" {
		t.Errorf("failed hello world")
	}
}
