package llm

import (
	"ansmeee-ai-agent/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ModelConfigStore persists user model configurations.
type ModelConfigStore struct {
	db *gorm.DB
}

// NewModelConfigStore creates a new model config store.
func NewModelConfigStore(db *gorm.DB) *ModelConfigStore {
	return &ModelConfigStore{db: db}
}

// GetByUser returns the model config for a user, or nil if not set.
func (s *ModelConfigStore) GetByUser(userID int64) (*models.ModelConfig, error) {
	var cfg models.ModelConfig
	err := s.db.Where("user_id = ?", userID).First(&cfg).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

// Save upserts the model config for a user.
func (s *ModelConfigStore) Save(userID int64, model, baseURL, token string) (*models.ModelConfig, error) {
	cfg := models.ModelConfig{
		UserID:  userID,
		Model:   model,
		BaseURL: baseURL,
		Token:   token,
	}
	err := s.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"model", "base_url", "token"}),
	}).Create(&cfg).Error
	if err != nil {
		return nil, err
	}
	return s.GetByUser(userID)
}
