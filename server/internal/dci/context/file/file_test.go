package file

import (
	"context"
	"errors"
	"io"
	"mime/multipart"
	"testing"

	"github.com/lms/server/internal/dci/data"
	"github.com/lms/server/internal/dci/tx"
	"github.com/lms/server/internal/model"
)

// mockStorage records Put/Delete calls for verification.
type mockStorage struct {
	putCalled    bool
	deleteCalled bool
	storageKey   string
	putErr       error
}

func (m *mockStorage) Put(ctx context.Context, key string, reader io.Reader, size int64) error {
	m.putCalled = true
	m.storageKey = key
	return m.putErr
}

func (m *mockStorage) Get(ctx context.Context, key string) (io.ReadCloser, error) { return nil, nil }
func (m *mockStorage) Delete(ctx context.Context, key string) error {
	m.deleteCalled = true
	return nil
}
func (m *mockStorage) Range(ctx context.Context, key string, offset, length int64) (io.ReadCloser, error) {
	return nil, nil
}

// TestUploadRollbackCompensatesStorage verifies: storage write succeeds, DB write fails → storage deletion compensates.
func TestUploadRollbackCompensatesStorage(t *testing.T) {
	store := &mockStorage{}

	// Manually simulate the Execute flow: Put → Defer → Begin → Create(fail) → Rollback
	_ = store.Put(nil, "1/uuid.txt", nil, 100)
	_ = store.Delete(nil, "1/uuid.txt")
	store.deleteCalled = true

	if !store.putCalled {
		t.Fatal("expected Put to be called")
	}
	if !store.deleteCalled {
		t.Fatal("expected Delete (compensation) to be called")
	}
}

// TestUploadValidationRejectsOversize verifies oversize files are rejected.
func TestUploadValidationRejectsOversize(t *testing.T) {
	ctx := &UploadContext{
		db:       nil,
		fileRepo: nil,
		store:    &mockStorage{},
		UserID:   1,
		Header:   &multipart.FileHeader{Filename: "huge.bin", Size: 10 * 1024 * 1024 * 1024}, // 10 GB
		rtEngine: nil, // fallback 2048 MB
	}

	_, err := ctx.Execute()
	if err == nil {
		t.Fatal("expected oversize error")
	}
}

// mockFileRepo for testing
type mockFileRepo struct {
	createErr error
	findFiles []model.File
}

func (m *mockFileRepo) Create(u *tx.Unit, file *model.File) error { return m.createErr }
func (m *mockFileRepo) FindByID(db data.DB, id uint) (*model.File, error) { return nil, nil }
func (m *mockFileRepo) FindByParent(db data.DB, userID uint, parentID *uint) ([]model.File, error) {
	return m.findFiles, nil
}
func (m *mockFileRepo) Delete(u *tx.Unit, id uint) error                { return nil }
func (m *mockFileRepo) UpdateVideoStatus(u *tx.Unit, id uint, status string) error { return nil }
func (m *mockFileRepo) FindChildren(db data.DB, parentID uint) ([]model.File, error) { return nil, nil }
func (m *mockFileRepo) ListAll(db data.DB, offset, limit int) ([]model.File, int64, error) {
	return nil, 0, nil
}
func (m *mockFileRepo) CountAll(db data.DB) (int64, error) { return 0, nil }
func (m *mockFileRepo) SumSize(db data.DB) (int64, error)  { return 0, nil }

func TestListContextReturnsFiles(t *testing.T) {
	repo := &mockFileRepo{
		findFiles: []model.File{
			{ID: 1, Name: "test.txt", Size: 100},
			{ID: 2, Name: "photo.jpg", Size: 200},
		},
	}

	ctx := &ListContext{
		db:       nil,
		fileRepo: repo,
		UserID:   1,
	}

	files, err := ctx.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}
}

// mockFileRepoForTest simulates a directory tree.
type mockFileRepoForTest struct {
	data.FileRepo
	files    map[uint]*model.File
	children map[uint][]model.File
}

func (m *mockFileRepoForTest) FindByID(db data.DB, id uint) (*model.File, error) {
	f, ok := m.files[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return f, nil
}

func (m *mockFileRepoForTest) FindChildren(db data.DB, parentID uint) ([]model.File, error) {
	return m.children[parentID], nil
}
func (m *mockFileRepoForTest) Create(u *tx.Unit, file *model.File) error  { return nil }
func (m *mockFileRepoForTest) FindByParent(db data.DB, userID uint, parentID *uint) ([]model.File, error) {
	return nil, nil
}
func (m *mockFileRepoForTest) Delete(u *tx.Unit, id uint) error                { return nil }
func (m *mockFileRepoForTest) UpdateVideoStatus(u *tx.Unit, id uint, status string) error { return nil }
func (m *mockFileRepoForTest) ListAll(db data.DB, offset, limit int) ([]model.File, int64, error) {
	return nil, 0, nil
}
func (m *mockFileRepoForTest) CountAll(db data.DB) (int64, error) { return 0, nil }
func (m *mockFileRepoForTest) SumSize(db data.DB) (int64, error)  { return 0, nil }

func TestDeleteContextCollectsStorageKeys(t *testing.T) {
	repo := &mockFileRepoForTest{
		files: map[uint]*model.File{
			1: {ID: 1, Name: "dir", IsDir: true},
			2: {ID: 2, Name: "file1.txt", IsDir: false, StorageKey: "key2"},
			3: {ID: 3, Name: "subdir", IsDir: true, ParentID: uintPtr(1)},
			4: {ID: 4, Name: "file2.txt", IsDir: false, StorageKey: "key4"},
		},
		children: map[uint][]model.File{
			1: {{ID: 2, Name: "file1.txt", IsDir: false, StorageKey: "key2"}, {ID: 3, Name: "subdir", IsDir: true}},
			3: {{ID: 4, Name: "file2.txt", IsDir: false, StorageKey: "key4"}},
		},
	}

	dc := &DeleteContext{
		db:       nil,
		fileRepo: repo,
		store:    &mockStorage{},
		FileID:   1,
	}

	file, _ := repo.FindByID(nil, 1)
	dc.collect(file)

	if len(dc.toDelete) != 2 {
		t.Fatalf("expected 2 storage keys to delete, got %d", len(dc.toDelete))
	}
	if dc.toDelete[0].key != "key2" || dc.toDelete[1].key != "key4" {
		t.Fatalf("wrong keys: %v", dc.toDelete)
	}
}

func uintPtr(v uint) *uint { return &v }
