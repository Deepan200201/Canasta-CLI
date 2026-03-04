package orchestrators

import (
	"fmt"
	"strings"
	"testing"

	"github.com/CanastaWiki/Canasta-CLI/internal/config"
)

// mockOrchestrator records calls for testing
type mockOrchestrator struct {
	calls           []string
	execOutput      string
	execErr         error
	copyToErr       error
	startErr        error
	stopErr         error
	checkRunningErr error
}

func (m *mockOrchestrator) CheckDependencies() error                 { return nil }
func (m *mockOrchestrator) WriteStackFiles(installPath string) error { return nil }
func (m *mockOrchestrator) UpdateStackFiles(installPath string, dryRun bool) (bool, error) {
	return false, nil
}

func (m *mockOrchestrator) Start(instance config.Installation) error {
	m.calls = append(m.calls, "Start")
	return m.startErr
}

func (m *mockOrchestrator) Stop(instance config.Installation) error {
	m.calls = append(m.calls, "Stop")
	return m.stopErr
}

func (m *mockOrchestrator) Update(installPath string) (*UpdateReport, error) {
	return nil, nil
}

func (m *mockOrchestrator) Destroy(installPath string) (string, error) {
	return "", nil
}

func (m *mockOrchestrator) ExecWithError(installPath, service, command string) (string, error) {
	m.calls = append(m.calls, fmt.Sprintf("ExecWithError:%s:%s", service, command))
	return m.execOutput, m.execErr
}

func (m *mockOrchestrator) ExecStreaming(installPath, service, command string) error {
	return nil
}

func (m *mockOrchestrator) CheckRunningStatus(instance config.Installation) error {
	m.calls = append(m.calls, "CheckRunningStatus")
	return m.checkRunningErr
}

func (m *mockOrchestrator) CopyFrom(installPath, service, containerPath, hostPath string) error {
	m.calls = append(m.calls, fmt.Sprintf("CopyFrom:%s:%s:%s", service, containerPath, hostPath))
	return nil
}

func (m *mockOrchestrator) CopyTo(installPath, service, hostPath, containerPath string) error {
	m.calls = append(m.calls, fmt.Sprintf("CopyTo:%s:%s:%s", service, hostPath, containerPath))
	return m.copyToErr
}

func (m *mockOrchestrator) RunBackup(installPath, envPath string, volumes map[string]string, args ...string) (string, error) {
	m.calls = append(m.calls, fmt.Sprintf("RunBackup:%s", strings.Join(args, " ")))
	return m.execOutput, m.execErr
}

func (m *mockOrchestrator) RestoreFromBackupVolume(installPath string, dirs map[string]string) error {
	m.calls = append(m.calls, "RestoreFromBackupVolume")
	return nil
}

func (m *mockOrchestrator) InitConfig(installPath string) error {
	m.calls = append(m.calls, "InitConfig")
	return nil
}

func (m *mockOrchestrator) UpdateConfig(installPath string) error {
	m.calls = append(m.calls, "UpdateConfig")
	return nil
}

func (m *mockOrchestrator) MigrateConfig(installPath string, dryRun bool) (bool, error) {
	m.calls = append(m.calls, "MigrateConfig")
	return false, nil
}

func (m *mockOrchestrator) ListServices(instance config.Installation) ([]string, error) {
	return nil, nil
}
func (m *mockOrchestrator) ExecInteractive(instance config.Installation, service string, command []string) error {
	return nil
}
func (m *mockOrchestrator) Name() string            { return "Mock" }
func (m *mockOrchestrator) SupportsDevMode() bool   { return true }
func (m *mockOrchestrator) SupportsImagePull() bool { return true }

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		wantErr bool
	}{
		{"compose", "compose", false},
		{"docker-compose alias", "docker-compose", false},
		{"kubernetes", "kubernetes", false},
		{"k8s alias", "k8s", false},
		{"unknown", "unknown-orch", true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orch, err := New(tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("New(%q) error = %v, wantErr %v", tt.id, err, tt.wantErr)
			}
			if !tt.wantErr && orch == nil {
				t.Error("expected non-nil orchestrator")
			}
		})
	}
}

