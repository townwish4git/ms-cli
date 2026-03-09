// Package train provides training project management for /train command.
package train

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/vigo999/ms-cli/configs"
	"gopkg.in/yaml.v3"
)

// TrainMode represents the training mode
type TrainMode int

const (
	ModeCompare TrainMode = iota // NPU,GPU对比训练
	ModeNPUOnly                  // NPU单独训练
	ModeGPUOnly                  // GPU单独训练
)

func (m TrainMode) String() string {
	switch m {
	case ModeCompare:
		return "NPU+GPU对比训练"
	case ModeNPUOnly:
		return "NPU单独训练"
	case ModeGPUOnly:
		return "GPU单独训练"
	default:
		return "Unknown"
	}
}

// PlatformConfig holds configuration for a single platform (NPU or GPU)
type PlatformConfig struct {
	HostAddress    string   `yaml:"host_address"`     // 机器IP
	User           string   `yaml:"user"`             // 用户名
	SSHPort        int      `yaml:"ssh_port"`         // SSH端口
	EnvVars        []string `yaml:"env_vars"`         // 环境变量
	LocalPath      string   `yaml:"local_path"`       // 本地工程目录
	RemoteCodePath string   `yaml:"remote_code_path"` // 远程代码路径
	RunBaseDir     string   `yaml:"run_base_dir"`     // 运行目录
	TrainScript    string   `yaml:"train_script"`     // 启动脚本位置
	TrainArgs      string   `yaml:"train_args"`       // 启动参数(含数据集)
	StartupCommand string   `yaml:"startup_command"`  // 环境初始化命令
}

// TrainProject is the training project configuration
type TrainProject struct {
	Version   int       `yaml:"version"`
	ID        string    `yaml:"id"`
	Name      string    `yaml:"name"`
	Mode      TrainMode `yaml:"mode"`
	CreatedAt time.Time `yaml:"created_at"`
	UpdatedAt time.Time `yaml:"updated_at"`

	// Global config (shared between platforms)
	RsyncCompress         bool     `yaml:"rsync_compress"`
	RsyncRespectGitIgnore bool     `yaml:"rsync_respect_gitignore"`
	SyncParallelism       int      `yaml:"sync_parallelism"`
	Exclude               []string `yaml:"exclude"`

	// Platform specific config
	NPUConfig *PlatformConfig `yaml:"npu_config,omitempty"`
	GPUConfig *PlatformConfig `yaml:"gpu_config,omitempty"`
}

// ProjectManager manages training projects
type ProjectManager struct {
	BaseDir string
}

// NewProjectManager creates a new ProjectManager
func NewProjectManager(workDir string) *ProjectManager {
	if workDir == "" {
		workDir, _ = os.Getwd()
	}
	return &ProjectManager{
		BaseDir: filepath.Join(workDir, ".train"),
	}
}

// GetProjectsDir returns the projects directory path
func (pm *ProjectManager) GetProjectsDir() string {
	return filepath.Join(pm.BaseDir, "projects")
}

// ListProjects returns all projects sorted by creation time (newest first)
func (pm *ProjectManager) ListProjects() ([]TrainProject, error) {
	projects := []TrainProject{}

	projectsDir := pm.GetProjectsDir()
	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return projects, nil
		}
		return nil, fmt.Errorf("read projects directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		projectID := entry.Name()
		if !strings.HasPrefix(projectID, "project-") {
			continue
		}

		project, err := pm.LoadProject(projectID)
		if err != nil {
			continue // Skip invalid projects
		}
		projects = append(projects, *project)
	}

	// Sort by creation time (newest first)
	sort.Slice(projects, func(i, j int) bool {
		return projects[i].CreatedAt.After(projects[j].CreatedAt)
	})

	return projects, nil
}

// LoadProject loads a project by ID
func (pm *ProjectManager) LoadProject(projectID string) (*TrainProject, error) {
	projectDir := filepath.Join(pm.GetProjectsDir(), projectID)
	configPath := filepath.Join(projectDir, "config.yaml")

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("read project config: %w", err)
	}

	var project TrainProject
	if err := yaml.Unmarshal(data, &project); err != nil {
		return nil, fmt.Errorf("parse project config: %w", err)
	}

	// Set defaults for backwards compatibility
	if project.Version == 0 {
		project.Version = 2
	}

	return &project, nil
}

