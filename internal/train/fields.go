package train

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ConfigFieldType represents the type of a configuration field
type ConfigFieldType int

const (
	FieldTypeHost ConfigFieldType = iota
	FieldTypeEnvVars
	FieldTypeLocalPath
	FieldTypeRemotePath
	FieldTypeRunDir
	FieldTypeTrainScript
	FieldTypeTrainArgs
	FieldTypeStartupCommand
	FieldTypeGlobalConfig
)

// ConfigField represents a configuration field in the wizard
type ConfigField struct {
	Type        ConfigFieldType
	Label       string
	Description string
	Platform    string // "NPU", "GPU", or "" for global
}

// GetConfigFields returns all configuration fields based on training mode
func GetConfigFields(mode TrainMode) []ConfigField {
	var fields []ConfigField

	switch mode {
	case ModeCompare:
		// NPU and GPU have separate configs
		fields = []ConfigField{
			{Type: FieldTypeHost, Label: "NPU 机器配置", Description: "NPU机器连接信息", Platform: "NPU"},
			{Type: FieldTypeEnvVars, Label: "NPU 环境变量", Description: "NPU环境变量设置", Platform: "NPU"},
			{Type: FieldTypeLocalPath, Label: "NPU 本地工程目录", Description: "本地代码路径", Platform: "NPU"},
			{Type: FieldTypeRemotePath, Label: "NPU 远程代码路径", Description: "远程代码存放路径", Platform: "NPU"},
			{Type: FieldTypeRunDir, Label: "NPU 运行目录", Description: "训练日志输出目录", Platform: "NPU"},
			{Type: FieldTypeTrainScript, Label: "NPU 启动脚本", Description: "训练脚本路径", Platform: "NPU"},
			{Type: FieldTypeTrainArgs, Label: "NPU 启动参数", Description: "训练脚本参数（含数据集）", Platform: "NPU"},
			{Type: FieldTypeStartupCommand, Label: "NPU 环境初始化命令", Description: "激活conda环境等初始化命令", Platform: "NPU"},
			{Type: FieldTypeHost, Label: "GPU 机器配置", Description: "GPU机器连接信息", Platform: "GPU"},
			{Type: FieldTypeEnvVars, Label: "GPU 环境变量", Description: "GPU环境变量设置", Platform: "GPU"},
			{Type: FieldTypeLocalPath, Label: "GPU 本地工程目录", Description: "本地代码路径", Platform: "GPU"},
			{Type: FieldTypeRemotePath, Label: "GPU 远程代码路径", Description: "远程代码存放路径", Platform: "GPU"},
			{Type: FieldTypeRunDir, Label: "GPU 运行目录", Description: "训练日志输出目录", Platform: "GPU"},
			{Type: FieldTypeTrainScript, Label: "GPU 启动脚本", Description: "训练脚本路径", Platform: "GPU"},
			{Type: FieldTypeTrainArgs, Label: "GPU 启动参数", Description: "训练脚本参数（含数据集）", Platform: "GPU"},
			{Type: FieldTypeStartupCommand, Label: "GPU 环境初始化命令", Description: "激活conda环境等初始化命令", Platform: "GPU"},
			{Type: FieldTypeGlobalConfig, Label: "全局同步配置", Description: "rsync压缩、并行度等全局设置", Platform: ""},
		}
	case ModeNPUOnly:
		fields = []ConfigField{
			{Type: FieldTypeHost, Label: "NPU 机器配置", Description: "NPU机器连接信息", Platform: "NPU"},
			{Type: FieldTypeEnvVars, Label: "NPU 环境变量", Description: "NPU环境变量设置", Platform: "NPU"},
			{Type: FieldTypeLocalPath, Label: "NPU 本地工程目录", Description: "本地代码路径", Platform: "NPU"},
			{Type: FieldTypeRemotePath, Label: "NPU 远程代码路径", Description: "远程代码存放路径", Platform: "NPU"},
			{Type: FieldTypeRunDir, Label: "NPU 运行目录", Description: "训练日志输出目录", Platform: "NPU"},
			{Type: FieldTypeTrainScript, Label: "NPU 启动脚本", Description: "训练脚本路径", Platform: "NPU"},
			{Type: FieldTypeTrainArgs, Label: "NPU 启动参数", Description: "训练脚本参数（含数据集）", Platform: "NPU"},
			{Type: FieldTypeStartupCommand, Label: "NPU 环境初始化命令", Description: "激活conda环境等初始化命令", Platform: "NPU"},
			{Type: FieldTypeGlobalConfig, Label: "全局同步配置", Description: "rsync压缩、并行度等全局设置", Platform: ""},
		}
	case ModeGPUOnly:
		fields = []ConfigField{
			{Type: FieldTypeHost, Label: "GPU 机器配置", Description: "GPU机器连接信息", Platform: "GPU"},
			{Type: FieldTypeEnvVars, Label: "GPU 环境变量", Description: "GPU环境变量设置", Platform: "GPU"},
			{Type: FieldTypeLocalPath, Label: "GPU 本地工程目录", Description: "本地代码路径", Platform: "GPU"},
			{Type: FieldTypeRemotePath, Label: "GPU 远程代码路径", Description: "远程代码存放路径", Platform: "GPU"},
			{Type: FieldTypeRunDir, Label: "GPU 运行目录", Description: "训练日志输出目录", Platform: "GPU"},
			{Type: FieldTypeTrainScript, Label: "GPU 启动脚本", Description: "训练脚本路径", Platform: "GPU"},
			{Type: FieldTypeTrainArgs, Label: "GPU 启动参数", Description: "训练脚本参数（含数据集）", Platform: "GPU"},
			{Type: FieldTypeStartupCommand, Label: "GPU 环境初始化命令", Description: "激活conda环境等初始化命令", Platform: "GPU"},
			{Type: FieldTypeGlobalConfig, Label: "全局同步配置", Description: "rsync压缩、并行度等全局设置", Platform: ""},
		}
	}

	return fields
}

