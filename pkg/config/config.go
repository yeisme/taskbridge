// Package config 提供配置管理功能
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/viper"
)

// Config 应用配置
type Config struct {
	App          AppConfig          `mapstructure:"app"`
	Storage      StorageConfig      `mapstructure:"storage"`
	Sync         SyncConfig         `mapstructure:"sync"`
	Intelligence IntelligenceConfig `mapstructure:"intelligence"`
	Providers    ProvidersConfig    `mapstructure:"providers"`
	Templates    TemplatesConfig    `mapstructure:"templates"`
}

// AppConfig 应用配置
type AppConfig struct {
	Name     string `mapstructure:"name"`
	Version  string `mapstructure:"version"`
	LogLevel string `mapstructure:"log_level"`
}

// StorageConfig 存储配置
type StorageConfig struct {
	Type string `mapstructure:"type"` // file, mongodb
	Path string `mapstructure:"path"`

	File  FileStorageConfig  `mapstructure:"file"`
	NoSQL NoSQLStorageConfig `mapstructure:"nosql"`
}

// FileStorageConfig 文件存储配置
type FileStorageConfig struct {
	Format   string `mapstructure:"format"`   // json, markdown
	Template string `mapstructure:"template"` // 自定义模板路径
}

// NoSQLStorageConfig NoSQL 存储配置
type NoSQLStorageConfig struct {
	// URL 数据库连接 URL
	// 格式示例：
	// - MongoDB: mongodb://localhost:27017/taskbridge
	URL string `mapstructure:"url"`

	// Database 数据库名称（用于 MongoDB 等）
	Database string `mapstructure:"database"`

	// Collection 集合名称（用于 MongoDB 等）
	Collection string `mapstructure:"collection"`
}

// SyncConfig 同步配置
type SyncConfig struct {
	Mode               string        `mapstructure:"mode"`                // once, interval, realtime
	Interval           time.Duration `mapstructure:"interval"`            // 同步间隔
	ConflictResolution string        `mapstructure:"conflict_resolution"` // local_wins, remote_wins, newer_wins, manual
	RetryCount         int           `mapstructure:"retry_count"`
	RetryDelay         time.Duration `mapstructure:"retry_delay"`
}

// IntelligenceConfig CLI 智能治理配置
type IntelligenceConfig struct {
	Enabled     bool              `mapstructure:"enabled"`
	Timezone    string            `mapstructure:"timezone"`
	Overdue     OverdueConfig     `mapstructure:"overdue"`
	LongTerm    LongTermConfig    `mapstructure:"long_term"`
	Decompose   DecomposeConfig   `mapstructure:"decomposition"`
	Achievement AchievementConfig `mapstructure:"achievement"`
}

// OverdueConfig 逾期治理配置
type OverdueConfig struct {
	WarningThreshold  int  `mapstructure:"warning_threshold"`
	OverloadThreshold int  `mapstructure:"overload_threshold"`
	SevereDays        int  `mapstructure:"severe_days"`
	AskBeforeDelete   bool `mapstructure:"ask_before_delete"`
	MaxCandidates     int  `mapstructure:"max_candidates"`
}

// LongTermConfig 长期任务调配配置
type LongTermConfig struct {
	MinAgeDays               int    `mapstructure:"min_age_days"`
	ShortTermWindowDays      int    `mapstructure:"short_term_window_days"`
	ShortTermMin             int    `mapstructure:"short_term_min"`
	ShortTermMax             int    `mapstructure:"short_term_max"`
	PromoteCountWhenShortage int    `mapstructure:"promote_count_when_shortage"`
	RetainCountWhenOverflow  int    `mapstructure:"retain_count_when_overflow"`
	OverflowStrategy         string `mapstructure:"overflow_strategy"`
}

// DecomposeConfig 复杂任务拆分配置
type DecomposeConfig struct {
	ComplexityThreshold    int      `mapstructure:"complexity_threshold"`
	DetectAbstractKeywords bool     `mapstructure:"detect_abstract_keywords"`
	AskBeforeSplit         bool     `mapstructure:"ask_before_split"`
	PreferredStrategy      string   `mapstructure:"preferred_strategy"`
	AbstractKeywords       []string `mapstructure:"abstract_keywords"`
}

