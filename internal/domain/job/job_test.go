package job

import "testing"

func TestNew(t *testing.T) {
	task, err := New("job-001", "5475458029853092", "Waba Percobaan")
	if err != nil {
		t.Fatalf("New() gagal: %v", err)
	}

	if task.Status() != StatusQueued {
		t.Errorf("status = %q, ingin %q", task.Status(), StatusQueued)
	}

	if task.CreatedAt.IsZero() {
		t.Error("waktu pembuatan belum diisi")
	}
}

func TestNewRejectsEmptyFields(t *testing.T) {
	test := []struct {
		name       string
		id         string
		businessID string
		wabaName   string
	}{
		{
			name:       "ID kosong",
			businessID: "123456",
			wabaName:   "Waba Percobaan",
		},
		{
			name:     "business ID kosong",
			id:       "123456",
			wabaName: "Waba Percobaan",
		},
		{
			name:       "nama hanya spasi",
			id:         "job-001",
			businessID: "123456",
			wabaName:   "   ",
		},
	}

	for _, tt := range test {
		t.Run(tt.name, func(t *testing.T) {
			task, err := New(tt.id, tt.businessID, tt.wabaName)

			if err == nil {
				t.Fatal("ingin error, tetapi mendapat nil")
			}

			if task != nil {
				t.Error("Job seharusnya nil ketika validasi gagal")
			}
		})
	}

}

func TestAwaitManualAction(t *testing.T) {
	task, err := New("job-001", "5475458029853092", "Waba Percobaan")
	if err != nil {
		t.Fatal(err)
	}

	if err := task.AwaitManualAction(); err == nil {
		t.Fatal("job queued harus ditolak")
	}

	if task.Status() != StatusQueued {
		t.Fatal("transisi yang ditolak tidak boleh mengubah status")
	}

	if err := task.Start(); err != nil {
		t.Fatal(err)
	}

	if err := task.AwaitManualAction(); err != nil {
		t.Fatalf("AwaitManualAction() gagal; %v", err)
	}

	if task.Status() != StatusAwaitingManualAction {
		t.Errorf(
			"status = %q, ingin = %q",
			task.Status(),
			StatusAwaitingManualAction,
		)
	}

	if err := task.AwaitManualAction(); err == nil {
		t.Error("job yang sudah menunggu tindakan manual harus ditolak")
	}
}

func TestStart(t *testing.T) {
	task, err := New("job-001", "5475458029853092", "Waba Percobaan")
	if err != nil {
		t.Fatal(err)
	}

	if err := task.Start(); err != nil {
		t.Fatalf("Start() gagal: %v", err)
	}

	if task.Status() != StatusRunning {
		t.Errorf("status = %q, ingin %q", task.Status(), StatusRunning)
	}

	if err := task.Start(); err == nil {
		t.Fatal("memulai Job yang sudah running harus ditolak")
	}
}
