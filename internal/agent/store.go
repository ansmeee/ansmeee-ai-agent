package agent

import (
	"errors"
	"fmt"

	"ansmeee-ai-agent/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ErrAgentNotFound is returned when an agent does not exist.
var ErrAgentNotFound = errors.New("agent not found")

// AgentStore persists agent configurations in MySQL with master/slave via GORM.
type AgentStore struct {
	db *gorm.DB
}

// NewAgentStoreWithDB creates a new agent store using a shared GORM DB.
func NewAgentStoreWithDB(db *gorm.DB) (*AgentStore, error) {
	s := &AgentStore{db: db}
	return s, nil
}

func genUUID() string {
	return uuid.New().String()
}

// EnsureDefault creates a default agent for the user if none exist.
func (s *AgentStore) EnsureDefault(userID int64) error {
	var count int64
	if err := s.db.Model(&models.Agent{}).Where("user_id = ?", userID).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	_, err := s.Create(userID, "默认助手", "通用 AI 助手",
		"直接回答用户问题。当需要获取实时信息或进行数学计算时使用工具。评估工具结果后再回复。不要做自我介绍。",
		[]string{"calculator", "datetime"}, nil, 0)
	return err
}

// List returns agents for a user.
func (s *AgentStore) List(userID int64) []*models.Agent {
	var agents []*models.Agent
	s.db.Where("user_id = ?", userID).Order("id").Find(&agents)
	return agents
}

// Get returns a single agent by UUID, scoped to a user.
func (s *AgentStore) Get(id string, userID int64) (*models.Agent, error) {
	var a models.Agent
	if err := s.db.Where("uuid = ? AND user_id = ?", id, userID).First(&a).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAgentNotFound
		}
		return nil, fmt.Errorf("query agent: %w", err)
	}
	return &a, nil
}

// Create adds a new agent for a user.
func (s *AgentStore) Create(userID int64, title, description, prompt string,
	tools []string, modelConfig *models.AgentModelConfig, maxIterations int8,
) (*models.Agent, error) {
	a := models.Agent{
		UUID:          genUUID(),
		UserID:        userID,
		Title:         title,
		Description:   description,
		Prompt:        prompt,
		Tools:         models.JSONStringSlice(tools),
		ModelConfig:   modelConfig,
		MaxIterations: maxIterations,
		Status:        models.AgentStatusEnabled,
	}
	if a.MaxIterations == 0 {
		a.MaxIterations = 5
	}
	if a.MaxIterations > 10 {
		a.MaxIterations = 10
	}
	if err := s.db.Create(&a).Error; err != nil {
		return nil, fmt.Errorf("insert agent: %w", err)
	}
	return &a, nil
}

// Update modifies an existing agent using a whitelist of allowed fields.
func (s *AgentStore) Update(id string, userID int64, updates map[string]interface{}) (*models.Agent, error) {
	allowed := map[string]bool{
		"title": true, "description": true, "prompt": true,
		"tools": true, "model_config": true, "max_iterations": true, "status": true,
	}
	filtered := make(map[string]interface{})
	for k, v := range updates {
		if allowed[k] {
			if k == "description" {
				filtered["intro"] = v
			} else {
				filtered[k] = v
			}
		}
	}
	if len(filtered) == 0 {
		return s.Get(id, userID)
	}
	if err := s.db.Model(&models.Agent{}).Where("uuid = ? AND user_id = ?", id, userID).Updates(filtered).Error; err != nil {
		return nil, fmt.Errorf("update agent: %w", err)
	}
	return s.Get(id, userID)
}

// Delete removes an agent by UUID, scoped to a user.
func (s *AgentStore) Delete(id string, userID int64) error {
	result := s.db.Where("uuid = ? AND user_id = ?", id, userID).Delete(&models.Agent{})
	if result.Error != nil {
		return fmt.Errorf("delete agent: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("agent %q not found", id)
	}
	return nil
}