// AchievementConfig 成就分析配置
type AchievementConfig struct {
	SnapshotGranularity   string `mapstructure:"snapshot_granularity"`
	StreakGoalPerDay      int    `mapstructure:"streak_goal_per_day"`
	BadgeEnabled          bool   `mapstructure:"badge_enabled"`
	NarrativeEnabled      bool   `mapstructure:"narrative_enabled"`
	ComparePreviousPeriod bool   `mapstructure:"compare_previous_period"`
}

// ProvidersConfig Provider 配置
type ProvidersConfig struct {
	Microsoft ProviderConfig `mapstructure:"microsoft"`
	Google    ProviderConfig `mapstructure:"google"`
	Feishu    ProviderConfig `mapstructure:"feishu"`
	TickTick  ProviderConfig `mapstructure:"ticktick"`
	Dida      ProviderConfig `mapstructure:"dida"`
	Todoist   ProviderConfig `mapstructure:"todoist"`
	OmniFocus ProviderConfig `mapstructure:"omnifocus"`
	Apple     ProviderConfig `mapstructure:"apple"`
}

// ProviderConfig 单个 Provider 的配置
type ProviderConfig struct {
	Enabled         bool                   `mapstructure:"enabled"`
	ClientID        string                 `mapstructure:"clientid"`
	ClientSecret    string                 `mapstructure:"clientsecret"`
	TenantID        string                 `mapstructure:"tenantid"`
	AppID           string                 `mapstructure:"appid"`
	AppSecret       string                 `mapstructure:"appsecret"`
	APIKey          string                 `mapstructure:"apikey"`
	APIToken        string                 `mapstructure:"apitoken"`
	DatabaseID      string                 `mapstructure:"databaseid"`
	Username        string                 `mapstructure:"username"`
	Password        string                 `mapstructure:"password"`
	CredentialsFile string                 `mapstructure:"credentialsfile"`
	Transport       string                 `mapstructure:"transport"`
	ListNames       []string               `mapstructure:"listnames"`
	Extra           map[string]interface{} `mapstructure:",remain"`
}

// TemplatesConfig 模板配置
type TemplatesConfig struct {
	JSON     TemplateConfig `mapstructure:"json"`
	Markdown TemplateConfig `mapstructure:"markdown"`
}

// TemplateConfig 单个模板的配置
type TemplateConfig struct {
	Path string `mapstructure:"path"`
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		App: AppConfig{
			Name:     "taskbridge",
			Version:  "1.0.1",
			LogLevel: "info",
		},
		Storage: StorageConfig{
			Type: "file",
			Path: "./data",
			File: FileStorageConfig{
				Format:   "json",
				Template: "",
			},
			NoSQL: NoSQLStorageConfig{
				URL:        "mongodb://localhost:27017/taskbridge",
				Database:   "taskbridge",
				Collection: "tasks",
			},
		},
		Sync: SyncConfig{
			Mode:               "interval",
			Interval:           5 * time.Minute,
			ConflictResolution: "newer_wins",
			RetryCount:         3,
			RetryDelay:         1 * time.Second,
		},
		Intelligence: IntelligenceConfig{
			Enabled:  true,
			Timezone: "Asia/Shanghai",
			Overdue: OverdueConfig{
				WarningThreshold:  3,
				OverloadThreshold: 10,
				SevereDays:        7,
				AskBeforeDelete:   true,
				MaxCandidates:     30,
			},
			LongTerm: LongTermConfig{
				MinAgeDays:               7,
				ShortTermWindowDays:      7,
				ShortTermMin:             5,
				ShortTermMax:             10,
				PromoteCountWhenShortage: 3,
				RetainCountWhenOverflow:  1,
				OverflowStrategy:         "defer",
			},
			Decompose: DecomposeConfig{
				ComplexityThreshold:    60,
				DetectAbstractKeywords: true,
				AskBeforeSplit:         true,
				PreferredStrategy:      "project_split",
				AbstractKeywords: []string{
					"优化", "推进", "完善", "研究", "整理",
				},
			},
			Achievement: AchievementConfig{
				SnapshotGranularity:   "daily",
				StreakGoalPerDay:      1,
				BadgeEnabled:          true,
				NarrativeEnabled:      true,
				ComparePreviousPeriod: true,
			},
		},
		Providers: ProvidersConfig{
			Microsoft: ProviderConfig{Enabled: false},
			Google:    ProviderConfig{Enabled: false},
			Feishu:    ProviderConfig{Enabled: false},
			TickTick:  ProviderConfig{Enabled: false},
			Dida:      ProviderConfig{Enabled: false},
			Todoist:   ProviderConfig{Enabled: false},
			OmniFocus: ProviderConfig{Enabled: false},
			Apple:     ProviderConfig{Enabled: false},
		},
		Templates: TemplatesConfig{
			JSON:     TemplateConfig{Path: "./templates/json/default.json"},
			Markdown: TemplateConfig{Path: "./templates/markdown/default.md"},
		},
	}
}

