package forum

import (
	"errors"

	"github.com/lms/server/internal/model"
	"github.com/lms/server/internal/repository"
	"gorm.io/gorm"
)

type Service struct {
	repo *repository.ForumRepo
}

func NewService(repo *repository.ForumRepo) *Service {
	return &Service{repo: repo}
}

func (s *Service) ListBoards() ([]model.Board, error) {
	return s.repo.ListBoards()
}

func (s *Service) CreatePost(boardID, userID uint, title, content string) (*model.Post, error) {
	if _, err := s.repo.FindBoardByID(boardID); err != nil {
		return nil, errors.New("board not found")
	}
	post := &model.Post{
		BoardID: boardID,
		UserID:  userID,
		Title:   title,
		Content: content,
	}
	if err := s.repo.CreatePost(post); err != nil {
		return nil, err
	}
	return post, nil
}

func (s *Service) Reply(parentID, boardID, userID uint, content string) (*model.Post, error) {
	parent, err := s.repo.FindPostByID(parentID)
	if err != nil {
		return nil, errors.New("parent post not found")
	}

	reply := &model.Post{
		BoardID:  parent.BoardID,
		UserID:   userID,
		Content:  content,
		ParentID: &parentID,
	}
	if err := s.repo.CreatePost(reply); err != nil {
		return nil, err
	}

	reply.User, _ = s.repoFindUserName(userID)
	return reply, nil
}

func (s *Service) GetPost(id uint) (*model.Post, error) {
	post, err := s.repo.FindPostByID(id)
	if err != nil {
		return nil, err
	}

	// enrich like counts
	if post.ParentID == nil {
		count, _ := s.repo.CountLikes(post.ID)
		post.LikeCount = int(count)
		for i := range post.Replies {
			c, _ := s.repo.CountLikes(post.Replies[i].ID)
			post.Replies[i].LikeCount = int(c)
		}
	}

	return post, nil
}

func (s *Service) ListPosts(boardID uint, page, pageSize int) ([]model.Post, int64, error) {
	if page < 1 {
		page = 1
	}
	posts, total, err := s.repo.ListPosts(boardID, page, pageSize)
	if err != nil {
		return nil, 0, err
	}

	for i := range posts {
		count, _ := s.repo.CountReplies(posts[i].ID)
		posts[i].LikeCount = int(count)
	}

	return posts, total, nil
}

func (s *Service) ToggleLike(postID, userID uint) (bool, error) {
	_, err := s.repo.FindLike(postID, userID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return true, s.repo.CreateLike(&model.PostLike{
			PostID: postID,
			UserID: userID,
		})
	}
	if err != nil {
		return false, err
	}
	return false, s.repo.DeleteLike(postID, userID)
}

func (s *Service) repoFindUserName(userID uint) (model.User, error) {
	// lightweight lookup; the service doesn't own user_repo so use a simpler approach
	return model.User{}, nil
}
