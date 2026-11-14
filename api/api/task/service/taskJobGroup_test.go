package service

import (
	"testing"

	"dodevops-api/api/task/model"
)

func TestTaskJobGroupsFilterRenameAndDeleteWithoutDeletingJobs(t *testing.T) {
	db := newTaskHistoryTestDB(t)
	groups := NewTaskJobGroupService(db)
	prod, err := groups.Save(0, "  生产环境  ")
	if err != nil || prod.Name != "生产环境" {
		t.Fatalf("create group: %v", err)
	}
	dev, err := groups.Save(0, "测试环境")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := groups.Save(0, "生产环境"); err == nil {
		t.Fatal("duplicate name accepted")
	}
	if _, err := groups.Save(dev.ID, "生产环境"); err == nil {
		t.Fatal("duplicate rename accepted")
	}
	if _, err := groups.Save(0, "  "); err == nil {
		t.Fatal("blank group accepted")
	}

	jobs := NewTaskService(db)
	create := func(name string, groupID *uint) model.Task {
		t.Helper()
		task := model.Task{Name: name, Type: 1, ExecutionMode: 2, Command: "echo ok", CommandType: 10, HostIDs: "1", Status: 1, GroupID: groupID}
		if err := jobs.CreateTask(&task); err != nil {
			t.Fatal(err)
		}
		return task
	}
	first := create("production-one", &prod.ID)
	create("production-two", &prod.ID)
	create("development", &dev.ID)
	create("legacy", nil)
	invalid := uint(999)
	bad := model.Task{Name: "invalid", Type: 1, ExecutionMode: 2, Command: "echo ok", CommandType: 10, HostIDs: "1", GroupID: &invalid}
	if err := jobs.CreateTask(&bad); err == nil {
		t.Fatal("nonexistent group accepted")
	}
	list, total, err := jobs.ListTasksWithDetails(1, 1, "", 0, &prod.ID)
	if err != nil || total != 2 || len(list) != 1 || list[0].GroupName != "生产环境" {
		t.Fatalf("group pagination mismatch: %v, %d, %#v", err, total, list)
	}
	_, total, err = jobs.ListTasksWithDetails(1, 10, "one", 1, &prod.ID)
	if err != nil || total != 1 {
		t.Fatalf("combined filtering failed: %v, %d", err, total)
	}
	zero := uint(0)
	_, total, err = jobs.ListTasksWithDetails(1, 10, "", 0, &zero)
	if err != nil || total != 1 {
		t.Fatalf("ungrouped filtering failed: %v, %d", err, total)
	}
	if _, err := groups.Save(prod.ID, "线上环境"); err != nil {
		t.Fatal(err)
	}
	list, _, err = jobs.ListTasksWithDetails(1, 10, "", 0, &prod.ID)
	if err != nil || list[0].GroupName != "线上环境" {
		t.Fatalf("rename not reflected: %v", err)
	}
	summaries, err := groups.List()
	if err != nil || len(summaries) != 2 || summaries[0].TaskCount != 2 {
		t.Fatalf("group counts mismatch: %v, %#v", err, summaries)
	}

	// Removing a group must not disturb jobs, their workers or current execution.
	db.Model(&model.Task{}).Where("id = ?", first.ID).Update("status", model.TaskStatusRunning)
	if err := groups.Delete(prod.ID); err != nil {
		t.Fatal(err)
	}
	saved, err := jobs.GetTask(first.ID)
	if err != nil || saved.GroupID != nil || saved.Status != model.TaskStatusRunning || saved.Command != "echo ok" {
		t.Fatalf("group deletion changed job: %v, %#v", err, saved)
	}
	var workCount int64
	db.Model(&model.TaskWork{}).Count(&workCount)
	if workCount != 4 {
		t.Fatalf("group deletion changed work count: %d", workCount)
	}
	_, total, err = jobs.ListTasksWithDetails(1, 10, "", 0, &zero)
	if err != nil || total != 3 {
		t.Fatalf("deleted group tasks must become ungrouped: %v, %d", err, total)
	}
	if err := groups.Delete(prod.ID); err == nil {
		t.Fatal("deleting missing group should fail")
	}
}

func TestTaskGroupCanBeChangedAndCleared(t *testing.T) {
	db := newTaskHistoryTestDB(t)
	group, err := NewTaskJobGroupService(db).Save(0, "维护")
	if err != nil {
		t.Fatal(err)
	}
	svc := NewTaskService(db)
	task := model.Task{Name: "regroup", Type: 1, ExecutionMode: 2, Command: "echo ok", CommandType: 10, HostIDs: "1", Status: 1}
	if err := svc.CreateTask(&task); err != nil {
		t.Fatal(err)
	}
	task.GroupID = &group.ID
	if err := svc.UpdateTask(task.ID, &task); err != nil {
		t.Fatal(err)
	}
	saved, err := svc.GetTask(task.ID)
	if err != nil || saved.GroupID == nil || *saved.GroupID != group.ID {
		t.Fatalf("assignment failed: %v", err)
	}
	task.GroupID = nil
	if err := svc.UpdateTask(task.ID, &task); err != nil {
		t.Fatal(err)
	}
	saved, err = svc.GetTask(task.ID)
	if err != nil || saved.GroupID != nil {
		t.Fatalf("clearing group failed: %v", err)
	}
}