// Load 加载配置
func Load(configPath string) (*Config, error) {
	v := viper.New()

	// 设置默认值
	defaultCfg := DefaultConfig()
	setDefaults(v, defaultCfg)

	// 设置配置文件
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		// 查找配置文件
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath("./configs")
		// 添加 HOME 目录支持
		if homeDir, err := os.UserHomeDir(); err == nil {
			v.AddConfigPath(filepath.Join(homeDir, ".taskbridge"))
		}
		v.AddConfigPath("/etc/taskbridge")
	}

	// 读取环境变量
	v.SetEnvPrefix("TASKBRIDGE")
	v.AutomaticEnv()

	// 读取配置文件
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
		// 配置文件不存在，使用默认值
	}

	// 解析配置
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	return &cfg, nil
}

// setDefaults 设置默认值
func setDefaults(v *viper.Viper, cfg *Config) {
	v.SetDefault("app.name", cfg.App.Name)
	v.SetDefault("app.version", cfg.App.Version)
	v.SetDefault("app.log_level", cfg.App.LogLevel)

	v.SetDefault("storage.type", cfg.Storage.Type)
	v.SetDefault("storage.path", cfg.Storage.Path)
	v.SetDefault("storage.file.format", cfg.Storage.File.Format)
	v.SetDefault("storage.nosql.url", cfg.Storage.NoSQL.URL)
	v.SetDefault("storage.nosql.database", cfg.Storage.NoSQL.Database)
	v.SetDefault("storage.nosql.collection", cfg.Storage.NoSQL.Collection)

	v.SetDefault("sync.mode", cfg.Sync.Mode)
	v.SetDefault("sync.interval", cfg.Sync.Interval)
	v.SetDefault("sync.conflict_resolution", cfg.Sync.ConflictResolution)
	v.SetDefault("sync.retry_count", cfg.Sync.RetryCount)
	v.SetDefault("sync.retry_delay", cfg.Sync.RetryDelay)

	v.SetDefault("intelligence.enabled", cfg.Intelligence.Enabled)
	v.SetDefault("intelligence.timezone", cfg.Intelligence.Timezone)
	v.SetDefault("intelligence.overdue.warning_threshold", cfg.Intelligence.Overdue.WarningThreshold)
	v.SetDefault("intelligence.overdue.overload_threshold", cfg.Intelligence.Overdue.OverloadThreshold)
	v.SetDefault("intelligence.overdue.severe_days", cfg.Intelligence.Overdue.SevereDays)
	v.SetDefault("intelligence.overdue.ask_before_delete", cfg.Intelligence.Overdue.AskBeforeDelete)
	v.SetDefault("intelligence.overdue.max_candidates", cfg.Intelligence.Overdue.MaxCandidates)
	v.SetDefault("intelligence.long_term.min_age_days", cfg.Intelligence.LongTerm.MinAgeDays)
	v.SetDefault("intelligence.long_term.short_term_window_days", cfg.Intelligence.LongTerm.ShortTermWindowDays)
	v.SetDefault("intelligence.long_term.short_term_min", cfg.Intelligence.LongTerm.ShortTermMin)
	v.SetDefault("intelligence.long_term.short_term_max", cfg.Intelligence.LongTerm.ShortTermMax)
	v.SetDefault("intelligence.long_term.promote_count_when_shortage", cfg.Intelligence.LongTerm.PromoteCountWhenShortage)
	v.SetDefault("intelligence.long_term.retain_count_when_overflow", cfg.Intelligence.LongTerm.RetainCountWhenOverflow)
	v.SetDefault("intelligence.long_term.overflow_strategy", cfg.Intelligence.LongTerm.OverflowStrategy)
	v.SetDefault("intelligence.decomposition.complexity_threshold", cfg.Intelligence.Decompose.ComplexityThreshold)
	v.SetDefault("intelligence.decomposition.detect_abstract_keywords", cfg.Intelligence.Decompose.DetectAbstractKeywords)
	v.SetDefault("intelligence.decomposition.ask_before_split", cfg.Intelligence.Decompose.AskBeforeSplit)
	v.SetDefault("intelligence.decomposition.preferred_strategy", cfg.Intelligence.Decompose.PreferredStrategy)
	v.SetDefault("intelligence.decomposition.abstract_keywords", cfg.Intelligence.Decompose.AbstractKeywords)
	v.SetDefault("intelligence.achievement.snapshot_granularity", cfg.Intelligence.Achievement.SnapshotGranularity)
	v.SetDefault("intelligence.achievement.streak_goal_per_day", cfg.Intelligence.Achievement.StreakGoalPerDay)
	v.SetDefault("intelligence.achievement.badge_enabled", cfg.Intelligence.Achievement.BadgeEnabled)
	v.SetDefault("intelligence.achievement.narrative_enabled", cfg.Intelligence.Achievement.NarrativeEnabled)
	v.SetDefault("intelligence.achievement.compare_previous_period", cfg.Intelligence.Achievement.ComparePreviousPeriod)
}

