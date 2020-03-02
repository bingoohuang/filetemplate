package filetemplate

// File is the structure for config file
type File struct {
	// 配置总文件名（带路径，全路径或者相对路径，支持~开头的相对路径）
	Filename string `json:"filename"`
	// 配置总文件名内容
	// 当为空时，表示不进行配置总文件名的替换
	Content string `json:"content"`

	// 子配置文件所在目录
	// 场景：当总配置文件中，以形如include {sub_conf_dir}/*.conf包含子配置文件
	SubConfDir string `json:"sub_conf_dir"`

	// 子配置使用方式
	// 当为空时，采用全量写入的方式（软删除子配置所在目录所有文件）
	// 当为overwrite时，仅仅写入指定的子配置文件
	SubConfMode string `json:"sub_conf_mode"`

	// 子配置内容 文件名（不包含路径）->文件内容
	SubConf map[string]string `json:"sub_conf"`

	// 直接配置重新加载的命令，例如Nginx的 nginx -s reload
	// 或者 kill -s HUP ${pid}，中间的参数可以采用${pid}替换的形式
	ReloadCmd string `json:"reload_cmd"`
	// pid号（整数），或者pid文件（文件路径）,或者找到pid的命令，比如pgrep goland
	// 或者 ps -ef|grep goland|grep -v grep|awk '{print $2}'
	PID string `json:"pid"`
}
