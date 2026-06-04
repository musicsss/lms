package file

import (
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"strconv"

	"github.com/google/uuid"
	"github.com/lms/server/internal/model"
	"github.com/lms/server/internal/repository"
	"github.com/lms/server/internal/runtimecfg"
	"github.com/lms/server/internal/storage"
	"gorm.io/gorm"
)

type Service struct {
	fileRepo  *repository.FileRepo
	shareRepo *repository.ShareRepo
	store     storage.Driver
	rtEngine  *runtimecfg.Engine
}

func NewService(fileRepo *repository.FileRepo, shareRepo *repository.ShareRepo, store storage.Driver, rtEngine *runtimecfg.Engine) *Service {
	return &Service{fileRepo: fileRepo, shareRepo: shareRepo, store: store, rtEngine: rtEngine}
}

func (s *Service) List(userID uint, parentID *uint) ([]model.File, error) {
	return s.fileRepo.FindByParent(userID, parentID)
}

func (s *Service) maxUploadSizeMB() int {
	if s.rtEngine != nil {
		if v := s.rtEngine.GetSet("FILEUPLD"); v != nil {
			if mb, err := strconv.Atoi(v["MAXSIZE"]); err == nil && mb > 0 {
				return mb
			}
		}
	}
	return 2048
}

func (s *Service) Upload(userID uint, parentID *uint, header *multipart.FileHeader) (*model.File, error) {
	maxMB := s.maxUploadSizeMB()
	maxBytes := int64(maxMB) * 1024 * 1024
	if header.Size > maxBytes {
		return nil, fmt.Errorf("file size exceeds limit of %d MB", maxMB)
	}

	src, err := header.Open()
	if err != nil {
		return nil, err
	}
	defer src.Close()

	ext := filepath.Ext(header.Filename)
	storageKey := fmt.Sprintf("%d/%s%s", userID, uuid.New().String(), ext)

	if err := s.store.Put(nil, storageKey, src, header.Size); err != nil {
		return nil, fmt.Errorf("store file: %w", err)
	}

	mimeType := header.Header.Get("Content-Type")
	isVideo := isVideoMime(mimeType)

	file := &model.File{
		UserID:      userID,
		ParentID:    parentID,
		Name:        header.Filename,
		Size:        header.Size,
		MimeType:    mimeType,
		StorageKey:  storageKey,
		IsVideo:     isVideo,
		VideoStatus: videoStatus(isVideo),
	}

	if err := s.fileRepo.Create(file); err != nil {
		s.store.Delete(nil, storageKey)
		return nil, err
	}

	return file, nil
}

func (s *Service) CreateDir(userID uint, parentID *uint, name string) (*model.File, error) {
	dir := &model.File{
		UserID:   userID,
		ParentID: parentID,
		Name:     name,
		IsDir:    true,
	}
	if err := s.fileRepo.Create(dir); err != nil {
		return nil, err
	}
	return dir, nil
}

func (s *Service) Download(id uint) (*model.File, io.ReadCloser, error) {
	file, err := s.fileRepo.FindByID(id)
	if err != nil {
		return nil, nil, err
	}

	reader, err := s.store.Get(nil, file.StorageKey)
	if err != nil {
		return nil, nil, err
	}

	return file, reader, nil
}

func (s *Service) Delete(id uint) error {
	file, err := s.fileRepo.FindByID(id)
	if err != nil {
		return err
	}

	if file.IsDir {
		return s.deleteDir(file)
	}

	if err := s.store.Delete(nil, file.StorageKey); err != nil {
		return err
	}
	return s.fileRepo.Delete(id)
}

func (s *Service) deleteDir(dir *model.File) error {
	children, err := s.fileRepo.FindChildren(dir.ID)
	if err != nil {
		return err
	}
	for _, child := range children {
		if child.IsDir {
			if err := s.deleteDir(&child); err != nil {
				return err
			}
		} else {
			s.store.Delete(nil, child.StorageKey)
			s.fileRepo.Delete(child.ID)
		}
	}
	return s.fileRepo.Delete(dir.ID)
}

func (s *Service) CreateShare(fileID uint, password string, expireHours int) (*model.FileShare, error) {
	_, err := s.fileRepo.FindByID(fileID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("file not found")
		}
		return nil, err
	}

	share := &model.FileShare{
		FileID:   fileID,
		Token:    uuid.New().String(),
		Password: password,
	}

	if err := s.shareRepo.Create(share); err != nil {
		return nil, err
	}

	return share, nil
}

func (s *Service) GetShare(token string) (*model.FileShare, error) {
	return s.shareRepo.FindByToken(token)
}

func isVideoMime(mime string) bool {
	videoPrefixes := []string{"video/", "application/vnd.apple.mpegurl"}
	for _, p := range videoPrefixes {
		if len(mime) >= len(p) && mime[:len(p)] == p {
			return true
		}
	}
	return false
}

func videoStatus(isVideo bool) string {
	if isVideo {
		return "pending"
	}
	return "none"
}
