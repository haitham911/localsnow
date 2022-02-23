package store

import (
	"github.com/gulfcoastdevops/snow/model"
	"github.com/jinzhu/gorm"
)

// UserStore is data access struct for user
type UserStore struct {
	db *gorm.DB
}

// NewUserStore returns a new UserStore
func NewUserStore(db *gorm.DB) *UserStore {
	return &UserStore{
		db: db,
	}
}

// GetByEmail finds a user from email
func (s *UserStore) GetByEmail(email string) (*model.User, error) {
	var m model.User
	if err := s.db.Where("email = ?", email).First(&m).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

// GetByID finds a user from id
func (s *UserStore) GetByID(id uint) (*model.User, error) {
	var m model.User
	if err := s.db.Find(&m, id).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

// GetByUsername finds a user from username
func (s *UserStore) GetByUsername(username string) (*model.User, error) {
	var m model.User
	if err := s.db.Where("username = ?", username).First(&m).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

// Create create a user
func (s *UserStore) Create(m *model.User) error {
	return s.db.Create(m).Error
}

// Update update all of user fields
func (s *UserStore) Update(m *model.User) error {
	return s.db.Model(m).Update(m).Error
}

// IsFollowing returns whether user A follows user B or not
func (s *UserStore) IsFollowing(a *model.User, b *model.User) (bool, error) {
	if a == nil || b == nil {
		return false, nil
	}

	var count int
	err := s.db.Table("follows").
		Where("from_user_id = ? AND to_user_id = ?", a.ID, b.ID).
		Count(&count).Error
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// Follow create follow relashionship to User B from user A
func (s *UserStore) Follow(a *model.User, b *model.User) error {
	return s.db.Model(a).Association("Follows").Append(b).Error
}

// Unfollow delete follow relashionship to User B from user A
func (s *UserStore) Unfollow(a *model.User, b *model.User) error {
	return s.db.Model(a).Association("Follows").Delete(b).Error
}

// GetFollowingUserIDs returns user ids current user follows
func (s *UserStore) GetFollowingUserIDs(m *model.User) ([]uint, error) {
	rows, err := s.db.Table("follows").
		Select("to_user_id").
		Where("from_user_id = ?", m.ID).
		Rows()
	if err != nil {
		return []uint{}, err
	}
	defer rows.Close()

	var ids []uint
	for rows.Next() {
		var id uint
		rows.Scan(&id)
		ids = append(ids, id)
	}

	return ids, nil
}
