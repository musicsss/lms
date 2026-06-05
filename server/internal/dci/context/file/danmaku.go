package file

import (
	"errors"
	"github.com/lms/server/internal/dci/data"
	"github.com/lms/server/internal/model"

	"gorm.io/gorm"
)

// 鈹€鈹€ SendDanmakuContext 鈹€鈹€

type SendDanmakuContext struct {
	db           *gorm.DB
	danmakuRepo  data.DanmakuRepo
	fileRepo     data.FileRepo

	VideoID uint
	UserID  uint

	Content  string
	TimeSec  float64
	Color    string
	FontSize int
	DmType   int

	result *model.Danmaku
}

func NewSendDanmakuContext(db *gorm.DB, danmakuRepo data.DanmakuRepo, fileRepo data.FileRepo, videoID, userID uint, content string, timeSec float64, color string, fontSize int, dmType int) *SendDanmakuContext {
	return &SendDanmakuContext{
		db:          db,
		danmakuRepo: danmakuRepo,
		fileRepo:    fileRepo,
		VideoID:     videoID,
		UserID:      userID,
		Content:     content,
		TimeSec:     timeSec,
		Color:       color,
		FontSize:    fontSize,
		DmType:      dmType,
	}
}

func (c *SendDanmakuContext) Execute() (*model.Danmaku, error) {
	// Validate video exists
	_, err := c.fileRepo.FindByID(c.db, c.VideoID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("video not found")
		}
		return nil, err
	}

	// Default color
	if c.Color == "" {
		c.Color = "#ffffff"
	}

	// Default font size
	if c.FontSize <= 0 {
		c.FontSize = 25
	}

	// Default type
	if c.DmType <= 0 {
		c.DmType = 1
	}

	dm := &model.Danmaku{
		VideoID:  c.VideoID,
		UserID:   c.UserID,
		Content:  c.Content,
		TimeSec:  c.TimeSec,
		Color:    c.Color,
		FontSize: c.FontSize,
		Type:     c.DmType,
		Status:   "approved",
	}
	if err := c.danmakuRepo.CreateDanmaku(c.db, dm); err != nil {
		return nil, err
	}
	c.result = dm
	return dm, nil
}

// 鈹€鈹€ GetDanmakuContext 鈹€鈹€

type GetDanmakuContext struct {
	db          *gorm.DB
	danmakuRepo data.DanmakuRepo

	VideoID uint

	result []model.Danmaku
}

func NewGetDanmakuContext(db *gorm.DB, danmakuRepo data.DanmakuRepo, videoID uint) *GetDanmakuContext {
	return &GetDanmakuContext{db: db, danmakuRepo: danmakuRepo, VideoID: videoID}
}

func (c *GetDanmakuContext) Execute() ([]model.Danmaku, error) {
	danmaku, err := c.danmakuRepo.FindDanmakuByVideo(c.db, c.VideoID, "approved")
	if err != nil {
		return nil, err
	}
	c.result = danmaku
	return danmaku, nil
}

// 鈹€鈹€ ListDanmakuAdminContext 鈹€鈹€

type ListDanmakuAdminContext struct {
	db          *gorm.DB
	danmakuRepo data.DanmakuRepo

	Page     int
	PageSize int

	result []model.Danmaku
	total  int64
}

func NewListDanmakuAdminContext(db *gorm.DB, danmakuRepo data.DanmakuRepo, page, pageSize int) *ListDanmakuAdminContext {
	if page < 1 {
		page = 1
	}
	return &ListDanmakuAdminContext{db: db, danmakuRepo: danmakuRepo, Page: page, PageSize: pageSize}
}

func (c *ListDanmakuAdminContext) Execute() ([]model.Danmaku, int64, error) {
	danmaku, total, err := c.danmakuRepo.ListDanmakuAdmin(c.db, c.Page, c.PageSize)
	if err != nil {
		return nil, 0, err
	}
	c.result = danmaku
	c.total = total
	return danmaku, total, nil
}

// 鈹€鈹€ UpdateDanmakuStatusContext 鈹€鈹€

type UpdateDanmakuStatusContext struct {
	db          *gorm.DB
	danmakuRepo data.DanmakuRepo

	DmID   uint
	Status string
}

func NewUpdateDanmakuStatusContext(db *gorm.DB, danmakuRepo data.DanmakuRepo, dmID uint, status string) *UpdateDanmakuStatusContext {
	return &UpdateDanmakuStatusContext{db: db, danmakuRepo: danmakuRepo, DmID: dmID, Status: status}
}

func (c *UpdateDanmakuStatusContext) Execute() error {
	_, err := c.danmakuRepo.FindDanmakuByID(c.db, c.DmID)
	if err != nil {
		return err
	}
	return c.danmakuRepo.UpdateDanmakuStatus(c.db, c.DmID, c.Status)
}

// 鈹€鈹€ DeleteDanmakuContext 鈹€鈹€

type DeleteDanmakuContext struct {
	db          *gorm.DB
	danmakuRepo data.DanmakuRepo

	DmID uint
}

func NewDeleteDanmakuContext(db *gorm.DB, danmakuRepo data.DanmakuRepo, dmID uint) *DeleteDanmakuContext {
	return &DeleteDanmakuContext{db: db, danmakuRepo: danmakuRepo, DmID: dmID}
}

func (c *DeleteDanmakuContext) Execute() error {
	return c.danmakuRepo.DeleteDanmaku(c.db, c.DmID)
}