func TestExec(t *testing.T) {
	mock := &mockOrchestrator{execOutput: "success output"}
	output, err := Exec(mock, "/tmp/test", "web", "php test.php")
	if err != nil {
		t.Fatalf("Exec() error = %v", err)
	}
	if output != "success output" {
		t.Errorf("Exec() output = %q, want %q", output, "success output")
	}
}

func TestExecError(t *testing.T) {
	mock := &mockOrchestrator{
		execOutput: "error details",
		execErr:    fmt.Errorf("command failed"),
	}
	_, err := Exec(mock, "/tmp/test", "web", "php test.php")
	if err == nil {
		t.Fatal("expected error from Exec")
	}
	if !strings.Contains(err.Error(), "error details") {
		t.Errorf("error should contain output, got: %v", err)
	}
}

func TestStopAndStart(t *testing.T) {
	mock := &mockOrchestrator{}
	instance := config.Installation{Id: "test", Path: "/tmp/test"}

	err := StopAndStart(mock, instance)
	if err != nil {
		t.Fatalf("StopAndStart() error = %v", err)
	}

	if len(mock.calls) != 2 || mock.calls[0] != "Stop" || mock.calls[1] != "Start" {
		t.Errorf("expected [Stop, Start], got %v", mock.calls)
	}
}

func TestStopAndStartStopError(t *testing.T) {
	mock := &mockOrchestrator{stopErr: fmt.Errorf("stop failed")}
	instance := config.Installation{Id: "test", Path: "/tmp/test"}

	err := StopAndStart(mock, instance)
	if err == nil {
		t.Fatal("expected error when Stop fails")
	}

	// Start should not be called
	for _, call := range mock.calls {
		if call == "Start" {
			t.Error("Start should not be called when Stop fails")
		}
	}
}

func TestImportDatabase(t *testing.T) {
	mock := &mockOrchestrator{}
	instance := config.Installation{Id: "test", Path: "/tmp/test"}

	err := ImportDatabase(mock, "mywiki", "/path/to/dump.sql", "secret", instance)
	if err != nil {
		t.Fatalf("ImportDatabase() error = %v", err)
	}

	// Verify CopyTo was called
	hasCopyTo := false
	hasCreateDB := false
	hasImport := false
	for _, call := range mock.calls {
		if strings.HasPrefix(call, "CopyTo:") {
			hasCopyTo = true
		}
		if strings.Contains(call, "CREATE DATABASE") {
			hasCreateDB = true
		}
		if strings.Contains(call, "< /tmp/mywiki.sql") {
			hasImport = true
		}
	}

	if !hasCopyTo {
		t.Error("expected CopyTo call")
	}
	if !hasCreateDB {
		t.Error("expected CREATE DATABASE call")
	}
	if !hasImport {
		t.Error("expected import (mysql <) call")
	}
}

func TestImportDatabaseCompressed(t *testing.T) {
	mock := &mockOrchestrator{}
	instance := config.Installation{Id: "test", Path: "/tmp/test"}

	err := ImportDatabase(mock, "mywiki", "/path/to/dump.sql.gz", "secret", instance)
	if err != nil {
		t.Fatalf("ImportDatabase() error = %v", err)
	}

	// Verify gunzip was called
	hasGunzip := false
	for _, call := range mock.calls {
		if strings.Contains(call, "gunzip") {
			hasGunzip = true
		}
	}
	if !hasGunzip {
		t.Error("expected gunzip call for .sql.gz file")
	}
}

func TestImportDatabaseCopyError(t *testing.T) {
	mock := &mockOrchestrator{copyToErr: fmt.Errorf("copy failed")}
	instance := config.Installation{Id: "test", Path: "/tmp/test"}

	err := ImportDatabase(mock, "mywiki", "/path/to/dump.sql", "secret", instance)
	if err == nil {
		t.Fatal("expected error when CopyTo fails")
	}
}

