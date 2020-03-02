package filetemplate

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gobars/cmd"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/sirupsen/logrus"
)

// File is the structure for config file
type File struct {
	// ID
	// 当ID不为空时，表示使用指定ID的内容为模板，填充ZERO部分字段
	// 当ID为空时，表示直接操作
	ID string `json:"id,omitempty"`
	// 当前结构体的描述/备注信息
	Desc string `json:"desc"`

	// 配置总文件名（带路径，全路径或者相对路径，支持~开头的相对路径）
	Filename string `json:"filename"`
	// 配置总文件名内容
	// 当为空时，表示不进行配置总文件名的替换
	Content string `json:"content,omitempty"`

	// 子配置文件所在目录
	// 场景：当总配置文件中，以形如include {sub_conf_dir}/*.conf包含子配置文件
	SubDir string `json:"sub_dir,omitempty"`

	// 子配置使用方式
	// 当为空时，采用全量写入的方式（软删除子配置所在目录所有文件）
	// 当为overwrite时，仅仅写入指定的子配置文件
	SubMode string `json:"sub_mode,omitempty"`

	// 子配置内容 文件名（不包含路径）->文件内容
	Subs map[string]string `json:"subs,omitempty"`

	// 重新加载的命令，例如Nginx的 nginx -s reload
	// 或者 kill -s HUP ${pid}，中间的参数可以采用${pid}替换的形式
	Reload string `json:"reload,omitempty"`
	// pid号（整数），或者pid文件（文件路径）,或者找到pid的命令，比如pgrep goland
	// 或者 ps -ef|grep goland|grep -v grep|awk '{print $2}'
	PID string `json:"pid,omitempty"`
}

// Execute executes the file request.
func (f File) Execute() (interface{}, error) {
	vv := make([]interface{}, 0)

	if v, err := writeContent(f.Filename, f.Content); err != nil {
		return vv, err
	} else if v != nil {
		vv = append(vv, v)
	}

	if v, err := f.writeSubs(); err != nil {
		return vv, err
	} else if v != nil {
		vv = append(vv, v)
	}

	if v, err := f.reload(); err != nil {
		return vv, err
	} else if v != nil {
		vv = append(vv, v)
	}

	return vv, nil
}

func writeContent(file, content string) (interface{}, error) {
	filename := fixFileName(file)
	fs, err := os.Stat(filename)

	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	if err == nil { // 文件已经存在
		old, err := ioutil.ReadFile(filename)
		if err != nil {
			return nil, err
		}

		if content == string(old) {
			return fmt.Sprintf("file %s's content is same with the old", filename), nil
		}

		if err := renameFile(filename); err != nil {
			return nil, err
		}
	}

	var mode os.FileMode = 0777
	if fs != nil {
		mode = fs.Mode()
	}

	return ioutil.WriteFile(filename, []byte(content), mode), nil
}

func renameFile(filename string) error {
	t := time.Now().Format("20060102150405000")
	renamed := filename + "." + t

	if err := os.Rename(filename, renamed); err != nil {
		return fmt.Errorf("failed to rename to %s failed, error %v", renamed, err)
	}

	return nil
}

func fixFileName(filename string) string {
	expand, err := homedir.Expand(filename)
	if err != nil {
		logrus.Warnf("failed to expand homedir for %s, error %v", filename, err)
		return filename
	}

	return expand
}

func (f File) writeSubs() (interface{}, error) {
	if f.SubDir == "" || len(f.Subs) == 0 {
		return nil, nil
	}

	subDir := fixFileName(f.SubDir)
	fs, err := os.Stat(subDir)

	if err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(subDir, 0777); err != nil {
				return nil, fmt.Errorf("failed to create dir %s, error %w", subDir, err)
			}
		} else {
			return nil, fmt.Errorf("failed to os.Stat %s, error %w", subDir, err)
		}
	}

	if !fs.IsDir() {
		return nil, fmt.Errorf("subDir should be a direcory %s", subDir)
	}

	switch f.SubMode {
	case "":
		return f.overwriteSubsDirectly(subDir)
	case "overwrite":
		return nil, f.overwriteSubs(subDir)
	default:
		return nil, fmt.Errorf("unknown subMode %s, required (empty) or overwrite", f.SubMode)
	}
}

func (f File) overwriteSubsDirectly(dir string) (interface{}, error) {
	vv := make([]interface{}, 0, len(f.Subs))

	for subFile, subContent := range f.Subs {
		subFilename := filepath.Join(dir, subFile)

		if v, err := writeContent(subFilename, subContent); err != nil {
			return v, err
		} else if v != nil {
			vv = append(vv, v)
		}
	}

	return vv, nil
}

func (f File) overwriteSubs(dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}

		return renameFile(path)
	})
}

func (f File) reload() (interface{}, error) {
	if f.Reload == "" {
		return nil, nil
	}

	pid := FindPid(f.PID)
	varMap := map[string]string{
		"pid": pid,
	}

	reloadCmd := substitute(f.Reload, varMap)
	if reloadCmd == "" {
		return nil, fmt.Errorf("reload %s evaluated to empty", f.Reload)
	}

	logrus.Infof("reload %s evaluated to %s", f.Reload, reloadCmd)

	_, r := cmd.Bash(reloadCmd, cmd.Timeout(10*time.Second)) // nolint gomnd
	if r.Exit == 0 {
		logrus.Infof("reload %s successfully", reloadCmd)
	} else if r.Error != nil {
		logrus.Infof("reload %s failed, error %v", reloadCmd, r.Error)
		return nil, r.Error
	}

	if len(r.Stdout) > 0 {
		logrus.Infof("reload %s returned stdout %v", reloadCmd, r.Stdout)
	}

	if len(r.Stderr) > 0 {
		logrus.Infof("reload %s returned stderr %v", reloadCmd, r.Stderr)
	}

	return "reloaded successfully", nil
}

func substitute(s string, vars map[string]string) string {
	replaced := ""

	for {
		startPos := strings.Index(s, `${`)
		if startPos < 0 {
			replaced += s
			break
		}

		replaced += s[:startPos]
		s = s[startPos+2:]
		endPos := strings.Index(s, "}")

		if endPos < 0 {
			break
		}

		substituteVar := strings.ToLower(strings.TrimSpace(s[:endPos]))
		substituteVal := vars[substituteVar]
		replaced += substituteVal
		s = s[endPos+1:]
	}

	return replaced
}