// GetFieldCount returns the number of fields for a given mode
func GetFieldCount(mode TrainMode) int {
	return len(GetConfigFields(mode))
}

// FieldValue represents the value of a field
type FieldValue struct {
	StringValue string
	StringSlice []string
	IntValue    int
	BoolValue   bool
}

// GetFieldValue gets the value of a field from a project
func GetFieldValue(project *TrainProject, field ConfigField) *FieldValue {
	var cfg *PlatformConfig
	if field.Platform == "NPU" {
		cfg = project.NPUConfig
	} else if field.Platform == "GPU" {
		cfg = project.GPUConfig
	}

	switch field.Type {
	case FieldTypeHost:
		if cfg != nil {
			return &FieldValue{StringValue: fmt.Sprintf("%s@%s:%d", cfg.User, cfg.HostAddress, cfg.SSHPort)}
		}
	case FieldTypeEnvVars:
		if cfg != nil {
			return &FieldValue{StringSlice: cfg.EnvVars}
		}
	case FieldTypeLocalPath:
		if cfg != nil {
			return &FieldValue{StringValue: cfg.LocalPath}
		}
	case FieldTypeRemotePath:
		if cfg != nil {
			return &FieldValue{StringValue: cfg.RemoteCodePath}
		}
	case FieldTypeRunDir:
		if cfg != nil {
			return &FieldValue{StringValue: cfg.RunBaseDir}
		}
	case FieldTypeTrainScript:
		if cfg != nil {
			return &FieldValue{StringValue: cfg.TrainScript}
		}
	case FieldTypeTrainArgs:
		if cfg != nil {
			return &FieldValue{StringValue: cfg.TrainArgs}
		}
	case FieldTypeStartupCommand:
		if cfg != nil {
			return &FieldValue{StringValue: cfg.StartupCommand}
		}
	case FieldTypeGlobalConfig:
		return &FieldValue{
			BoolValue:   project.RsyncCompress,
			IntValue:    project.SyncParallelism,
			StringSlice: project.Exclude,
		}
	}

	return &FieldValue{}
}