func TestImportDatabaseDefaultPassword(t *testing.T) {
	mock := &mockOrchestrator{}
	instance := config.Installation{Id: "test", Path: "/tmp/test"}

	err := ImportDatabase(mock, "mywiki", "/path/to/dump.sql", "", instance)
	if err != nil {
		t.Fatalf("ImportDatabase() error = %v", err)
	}

	// Verify default password "mediawiki" is used
	hasDefaultPw := false
	for _, call := range mock.calls {
		if strings.Contains(call, "mediawiki") {
			hasDefaultPw = true
		}
	}
	if !hasDefaultPw {
		t.Error("expected default password 'mediawiki' to be used")
	}
}

func TestRunBackupRecordsCalls(t *testing.T) {
	mock := &mockOrchestrator{execOutput: "snapshot saved"}
	output, err := mock.RunBackup("/tmp/test", "/tmp/test/.env", nil, "-r", "s3:bucket", "snapshots")
	if err != nil {
		t.Fatalf("RunBackup() error = %v", err)
	}
	if output != "snapshot saved" {
		t.Errorf("RunBackup() output = %q, want %q", output, "snapshot saved")
	}
	if len(mock.calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(mock.calls))
	}
	expected := "RunBackup:-r s3:bucket snapshots"
	if mock.calls[0] != expected {
		t.Errorf("call = %q, want %q", mock.calls[0], expected)
	}
}

func TestIsLocalRepo(t *testing.T) {
	tests := []struct {
		name    string
		repoURL string
		want    bool
	}{
		{"local absolute path", "/path/to/repo", true},
		{"local root path", "/backups", true},
		{"s3 backend", "s3:bucket/path", false},
		{"sftp backend", "sftp:user@host:/path", false},
		{"rest backend", "rest:http://host:8000", false},
		{"gs backend", "gs:bucket/path", false},
		{"azure backend", "azure:container/path", false},
		{"b2 backend", "b2:bucket/path", false},
		{"rclone backend", "rclone:remote:path", false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isLocalRepo(tt.repoURL)
			if got != tt.want {
				t.Errorf("isLocalRepo(%q) = %v, want %v", tt.repoURL, got, tt.want)
			}
		})
	}
}

func TestRepoFromArgs(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"local path", []string{"-r", "/backups/repo", "init"}, "/backups/repo"},
		{"s3 backend", []string{"-r", "s3:bucket", "snapshots"}, "s3:bucket"},
		{"no -r flag", []string{"init"}, ""},
		{"-r at end", []string{"init", "-r"}, ""},
		{"empty args", []string{}, ""},
		{"-r with other flags", []string{"--verbose", "-r", "/tmp/repo", "backup", "/data"}, "/tmp/repo"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := repoFromArgs(tt.args)
			if got != tt.want {
				t.Errorf("repoFromArgs(%v) = %q, want %q", tt.args, got, tt.want)
			}
		})
	}
}

func TestRunBackupError(t *testing.T) {
	mock := &mockOrchestrator{
		execOutput: "error output",
		execErr:    fmt.Errorf("restic failed"),
	}
	_, err := mock.RunBackup("/tmp/test", "/tmp/test/.env", nil, "-r", "s3:bucket", "backup", "/data")
	if err == nil {
		t.Fatal("expected error from RunBackup")
	}
	if len(mock.calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(mock.calls))
	}
}

func TestEnsureRunningAlreadyRunning(t *testing.T) {
	mock := &mockOrchestrator{checkRunningErr: nil}
	instance := config.Installation{Id: "test", Path: "/tmp/test"}

	err := EnsureRunning(mock, instance)
	if err != nil {
		t.Fatalf("EnsureRunning() error = %v", err)
	}

	for _, call := range mock.calls {
		if call == "Start" {
			t.Error("Start should not be called when CheckRunningStatus returns nil (already running)")
		}
	}
}

func TestEnsureRunningNotRunning(t *testing.T) {
	mock := &mockOrchestrator{checkRunningErr: fmt.Errorf("not running")}
	instance := config.Installation{Id: "test", Path: "/tmp/test"}

	err := EnsureRunning(mock, instance)
	if err != nil {
		t.Fatalf("EnsureRunning() error = %v", err)
	}

	hasStartCall := false
	for _, call := range mock.calls {
		if call == "Start" {
			hasStartCall = true
		}
	}
	if !hasStartCall {
		t.Error("Start should be called when CheckRunningStatus returns an error (not running)")
	}
}
