package filetemplate

import (
	"io/ioutil"
	"os"
	"strconv"
	"time"

	"github.com/gobars/cmd"
	"github.com/sirupsen/logrus"
)

// FindPid finds pid as possible as it can
// 1. direct pid value(all digits)
// 2. pid file (file exists)
// 3. pid finder shell(like pgrep goland, ps -ef|grep goland|grep -v grep|awk '{print $2}', and etc.
func FindPid(v string) string {
	// 检查是否直接pid
	if _, err := strconv.Atoi(v); err == nil {
		return v
	}

	// 检查是否是pid文件
	{
		pidFile := fixFileName(v)
		stat, err := os.Stat(pidFile)

		if err == nil && !stat.IsDir() {
			pid, err := ioutil.ReadFile(pidFile)
			if err != nil {
				logrus.Warnf("failed to read pid file %s, error %v", pidFile, err)
			}

			return string(pid)
		}
	}

	// 可能是获取pid的命令行
	logrus.Warnf("start to exec %s", v)

	_, r := cmd.Bash(v, cmd.Timeout(10*time.Second)) // nolint gomnd

	if r.Error != nil {
		logrus.Warnf("failed to exec %s, error %v", v, r.Error)
	}

	if len(r.Stdout) > 0 {
		return r.Stdout[0]
	}

	return ""
}