// SaveProject saves a project to disk
func (pm *ProjectManager) SaveProject(project *TrainProject) error {
	projectDir := filepath.Join(pm.GetProjectsDir(), project.ID)

	// Ensure directory exists
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		return fmt.Errorf("create project directory: %w", err)
	}

	// Update timestamps
	project.Version = 2
	project.UpdatedAt = time.Now()
	if project.CreatedAt.IsZero() {
		project.CreatedAt = project.UpdatedAt
	}

	// Save config
	configPath := filepath.Join(projectDir, "config.yaml")
	data, err := yaml.Marshal(project)
	if err != nil {
		return fmt.Errorf("marshal project config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("write project config: %w", err)
	}

	return nil
}

// DeleteProject deletes a project by ID
func (pm *ProjectManager) DeleteProject(projectID string) error {
	projectDir := filepath.Join(pm.GetProjectsDir(), projectID)
	if err := os.RemoveAll(projectDir); err != nil {
		return fmt.Errorf("delete project directory: %w", err)
	}
	return nil
}

// GenerateProjectID generates a new unique project ID
func (pm *ProjectManager) GenerateProjectID() string {
	dateStr := time.Now().Format("20060102")

	projectsDir := pm.GetProjectsDir()
	entries, _ := os.ReadDir(projectsDir)

	maxSeq := 0
	prefix := fmt.Sprintf("project-%s-", dateStr)
	re := regexp.MustCompile(`project-(\d{8})-(\d+)`)

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, prefix) {
			matches := re.FindStringSubmatch(name)
			if len(matches) == 3 {
				if seq, err := strconv.Atoi(matches[2]); err == nil && seq > maxSeq {
					maxSeq = seq
				}
			}
		}
	}

	return fmt.Sprintf("%s%03d", prefix, maxSeq+1)
}

// ProjectExists checks if a project exists
func (pm *ProjectManager) ProjectExists(projectID string) bool {
	projectDir := filepath.Join(pm.GetProjectsDir(), projectID)
	_, err := os.Stat(projectDir)
	return err == nil
}

// GetProjectSummary returns a one-line summary of the project
func (p *TrainProject) GetProjectSummary() string {
	var parts []string

	switch p.Mode {
	case ModeCompare:
		if p.NPUConfig != nil {
			parts = append(parts, fmt.Sprintf("NPU: %s", p.NPUConfig.HostAddress))
		}
		if p.GPUConfig != nil {
			parts = append(parts, fmt.Sprintf("GPU: %s", p.GPUConfig.HostAddress))
		}
	case ModeNPUOnly:
		if p.NPUConfig != nil {
			parts = append(parts, fmt.Sprintf("NPU: %s", p.NPUConfig.HostAddress))
		}
	case ModeGPUOnly:
		if p.GPUConfig != nil {
			parts = append(parts, fmt.Sprintf("GPU: %s", p.GPUConfig.HostAddress))
		}
	}

	return strings.Join(parts, " | ")
}

// NewDefaultProject creates a new project with default values
func NewDefaultProject(mode TrainMode) *TrainProject {
	project := &TrainProject{
		Version:               2,
		Mode:                  mode,
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
		RsyncCompress:         false,
		RsyncRespectGitIgnore: true,
		SyncParallelism:       0,
		Exclude:               []string{".git", ".cache", "__pycache__", "*.pyc"},
	}

	// Initialize platform configs based on mode
	switch mode {
	case ModeCompare:
		project.NPUConfig = &PlatformConfig{
			SSHPort:        22,
			LocalPath:      ".",
			RemoteCodePath: "~/workspace/project",
			RunBaseDir:     "~/workspace/runs",
		}
		project.GPUConfig = &PlatformConfig{
			SSHPort:        22,
			LocalPath:      ".",
			RemoteCodePath: "~/workspace/project",
			RunBaseDir:     "~/workspace/runs",
		}
	case ModeNPUOnly:
		project.NPUConfig = &PlatformConfig{
			SSHPort:        22,
			LocalPath:      ".",
			RemoteCodePath: "~/workspace/project",
			RunBaseDir:     "~/workspace/runs",
		}
	case ModeGPUOnly:
		project.GPUConfig = &PlatformConfig{
			SSHPort:        22,
			LocalPath:      ".",
			RemoteCodePath: "~/workspace/project",
			RunBaseDir:     "~/workspace/runs",
		}
	}

	return project
}

