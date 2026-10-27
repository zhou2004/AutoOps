package service

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"dodevops-api/api/k8s/model"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCephRejectsUnsafeMaintenanceRequests(t *testing.T) {
	cases := []model.CephTaskRequest{
		{Operation: "shell", Params: map[string]string{"command": "id"}},
		{Operation: "rbd-snapshot", Params: map[string]string{"pool": "rbd", "image": "--help"}},
		{Operation: "rbd-snapshot", Params: map[string]string{"pool": "rbd", "image": "img;id"}},
		{Operation: "rbd-snapshot", Params: map[string]string{"pool": "rbd", "image": "img", "command": "id"}},
		{Operation: "backup-restore", Params: map[string]string{"repository": "backup", "backupId": "../escape", "targetPool": "rbd", "targetImage": "new"}},
		{Operation: "backup-delete", Params: map[string]string{"repository": "backup", "backupId": "full"}},
		{Operation: "rbd-migration-prepare", Confirmation: "rbd-migration-prepare", Params: map[string]string{"pool": "rbd", "image": "csi-vol-abc", "targetPool": "other", "targetImage": "new"}},
		{Operation: "rbd-copy", Params: map[string]string{"pool": "rbd", "image": "same", "snapshot": "snap", "targetPool": "rbd", "targetImage": "same"}},
		{Operation: "cephfs-delete", Confirmation: "cephfs-delete", Params: map[string]string{"filesystem": "fs", "group": "csi", "subvolume": "vol"}},
		{Operation: "cephfs-delete", Confirmation: "cephfs-delete", Params: map[string]string{"filesystem": "fs", "subvolume": "csi-vol-abc"}},
		{Operation: "cephfs-resize", Params: map[string]string{"filesystem": "fs", "subvolume": "vol", "sizeGiB": "-1"}},
		{Operation: "osd-flag", Confirmation: "osd-flag", Params: map[string]string{"flag": "pause", "action": "set"}},
		{Operation: "pg-repair", Confirmation: "pg-repair", Params: map[string]string{"pg": "1.a;id"}},
		{Operation: "rbd-snapshot", TimeoutSeconds: 1, Params: map[string]string{"pool": "rbd", "image": "img"}},
	}
	for _, req := range cases {
		t.Run(req.Operation+"/"+req.Params["image"]+req.Params["subvolume"], func(t *testing.T) {
			if _, err := validateCephTask(&req); err == nil {
				t.Fatalf("unsafe request accepted: %#v", req)
			}
		})
	}
}

func TestCephJobExecutionAndCredentialBoundaries(t *testing.T) {
	req := model.CephTaskRequest{Operation: "backup-full", Params: map[string]string{"pool": "rbd", "image": "data", "repository": "daily"}}
	job, err := newCephJob("ceph:test", "operator", req, "backup-pvc")
	if err != nil {
		t.Fatal(err)
	}
	if *job.Spec.BackoffLimit != 0 || job.Spec.Template.Spec.RestartPolicy != corev1.RestartPolicyNever {
		t.Fatal("storage mutations must not retry automatically")
	}
	if *job.Spec.ActiveDeadlineSeconds != 21600 || *job.Spec.Parallelism != 1 || *job.Spec.Completions != 1 {
		t.Fatal("expected bounded, single-worker job")
	}
	if *job.Spec.Template.Spec.AutomountServiceAccountToken {
		t.Fatal("maintenance worker must not receive a Kubernetes token")
	}
	container := job.Spec.Template.Spec.Containers[0]
	if container.Command[0] != "python3" || container.Command[2] != "-c" || len(container.Command) != 6 {
		t.Fatalf("invalid worker invocation: %v", container.Command[:3])
	}
	var body model.CephTaskRequest
	if err := json.Unmarshal([]byte(container.Command[5]), &body); err != nil || body.Operation != req.Operation {
		t.Fatalf("worker did not receive the task as one JSON argument: %v", err)
	}
	for _, env := range container.Env {
		if env.Name != "TASK_ID" || env.ValueFrom == nil {
			t.Fatal("unexpected secret or literal task id in job environment")
		}
	}
	for _, mount := range container.VolumeMounts {
		if mount.Name == "ceph-secret" {
			t.Fatal("raw mon secret should only be exposed to the init container")
		}
	}
	if job.Spec.Template.Spec.Volumes[1].Secret.SecretName != "rook-ceph-mon" {
		t.Fatal("expected existing secret reference")
	}
	if job.Spec.Template.Spec.Volumes[3].PersistentVolumeClaim.ClaimName != "backup-pvc" {
		t.Fatal("repository PVC was not mounted")
	}

	// Exercise the exact Python -c argv contract without Ceph binaries or secrets.
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("python3 is not installed")
	}
	username := filepath.Join(t.TempDir(), "username")
	if err = os.WriteFile(username, []byte("client.test"), 0600); err != nil {
		t.Fatal(err)
	}
	start := strings.ReplaceAll(cephWorkerStart, "/etc/ceph/username", username)
	probe := "import json,os,sys; assert __name__ == '__main__'; assert os.environ['CEPH_ARGS'] == '--name client.test'; print(json.loads(sys.argv[1])['operation'])"
	output, err := exec.Command(python, "-u", "-c", start, probe, container.Command[5]).CombinedOutput()
	if err != nil || strings.TrimSpace(string(output)) != "backup-full" {
		t.Fatalf("worker launcher failed: %v %s", err, output)
	}
}

func TestCephJobCancellationStates(t *testing.T) {
	cases := []struct {
		name      string
		active    int32
		cancelled bool
		condition batchv1.JobConditionType
		want      string
	}{
		{"pending", 0, false, "", "Pending"},
		{"running", 1, false, "", "Running"},
		{"terminating", 1, true, batchv1.JobSuspended, "Cancelling"},
		{"cancelled", 0, true, batchv1.JobSuspended, "Cancelled"},
		{"success race", 0, true, batchv1.JobComplete, "Succeeded"},
		{"failure race", 0, true, batchv1.JobFailed, "Failed"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			job := &batchv1.Job{ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{}}, Status: batchv1.JobStatus{Active: tc.active}}
			if tc.cancelled {
				job.Annotations["autoops.io/cancelled"] = "true"
			}
			if tc.condition != "" {
				job.Status.Conditions = []batchv1.JobCondition{{Type: tc.condition, Status: corev1.ConditionTrue}}
			}
			if got := cephJobView(job)["state"]; got != tc.want {
				t.Fatalf("got %v, want %s", got, tc.want)
			}
		})
	}
}

func TestCephDiagnosticsRedactNestedCredentials(t *testing.T) {
	value := map[string]any{"uid": "user", "keys": []any{"secret"}, "usage": []any{map[string]any{"access_key": "hidden", "secret_key": "hidden", "bytes": 123}}}
	encoded, err := json.Marshal(redactCephSecrets(value))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "hidden") || strings.Contains(string(encoded), "secret") || !strings.Contains(string(encoded), "123") {
		t.Fatalf("credential redaction failed: %s", encoded)
	}
	if _, err := cephInspectCommand("rgw-user", map[string]string{"uid": "user;id"}); err == nil {
		t.Fatal("unsafe diagnostic parameter accepted")
	}
	if _, err := cephInspectCommand("arbitrary-command", nil); err == nil {
		t.Fatal("unlisted diagnostic accepted")
	}
}
