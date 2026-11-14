package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"dodevops-api/api/task/dao"
	"dodevops-api/api/task/model"
	"github.com/gin-gonic/gin"
)

func TestAnsibleMetadataEditRetainsPlaybooksAndConfiguration(t *testing.T) {
	db := newTaskHistoryTestDB(t)
	if err := db.AutoMigrate(&model.TaskAnsible{}, &model.TaskAnsibleWork{}); err != nil {
		t.Fatal(err)
	}
	task := model.TaskAnsible{Name: "old-name", Description: "old description", Type: 2, UseConfig: 1, ExtraVars: "old: true", CliArgs: "--check", Status: 1}
	if err := db.Create(&task).Error; err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	work := model.TaskAnsibleWork{TaskID: task.ID, EntryFileName: "playbook.yml", EntryFilePath: filepath.Join(dir, "playbook.yml"), Status: 3}
	if err := db.Create(&work).Error; err != nil {
		t.Fatal(err)
	}
	svc := NewTaskAnsibleService(db)
	update := func(payload string, want int) {
		t.Helper()
		var req UpdateTaskRequest
		if err := json.Unmarshal([]byte(payload), &req); err != nil {
			t.Fatal(err)
		}
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		svc.UpdateTask(ctx, task.ID, &req)
		var response struct {
			Code int `json:"code"`
		}
		if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
			t.Fatal(err)
		}
		if response.Code != want {
			t.Fatalf("unexpected response: %s", recorder.Body.String())
		}
	}
	// The unchanged playbook need not be locally available for a metadata edit.
	update(`{"name":"new-name","description":"new description","playbookPaths":["playbook.yml"]}`, 200)
	var saved model.TaskAnsible
	db.Preload("Works").First(&saved, task.ID)
	if saved.Description != "new description" || saved.UseConfig != 1 || len(saved.Works) != 1 || saved.Works[0].ID != work.ID {
		t.Fatalf("metadata update damaged task: %#v", saved)
	}
	if ansibleTaskDirectory(&saved) != dir {
		t.Fatal("rename must retain the original directory")
	}
	update(`{"description":"","extraVars":"","cliArgs":"","variables":{},"cronExpr":"","viewId":null}`, 200)
	db.First(&saved, task.ID)
	if saved.Description != "" || saved.ExtraVars != "" || saved.CliArgs != "" || saved.GlobalVars != "{}" {
		t.Fatalf("clearing fields failed: %#v", saved)
	}
	update(`{"description":"must not save","playbookPaths":["missing.yml"]}`, 400)
	db.Preload("Works").First(&saved, task.ID)
	if saved.Description != "" || len(saved.Works) != 1 || saved.Works[0].ID != work.ID {
		t.Fatal("failed validation must leave task and work intact")
	}
	if err := os.WriteFile(filepath.Join(dir, "new.yml"), []byte("- hosts: all\n"), 0600); err != nil {
		t.Fatal(err)
	}
	update(`{"playbookPaths":["new.yml"]}`, 200)
	db.Preload("Works").First(&saved, task.ID)
	if len(saved.Works) != 1 || saved.Works[0].EntryFileName != "new.yml" || saved.TaskCount != 1 {
		t.Fatal("changed playbook was not saved")
	}
}

