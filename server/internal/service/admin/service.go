package admin

import (
	"errors"

	"github.com/lms/server/internal/model"
	"github.com/lms/server/internal/repository"
	"github.com/lms/server/internal/storage"
)

type Service struct {
	userRepo  *repository.UserRepo
	fileRepo  *repository.FileRepo
	forumRepo *repository.ForumRepo
	store     storage.Driver
}

func NewService(userRepo *repository.UserRepo, fileRepo *repository.FileRepo, forumRepo *repository.ForumRepo, store storage.Driver) *Service {
	return &Service{userRepo: userRepo, fileRepo: fileRepo, forumRepo: forumRepo, store: store}
}

type Stats struct {
	UserCount  int64 `json:"user_count"`
	FileCount  int64 `json:"file_count"`
	FileSize   int64 `json:"file_size"`
	PostCount  int64 `json:"post_count"`
	BoardCount int64 `json:"board_count"`
}

func (s *Service) GetStats() (*Stats, error) {
	userCount, err := s.userRepo.Count()
	if err != nil {
		return nil, err
	}
	fileCount, err := s.fileRepo.CountAll()
	if err != nil {
		return nil, err
	}
	fileSize, err := s.fileRepo.SumSize()
	if err != nil {
		return nil, err
	}
	postCount, err := s.forumRepo.CountPosts()
	if err != nil {
		return nil, err
	}
	boards, _ := s.forumRepo.ListBoards()
	return &Stats{
		UserCount:  userCount,
		FileCount:  fileCount,
		FileSize:   fileSize,
		PostCount:  postCount,
		BoardCount: int64(len(boards)),
	}, nil
}

func (s *Service) ListUsers(page, pageSize int, search string) ([]model.User, int64, error) {
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * pageSize
	return s.userRepo.List(offset, pageSize, search)
}

func (s *Service) UpdateUserRole(id uint, role string) error {
	if role != "admin" && role != "user" {
		return errors.New("invalid role")
	}
	return s.userRepo.UpdateRole(id, role)
}

func (s *Service) DeleteUser(id uint) error {
	user, err := s.userRepo.FindByID(id)
	if err != nil {
		return err
	}
	if user.Role == "admin" {
		adminCount, err := s.userRepo.CountByRole("admin")
		if err != nil {
			return err
		}
		if adminCount <= 1 {
			return errors.New("cannot delete the last admin")
		}
	}
	return s.userRepo.Delete(id)
}

func (s *Service) ListFiles(page, pageSize int) ([]model.File, int64, error) {
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * pageSize
	return s.fileRepo.ListAll(offset, pageSize)
}

func (s *Service) DeleteFile(id uint) error {
	file, err := s.fileRepo.FindByID(id)
	if err != nil {
		return err
	}
	return s.deleteRecursive(file)
}

func (s *Service) deleteRecursive(f *model.File) error {
	children, err := s.fileRepo.FindChildren(f.ID)
	if err != nil {
		return err
	}
	for _, child := range children {
		if err := s.deleteRecursive(&child); err != nil {
			return err
		}
	}
	if !f.IsDir {
		s.store.Delete(nil, f.StorageKey)
	}
	return s.fileRepo.Delete(f.ID)
}

func (s *Service) ListBoards() ([]model.Board, error) {
	return s.forumRepo.ListBoards()
}

func (s *Service) CreateBoard(name, slug, description string, sortOrder int) (*model.Board, error) {
	board := &model.Board{
		Name:        name,
		Slug:        slug,
		Description: description,
		SortOrder:   sortOrder,
	}
	if err := s.forumRepo.CreateBoard(board); err != nil {
		return nil, err
	}
	return board, nil
}

func (s *Service) UpdateBoard(id uint, name, slug, description string, sortOrder int) (*model.Board, error) {
	board, err := s.forumRepo.FindBoardByID(id)
	if err != nil {
		return nil, err
	}
	if name != "" {
		board.Name = name
	}
	if slug != "" {
		board.Slug = slug
	}
	if description != "" {
		board.Description = description
	}
	board.SortOrder = sortOrder
	if err := s.forumRepo.UpdateBoard(board); err != nil {
		return nil, err
	}
	return board, nil
}

func (s *Service) DeleteBoard(id uint) error {
	return s.forumRepo.DeleteBoard(id)
}

func (s *Service) DeletePostAdmin(id uint) error {
	if err := s.forumRepo.DeleteReplies(id); err != nil {
		return err
	}
	return s.forumRepo.DeletePost(id)
}

func (s *Service) ListPosts(boardID uint, page, pageSize int) ([]model.Post, int64, error) {
	if page < 1 {
		page = 1
	}
	return s.forumRepo.ListPosts(boardID, page, pageSize)
}