// Validate validates the project configuration
func (p *TrainProject) Validate() []string {
	var errors []string

	switch p.Mode {
	case ModeCompare:
		if p.NPUConfig == nil {
			errors = append(errors, "NPU配置不能为空")
		} else {
			errors = append(errors, validatePlatformConfig("NPU", p.NPUConfig)...)
		}
		if p.GPUConfig == nil {
			errors = append(errors, "GPU配置不能为空")
		} else {
			errors = append(errors, validatePlatformConfig("GPU", p.GPUConfig)...)
		}
	case ModeNPUOnly:
		if p.NPUConfig == nil {
			errors = append(errors, "NPU配置不能为空")
		} else {
			errors = append(errors, validatePlatformConfig("NPU", p.NPUConfig)...)
		}
	case ModeGPUOnly:
		if p.GPUConfig == nil {
			errors = append(errors, "GPU配置不能为空")
		} else {
			errors = append(errors, validatePlatformConfig("GPU", p.GPUConfig)...)
		}
	}

	return errors
}

func validatePlatformConfig(platform string, cfg *PlatformConfig) []string {
	var errors []string

	if cfg.HostAddress == "" {
		errors = append(errors, fmt.Sprintf("%s: 机器IP不能为空", platform))
	}
	if cfg.User == "" {
		errors = append(errors, fmt.Sprintf("%s: 用户名不能为空", platform))
	}
	if cfg.LocalPath == "" {
		errors = append(errors, fmt.Sprintf("%s: 本地工程目录不能为空", platform))
	}
	if cfg.TrainScript == "" {
		errors = append(errors, fmt.Sprintf("%s: 启动脚本不能为空", platform))
	}

	return errors
}

// ToTrainingHosts converts NPU and GPU configs to TrainingHostConfig slice
func (p *TrainProject) ToTrainingHosts() []configs.TrainingHostConfig {
	var hosts []configs.TrainingHostConfig

	if p.NPUConfig != nil {
		hosts = append(hosts, configs.TrainingHostConfig{
			Name:           "NPU",
			User:           p.NPUConfig.User,
			Address:        p.NPUConfig.HostAddress,
			LocalPath:      p.NPUConfig.LocalPath,
			TrainScript:    p.NPUConfig.TrainScript,
			StartupCommand: p.NPUConfig.StartupCommand,
			RemoteCodePath: p.NPUConfig.RemoteCodePath,
			RunBaseDir:     p.NPUConfig.RunBaseDir,
			TrainCommand:   buildTrainCommand(p.NPUConfig),
		})
	}

	if p.GPUConfig != nil {
		hosts = append(hosts, configs.TrainingHostConfig{
			Name:           "GPU",
			User:           p.GPUConfig.User,
			Address:        p.GPUConfig.HostAddress,
			LocalPath:      p.GPUConfig.LocalPath,
			TrainScript:    p.GPUConfig.TrainScript,
			StartupCommand: p.GPUConfig.StartupCommand,
			RemoteCodePath: p.GPUConfig.RemoteCodePath,
			RunBaseDir:     p.GPUConfig.RunBaseDir,
			TrainCommand:   buildTrainCommand(p.GPUConfig),
		})
	}

	return hosts
}

func buildTrainCommand(cfg *PlatformConfig) string {
	if cfg.TrainArgs != "" {
		return fmt.Sprintf("python -u %s %s", cfg.TrainScript, cfg.TrainArgs)
	}
	return fmt.Sprintf("python -u %s", cfg.TrainScript)
}