// SetFieldValue sets the value of a field in a project
func SetFieldValue(project *TrainProject, field ConfigField, value *FieldValue) error {
	var cfg *PlatformConfig
	if field.Platform == "NPU" {
		if project.NPUConfig == nil {
			project.NPUConfig = &PlatformConfig{SSHPort: 22}
		}
		cfg = project.NPUConfig
	} else if field.Platform == "GPU" {
		if project.GPUConfig == nil {
			project.GPUConfig = &PlatformConfig{SSHPort: 22}
		}
		cfg = project.GPUConfig
	}

	switch field.Type {
	case FieldTypeHost:
		// Parse "user@host:port" format
		if cfg != nil && value.StringValue != "" {
			parts := strings.Split(value.StringValue, "@")
			if len(parts) == 2 {
				cfg.User = parts[0]
				hostPort := strings.Split(parts[1], ":")
				cfg.HostAddress = hostPort[0]
				if len(hostPort) > 1 {
					if port, err := strconv.Atoi(hostPort[1]); err == nil {
						cfg.SSHPort = port
					}
				}
			}
		}
	case FieldTypeEnvVars:
		if cfg != nil {
			cfg.EnvVars = value.StringSlice
		}
	case FieldTypeLocalPath:
		if cfg != nil {
			cfg.LocalPath = value.StringValue
		}
	case FieldTypeRemotePath:
		if cfg != nil {
			cfg.RemoteCodePath = value.StringValue
		}
	case FieldTypeRunDir:
		if cfg != nil {
			cfg.RunBaseDir = value.StringValue
		}
	case FieldTypeTrainScript:
		if cfg != nil {
			cfg.TrainScript = value.StringValue
		}
	case FieldTypeTrainArgs:
		if cfg != nil {
			cfg.TrainArgs = value.StringValue
		}
	case FieldTypeStartupCommand:
		if cfg != nil {
			cfg.StartupCommand = value.StringValue
		}
	case FieldTypeGlobalConfig:
		project.RsyncCompress = value.BoolValue
		project.SyncParallelism = value.IntValue
		if len(value.StringSlice) > 0 {
			project.Exclude = value.StringSlice
		}
	}

	project.UpdatedAt = getCurrentTime()
	return nil
}

// GetFieldDisplayValue returns a display string for a field value
func GetFieldDisplayValue(value *FieldValue, fieldType ConfigFieldType) string {
	if value == nil {
		return "(未设置)"
	}

	switch fieldType {
	case FieldTypeHost:
		if value.StringValue == "" {
			return "(未设置)"
		}
		return value.StringValue
	case FieldTypeEnvVars:
		if len(value.StringSlice) == 0 {
			return "(无)"
		}
		return strings.Join(value.StringSlice, ", ")
	case FieldTypeLocalPath, FieldTypeRemotePath, FieldTypeRunDir, FieldTypeTrainScript:
		if value.StringValue == "" {
			return "(未设置)"
		}
		return value.StringValue
	case FieldTypeTrainArgs:
		if value.StringValue == "" {
			return "(无)"
		}
		return value.StringValue
	case FieldTypeStartupCommand:
		if value.StringValue == "" {
			return "(无)"
		}
		return value.StringValue
	case FieldTypeGlobalConfig:
		return fmt.Sprintf("压缩:%v 并行:%d", value.BoolValue, value.IntValue)
	default:
		return fmt.Sprintf("%v", value)
	}
}

// FormatHost formats host config for display
func FormatHost(cfg *PlatformConfig) string {
	if cfg == nil || cfg.HostAddress == "" {
		return "(未设置)"
	}
	return fmt.Sprintf("%s@%s:%d", cfg.User, cfg.HostAddress, cfg.SSHPort)
}

// ParseHost parses a host string in format "user@host:port"
func ParseHost(s string) (user, host string, port int, err error) {
	port = 22 // default

	parts := strings.Split(s, "@")
	if len(parts) != 2 {
		return "", "", 0, fmt.Errorf("invalid format, expected user@host:port")
	}
	user = parts[0]

	hostPort := strings.Split(parts[1], ":")
	host = hostPort[0]
	if len(hostPort) > 1 {
		port, err = strconv.Atoi(hostPort[1])
		if err != nil {
			return "", "", 0, fmt.Errorf("invalid port number")
		}
	}

	return user, host, port, nil
}

// ParseEnvVars parses environment variables from string (one per line or comma-separated)
func ParseEnvVars(s string) []string {
	if strings.TrimSpace(s) == "" {
		return []string{}
	}

	// Try newline first
	lines := strings.Split(s, "\n")
	if len(lines) > 1 {
		var result []string
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" {
				result = append(result, line)
			}
		}
		return result
	}

	// Try comma-separated
	parts := strings.Split(s, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// getCurrentTime returns current time
func getCurrentTime() time.Time {
	return time.Now()
}