// Save 保存配置到文件
func Save(cfg *Config, path string) error {
	v := viper.New()
	v.Set("app", cfg.App)
	v.Set("storage", cfg.Storage)
	v.Set("sync", cfg.Sync)
	v.Set("intelligence", cfg.Intelligence)
	v.Set("providers", cfg.Providers)
	v.Set("templates", cfg.Templates)

	// 确保目录存在
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("error creating config directory: %w", err)
	}

	return v.WriteConfigAs(path)
}

// GetConfigPath 获取配置文件路径
// 优先级:
// 1. 环境变量 TASKBRIDGE_CONFIG
// 2. HOME 目录 ~/.taskbridge/config.yaml
// 3. 当前目录 ./config.yaml
// 4. ./configs/config.yaml
// 5. /etc/taskbridge/config.yaml
func GetConfigPath() string {
	// 优先使用环境变量
	if path := os.Getenv("TASKBRIDGE_CONFIG"); path != "" {
		return path
	}

	// 获取 HOME 目录
	homeDir, err := os.UserHomeDir()
	if err == nil {
		homeConfig := filepath.Join(homeDir, ".taskbridge", "config.yaml")
		if _, err := os.Stat(homeConfig); err == nil {
			return homeConfig
		}
	}

	// 检查其他默认位置
	paths := []string{
		"./config.yaml",
		"./configs/config.yaml",
		"/etc/taskbridge/config.yaml",
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	// 返回 HOME 目录作为默认路径（即使不存在）
	if homeDir != "" {
		return filepath.Join(homeDir, ".taskbridge", "config.yaml")
	}

	return "./configs/config.yaml"
}

// GetDefaultConfigPath 获取默认配置文件路径（用于 init 命令）
// 始终返回 HOME 目录下的路径
func GetDefaultConfigPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "./configs/config.yaml"
	}
	return filepath.Join(homeDir, ".taskbridge", "config.yaml")
}
