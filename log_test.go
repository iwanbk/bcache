package bcache

import "testing"

func TestLog(t *testing.T) {
	logger := &nopLogger{}
	logger.Errorf("Nothing %s", "nothing")
	logger.Printf("Nothing %s", "nothing")
	logger.Debugf("Nothing %s", "nothing")
}
