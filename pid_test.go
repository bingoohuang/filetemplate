package filetemplate

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/bingoohuang/sysinfo"
	"github.com/stretchr/testify/assert"
)

// nolint gomnds
func TestPid(t *testing.T) {
	assert.Equal(t, "123", FindPid("123"))

	file, _ := ioutil.TempFile("", "a.pid")
	_, _ = file.Write([]byte("123"))
	file.Close()
	pidFile := file.Name()
	defer os.Remove(pidFile)

	assert.Equal(t, "123", FindPid(pidFile))

	top, _ := sysinfo.PsAuxTop(10)
	t0 := top[5]
	assert.Equal(t, strconv.Itoa(t0.Pid), FindPid("pgrep "+filepath.Base(t0.Command)))
}
