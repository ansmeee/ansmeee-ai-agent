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

// GetByUserAndType returns the model config for a user and model type, or nil if not set.
func (s *ModelConfigStore) GetByUserAndType(userID int64, modelType int8) (*models.ModelConfig, error) {
	var cfg models.ModelConfig
	err := s.db.Where("user_id = ? AND model_type = ?", userID, modelType).First(&cfg).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

// GetByUser returns all model configs for a user.
func (s *ModelConfigStore) GetByUser(userID int64) ([]*models.ModelConfig, error) {
	var cfgs []*models.ModelConfig
	err := s.db.Where("user_id = ?", userID).Find(&cfgs).Error
	if err != nil {
		return nil, err
	}
	return cfgs, nil
}

// Save upserts the model config for a user and model type.
func (s *ModelConfigStore) Save(userID int64, modelType int8, model, baseURL, token string) (*models.ModelConfig, error) {
	cfg := models.ModelConfig{
		UserID:    userID,
		ModelType: modelType,
		Model:     model,
		BaseURL:   baseURL,
		Token:     token,
	}
	err := s.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "model_type"}},
		DoUpdates: clause.AssignmentColumns([]string{"model", "base_url", "token"}),
	}).Create(&cfg).Error
	if err != nil {
		return nil, err
	}
	return s.GetByUserAndType(userID, modelType)
}
