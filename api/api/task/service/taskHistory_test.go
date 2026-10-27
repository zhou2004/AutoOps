package service

import (
	"strings"
	"testing"
	"time"

	"dodevops-api/api/task/dao"
	"dodevops-api/api/task/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestCommandRunnerSupportsConfiguredLanguages(t *testing.T) {
	tests := []struct {
		templateType int
		extension    string
		contains     string
	}{
		{model.TemplateTypeShell, ".sh", "bash"},
		{model.TemplateTypePython, ".py", "python3"},
		{model.TemplateTypeAnsible, ".yml", "ansible-playbook"},
		{model.TemplateTypeJavaScript, ".js", "node"},
		{model.TemplateTypePowerShell, ".ps1", "pwsh"},
		{model.TemplateTypeGo, ".go", "go run"},
		{model.TemplateTypeRuby, ".rb", "ruby"},
		{model.TemplateTypePHP, ".php", "php"},
		{model.TemplateTypePerl, ".pl", "perl"},
		{model.TemplateTypeCommand, ".command", "bash -lc"},
	}
	for _, test := range tests {
		extension, runner, err := commandRunner(test.templateType, "/tmp/task", "echo 'ok'")
		if err != nil {
			t.Fatalf("type %d: %v", test.templateType, err)
		}
		if extension != test.extension || !strings.Contains(runner, test.contains) {
			t.Fatalf("type %d: got extension=%q runner=%q", test.templateType, extension, runner)
		}
	}
	if _, _, err := commandRunner(999, "/tmp/task", ""); err == nil {
		t.Fatal("unsupported language should fail")
	}
}

func TestTaskHistoryRollbackRestoresExecutedTemplateSnapshot(t *testing.T) {
	db := newTaskHistoryTestDB(t)
	template := model.TaskTemplate{Name: "start", Type: model.TemplateTypeShell, Content: "bash start.sh"}
	if err := db.Create(&template).Error; err != nil {
		t.Fatal(err)
	}
	task := model.Task{
		Name: "deploy", Type: model.TaskTypeImmediate, ExecutionMode: 1,
		Shell: "1", HostIDs: "1", Status: model.TaskStatusPending, MaxHistoryKeep: 3,
	}
	if err := dao.NewTaskDao(db).Create(&task); err != nil {
		t.Fatal(err)
	}

	historyService := NewTaskHistoryService(db)
	history, works, err := historyService.BeginExecution(task.ID, model.TaskTriggerManual, "tester")
	if err != nil {
		t.Fatal(err)
	}
	if len(works) != 1 {
		t.Fatalf("expected one work, got %d", len(works))
	}
	if err := db.Model(&template).Update("content", "bash start2.sh").Error; err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	if err := historyService.CompleteLog(&works[0], model.TaskStatusSuccess, "ok", "", now, now); err != nil {
		t.Fatal(err)
	}
	if err := historyService.FinalizeExecution(history.ID); err != nil {
		t.Fatal(err)
	}
	if err := historyService.Rollback(task.ID, history.ID, "tester"); err != nil {
		t.Fatal(err)
	}

	var restored model.TaskTemplate
	if err := db.First(&restored, template.ID).Error; err != nil {
		t.Fatal(err)
	}
	if restored.Content != "bash start.sh" {
		t.Fatalf("expected executed snapshot, got %q", restored.Content)
	}
	var restoredHistory model.TaskExecutionHistory
	if err := db.First(&restoredHistory, history.ID).Error; err != nil {
		t.Fatal(err)
	}
	if restoredHistory.RolledBackAt == nil || restoredHistory.RolledBackBy != "tester" {
		t.Fatalf("rollback audit not recorded: %#v", restoredHistory)
	}
}

