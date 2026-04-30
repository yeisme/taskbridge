package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/yeisme/taskbridge/internal/controlplane"
	"github.com/yeisme/taskbridge/internal/project"
	"github.com/yeisme/taskbridge/internal/storage/filestore"
)

var (
	doctorFormat     string
	demoFormat       string
	quickstartFormat string
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "检查 TaskBridge 本地环境",
	RunE:  runDoctor,
}

var demoCmd = &cobra.Command{
	Use:   "demo",
	Short: "运行内置示例",
}

var demoTodayCmd = &cobra.Command{
	Use:   "today",
	Short: "用示例数据展示每日任务工作台",
	RunE:  runDemoToday,
}

var quickstartCmd = &cobra.Command{
	Use:   "quickstart",
	Short: "根据当前状态给出下一条建议命令",
	RunE:  runQuickstart,
}

func init() {
	rootCmd.AddCommand(doctorCmd)
	rootCmd.AddCommand(demoCmd)
	rootCmd.AddCommand(quickstartCmd)
	demoCmd.AddCommand(demoTodayCmd)

	doctorCmd.Flags().StringVarP(&doctorFormat, "format", "f", "text", "输出格式 (text, json)")
	demoTodayCmd.Flags().StringVarP(&demoFormat, "format", "f", "text", "输出格式 (text, json)")
	quickstartCmd.Flags().StringVarP(&quickstartFormat, "format", "f", "text", "输出格式 (text, json)")
}

func runDoctor(_ *cobra.Command, _ []string) error {
	result := buildDoctorResult()
	return printStructured(doctorFormat, result, func() {
		fmt.Println("TaskBridge doctor")
		for _, check := range result.Checks {
			fmt.Printf("- [%s] %s: %s\n", check.Status, check.ID, check.Message)
			if check.NextAction != "" {
				fmt.Printf("  next: %s\n", check.NextAction)
			}
		}
		if result.NextAction != "" {
			fmt.Printf("\n下一步: %s\n", result.NextAction)
		}
	})
}

func runQuickstart(_ *cobra.Command, _ []string) error {
	result := buildDoctorResult()
	payload := map[string]interface{}{
		"schema":      "taskbridge.quickstart.v1",
		"status":      result.Status,
		"next_action": result.NextAction,
		"checks":      result.Checks,
	}
	return printStructured(quickstartFormat, payload, func() {
		if result.NextAction == "" {
			fmt.Println("TaskBridge 已可用。建议运行: taskbridge today")
			return
		}
		fmt.Println("建议下一步:")
		fmt.Println(result.NextAction)
	})
}

func runDemoToday(_ *cobra.Command, _ []string) error {
	ctx := context.Background()
	tmp, err := os.MkdirTemp("", "taskbridge-demo-*")
	if err != nil {
		return commandError("创建 demo 存储失败", err)
	}
	defer os.RemoveAll(tmp)

	store, err := filestore.New(tmp, "json")
	if err != nil {
		return commandError("初始化 demo 存储失败", err)
	}
	current := time.Now()
	for _, task := range controlplane.DemoTasks(current) {
		t := task
		if err := store.SaveTask(ctx, &t); err != nil {
			return commandError("写入 demo 任务失败", err)
		}
	}
	if err := store.Flush(); err != nil {
		return commandError("刷新 demo 存储失败", err)
	}

	service := controlplane.Service{TaskStore: store}
	result, err := service.Today(ctx, controlplane.Options{Now: current})
	if err != nil {
		return commandError("生成 demo today 失败", err)
	}
	return printTodayResult(demoFormat, result)
}

func buildDoctorResult() controlplane.DoctorResult {
	checks := make([]controlplane.DoctorCheck, 0)
	status := "ok"
	add := func(check controlplane.DoctorCheck) {
		checks = append(checks, check)
		if check.Status == "error" {
			status = "error"
		} else if check.Status == "warning" && status == "ok" {
			status = "warning"
		}
	}

	if cfg == nil {
		add(controlplane.DoctorCheck{ID: "config", Status: "error", Message: "配置未初始化", NextAction: "重新运行 taskbridge"})
		return controlplane.DoctorResult{Schema: controlplane.SchemaDoctor, Status: "error", Checks: checks, NextAction: "重新运行 taskbridge"}
	}

	if cfg.Storage.Path == "" {
		add(controlplane.DoctorCheck{ID: "storage_path", Status: "error", Message: "storage path 为空", NextAction: "设置 TASKBRIDGE_STORAGE_PATH"})
	} else if err := os.MkdirAll(cfg.Storage.Path, 0o755); err != nil {
		add(controlplane.DoctorCheck{ID: "storage_path", Status: "error", Message: err.Error(), NextAction: "检查 storage path 权限"})
	} else if file, err := os.CreateTemp(cfg.Storage.Path, ".taskbridge-write-check-*"); err != nil {
		add(controlplane.DoctorCheck{ID: "storage_path", Status: "error", Message: err.Error(), NextAction: "检查 storage path 写权限"})
	} else {
		name := file.Name()
		_ = file.Close()
		_ = os.Remove(name)
		add(controlplane.DoctorCheck{ID: "storage_path", Status: "ok", Message: fmt.Sprintf("storage path is writable: %s", cfg.Storage.Path)})
	}

	if _, err := project.NewFileStore(cfg.Storage.Path); err != nil {
		add(controlplane.DoctorCheck{ID: "project_store", Status: "warning", Message: err.Error(), NextAction: "检查 projects.json 或 storage path"})
	} else {
		add(controlplane.DoctorCheck{ID: "project_store", Status: "ok", Message: "project store is readable"})
	}

	authenticated := 0
	for _, name := range getAuthProviderOrder() {
		snapshot := getProviderAuthSnapshot(name)
		if snapshot.Authenticated {
			authenticated++
		}
	}
	if authenticated == 0 {
		add(controlplane.DoctorCheck{ID: "provider_auth", Status: "warning", Message: "no provider authenticated", NextAction: "taskbridge demo today"})
	} else {
		add(controlplane.DoctorCheck{ID: "provider_auth", Status: "ok", Message: fmt.Sprintf("%d provider(s) authenticated", authenticated)})
	}

	next := "taskbridge today"
	if status != "ok" || authenticated == 0 {
		next = "taskbridge demo today"
	}
	return controlplane.DoctorResult{Schema: controlplane.SchemaDoctor, Status: status, Checks: checks, NextAction: next}
}