func TestTaskParametersSnapshotPrecedenceAndLiveLog(t *testing.T) {
	db := newTaskHistoryTestDB(t)
	template := model.TaskTemplate{Name: "deploy", Type: 1, Content: `printf '%s' "$APP_NAME"`, Parameters: map[string]string{"APP_NAME": "default", "VERSION": "v1", "EMPTY": "default"}}
	if err := db.Create(&template).Error; err != nil {
		t.Fatal(err)
	}
	task := model.Task{Name: "deploy-prod", Type: 1, ExecutionMode: 1, Shell: strconv.Itoa(int(template.ID)), HostIDs: "1", Status: 1, Parameters: map[string]string{"VERSION": "v2"}}
	if err := dao.NewTaskDao(db).Create(&task); err != nil {
		t.Fatal(err)
	}
	svc := NewTaskHistoryService(db)
	history, works, err := svc.BeginExecution(task.ID, 1, "tester", map[string]string{"APP_NAME": "orders", "EMPTY": ""})
	if err != nil {
		t.Fatal(err)
	}
	log, err := svc.GetExecutionLog(history.ID, works[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if log.Parameters["APP_NAME"] != "orders" || log.Parameters["VERSION"] != "v2" || log.Parameters["EMPTY"] != "" {
		t.Fatalf("incorrect precedence: %v", log.Parameters)
	}
	var persisted model.Task
	db.First(&persisted, task.ID)
	if len(persisted.Parameters) != 1 || persisted.Parameters["VERSION"] != "v2" {
		t.Fatal("runtime parameters modified saved task")
	}
	path := filepath.Join(t.TempDir(), "live.log")
	if err := os.WriteFile(path, []byte("第一行\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := svc.MarkLogStarted(history.ID, works[0].ID, time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := svc.PublishLogPath(&works[0], path); err != nil {
		t.Fatal(err)
	}
	detail, err := svc.Detail(task.ID, history.ID)
	if err != nil {
		t.Fatal(err)
	}
	if detail.FinishedAt != nil || detail.Logs[0].Log != "第一行\n" {
		t.Fatal("log must be visible before completion")
	}
	if err := os.WriteFile(path, []byte("第一行\n第二行\n"), 0600); err != nil {
		t.Fatal(err)
	}
	detail, err = svc.Detail(task.ID, history.ID)
	if err != nil || detail.Logs[0].Log != "第一行\n第二行\n" {
		t.Fatal("live log did not refresh")
	}
	now := time.Now()
	if err := svc.CompleteLog(&works[0], 3, detail.Logs[0].Log, path, now, now); err != nil {
		t.Fatal(err)
	}
	if err := svc.FinalizeExecution(history.ID); err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&template).Update("parameters", `{"APP_NAME":"changed"}`).Error; err != nil {
		t.Fatal(err)
	}
	if err := svc.Rollback(task.ID, history.ID, "tester"); err != nil {
		t.Fatal(err)
	}
	db.First(&template, template.ID)
	if template.Parameters["APP_NAME"] != "default" {
		t.Fatal("template parameter snapshot was not restored")
	}
}

func TestTaskParameterValuesAreNotEvaluatedAsShellCode(t *testing.T) {
	value := "it's $(printf injected); `printf injected`\n第二行"
	parameters := map[string]string{"APP_VALUE": value}
	if err := validateTaskParameters(parameters); err != nil {
		t.Fatal(err)
	}
	output, err := exec.Command("sh", "-c", taskParameterExports(parameters)+`printf '%s' "$APP_VALUE"`).Output()
	if err != nil || string(output) != value {
		t.Fatalf("parameter value changed: %q, %v", output, err)
	}
	if err := validateTaskParameters(map[string]string{"BAD;echo": "x"}); err == nil {
		t.Fatal("invalid parameter name accepted")
	}
}

func TestTaskLogFollowerStreamsBeforeCompletionAndDrainsTail(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "task.log")
	exitPath := filepath.Join(dir, "task.exit")
	if err := os.WriteFile(path, []byte("first\n"), 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "sh", "-c", taskLogFollowCommand(path, exitPath, fmt.Sprint(os.Getpid())))
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	first := make([]byte, len("first\n"))
	if _, err := io.ReadFull(stdout, first); err != nil {
		t.Fatal(err)
	}
	if string(first) != "first\n" {
		t.Fatalf("unexpected live output %q", first)
	}
	// Completion is signalled only after observing output from the running process.
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.WriteString("最后一行\n")
	f.Close()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(exitPath, []byte("0"), 0600); err != nil {
		t.Fatal(err)
	}
	rest, err := io.ReadAll(stdout)
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Wait(); err != nil {
		t.Fatal(err)
	}
	if string(rest) != "最后一行\n" {
		t.Fatalf("lost or duplicated trailing output: %q", rest)
	}
}

func TestTaskLogWindowIsBounded(t *testing.T) {
	path := filepath.Join(t.TempDir(), "large.log")
	if err := os.WriteFile(path, []byte(strings.Repeat("x", 3*1024*1024)+"END"), 0600); err != nil {
		t.Fatal(err)
	}
	content, err := readLiveTaskLog(path)
	if err != nil || !strings.HasSuffix(content, "END") || len(content) > 2*1024*1024+200 {
		t.Fatalf("invalid live window: %d, %v", len(content), err)
	}
}

func TestEditedAnsibleVariablesReplaceCreationTimeFile(t *testing.T) {
	dir := t.TempDir()
	if err := writeAnsibleGlobalVars(dir, `{"MESSAGE":"old"}`); err != nil {
		t.Fatal(err)
	}
	if err := writeAnsibleGlobalVars(dir, `{"MESSAGE":"a: b\nquoted \"value\""}`); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "vars", "all.yml"))
	if err != nil {
		t.Fatal(err)
	}
	var values map[string]string
	if err := json.Unmarshal(data, &values); err != nil {
		t.Fatal(err)
	}
	if values["MESSAGE"] != "a: b\nquoted \"value\"" {
		t.Fatalf("lost value formatting: %q", values["MESSAGE"])
	}
	if err := writeAnsibleGlobalVars(dir, "{}"); err != nil {
		t.Fatal(err)
	}
	data, _ = os.ReadFile(filepath.Join(dir, "vars", "all.yml"))
	if string(data) != "{}" {
		t.Fatal("cleared variables retained old values")
	}
}

func TestTaskAndTemplateParameterEditsCanClearValues(t *testing.T) {
	db := newTaskHistoryTestDB(t)
	svc := NewTaskService(db)
	task := model.Task{Name: "parameters", Type: 1, ExecutionMode: 2, Command: "echo ok", CommandType: 10, HostIDs: "1", Status: 1, Parameters: map[string]string{"VERSION": "v1"}}
	if err := svc.CreateTask(&task); err != nil {
		t.Fatal(err)
	}
	task.Parameters = map[string]string{"VERSION": "v2", "EXTRA": "added"}
	if err := svc.UpdateTask(task.ID, &task); err != nil {
		t.Fatal(err)
	}
	got, err := svc.GetTask(task.ID)
	if err != nil || got.Parameters["VERSION"] != "v2" {
		t.Fatalf("parameter edit failed: %v", err)
	}
	task.Parameters = map[string]string{}
	if err := svc.UpdateTask(task.ID, &task); err != nil {
		t.Fatal(err)
	}
	got, err = svc.GetTask(task.ID)
	if err != nil || len(got.Parameters) != 0 {
		t.Fatalf("clear task parameters failed: %v", err)
	}
	template := model.TaskTemplate{Name: "defaults", Type: 1, Content: "echo ok", Parameters: map[string]string{"VERSION": "v1"}}
	templateDAO := dao.NewTaskTemplateDao(db)
	if err := templateDAO.CreateTemplate(&template); err != nil {
		t.Fatal(err)
	}
	template.Parameters = map[string]string{}
	if err := templateDAO.UpdateTemplate(&template); err != nil {
		t.Fatal(err)
	}
	saved, err := templateDAO.GetTemplateByID(template.ID)
	if err != nil || len(saved.Parameters) != 0 {
		t.Fatalf("clear template parameters failed: %v", err)
	}
}