func TestTaskHistoryRetention(t *testing.T) {
	db := newTaskHistoryTestDB(t)
	task := model.Task{
		Name: "command", Type: model.TaskTypeImmediate, ExecutionMode: 2,
		Command: "echo ok", CommandType: model.TemplateTypeCommand,
		HostIDs: "1", Status: model.TaskStatusPending, MaxHistoryKeep: 2,
	}
	if err := dao.NewTaskDao(db).Create(&task); err != nil {
		t.Fatal(err)
	}
	historyService := NewTaskHistoryService(db)
	for i := 0; i < 3; i++ {
		history, works, err := historyService.BeginExecution(task.ID, model.TaskTriggerManual, "tester")
		if err != nil {
			t.Fatal(err)
		}
		now := time.Now()
		if err := historyService.CompleteLog(&works[0], model.TaskStatusSuccess, "ok", "", now, now); err != nil {
			t.Fatal(err)
		}
		if err := historyService.FinalizeExecution(history.ID); err != nil {
			t.Fatal(err)
		}
	}
	var historyCount, logCount int64
	if err := db.Model(&model.TaskExecutionHistory{}).Where("task_id = ?", task.ID).Count(&historyCount).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&model.TaskExecutionLog{}).Where("task_id = ?", task.ID).Count(&logCount).Error; err != nil {
		t.Fatal(err)
	}
	if historyCount != 2 || logCount != 2 {
		t.Fatalf("retention failed: histories=%d logs=%d", historyCount, logCount)
	}
}

func TestPausedExecutionRemainsOpenAndStoppedExecutionFinalizes(t *testing.T) {
	db := newTaskHistoryTestDB(t)
	task := model.Task{
		Name: "long-running", Type: model.TaskTypeImmediate, ExecutionMode: 2,
		Command: "sleep 60", CommandType: model.TemplateTypeCommand,
		HostIDs: "1", Status: model.TaskStatusPending, MaxHistoryKeep: 5,
	}
	if err := dao.NewTaskDao(db).Create(&task); err != nil {
		t.Fatal(err)
	}
	historyService := NewTaskHistoryService(db)
	history, works, err := historyService.BeginExecution(task.ID, model.TaskTriggerManual, "tester")
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&model.TaskWork{}).Where("id = ?", works[0].ID).Update("status", model.TaskStatusPaused).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&model.TaskExecutionLog{}).Where("history_id = ?", history.ID).Update("status", model.TaskStatusPaused).Error; err != nil {
		t.Fatal(err)
	}
	if err := historyService.FinalizeExecution(history.ID); err != nil {
		t.Fatal(err)
	}
	var pausedHistory model.TaskExecutionHistory
	if err := db.First(&pausedHistory, history.ID).Error; err != nil {
		t.Fatal(err)
	}
	if pausedHistory.FinishedAt != nil {
		t.Fatal("paused execution must remain unfinished")
	}

	works[0].HistoryID = history.ID
	now := time.Now()
	if err := historyService.CompleteLog(&works[0], model.TaskStatusStopped, "stopped by tester", "", now, now); err != nil {
		t.Fatal(err)
	}
	if err := historyService.FinalizeExecution(history.ID); err != nil {
		t.Fatal(err)
	}
	var stoppedHistory model.TaskExecutionHistory
	if err := db.First(&stoppedHistory, history.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stoppedHistory.Status != model.TaskStatusStopped || stoppedHistory.FinishedAt == nil {
		t.Fatalf("stopped execution was not finalized: %#v", stoppedHistory)
	}
	var stoppedTask model.Task
	if err := db.First(&stoppedTask, task.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stoppedTask.Status != model.TaskStatusStopped {
		t.Fatalf("expected stopped task status, got %d", stoppedTask.Status)
	}
}

func newTaskHistoryTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Task{}, &model.TaskWork{}, &model.TaskTemplate{}, &model.TaskExecutionHistory{}, &model.TaskExecutionLog{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("CREATE TABLE cmdb_host (id integer primary key, host_name text, ssh_ip text)").Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("INSERT INTO cmdb_host (id, host_name, ssh_ip) VALUES (1, 'server-1', '127.0.0.1')").Error; err != nil {
		t.Fatal(err)
	}
	return db
}
