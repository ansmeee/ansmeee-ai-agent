package agent

import (
	"fmt"

	"ansmeee-ai-agent/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

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
		"直接回答用户问题。当需要获取实时信息或进行数学计算时使用工具。评估工具结果后再回复。不要做自我介绍。")
	return err
}

// List returns agents for a user.
func (s *AgentStore) List(userID int64) []*models.Agent {
	var agents []*models.Agent
	s.db.Where("user_id = ?", userID).Order("id").Find(&agents)
	return agents
}

// Get returns a single agent by UUID.
func (s *AgentStore) Get(id string) (*models.Agent, error) {
	var a models.Agent
	if err := s.db.Where("uuid = ?", id).First(&a).Error; err != nil {
		return nil, fmt.Errorf("agent %q not found", id)
	}
	return &a, nil
}

// Create adds a new agent for a user.
func (s *AgentStore) Create(userID int64, title, description, prompt string) (*models.Agent, error) {
	a := models.Agent{
		UUID:        genUUID(),
		UserID:      userID,
		Title:       title,
		Description: description,
		Prompt:      prompt,
	}
	if err := s.db.Create(&a).Error; err != nil {
		return nil, fmt.Errorf("insert agent: %w", err)
	}
	return &a, nil
}

// Update modifies an existing agent.
func (s *AgentStore) Update(id, title, description, prompt string) (*models.Agent, error) {
	updates := map[string]interface{}{}
	if title != "" {
		updates["title"] = title
	}
	if description != "" {
		updates["intro"] = description
	}
	if prompt != "" {
		updates["prompt"] = prompt
	}
	if len(updates) == 0 {
		return s.Get(id)
	}
	if err := s.db.Model(&models.Agent{}).Where("uuid = ?", id).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("update agent: %w", err)
	}
	return s.Get(id)
}

// Delete removes an agent by UUID.
func (s *AgentStore) Delete(id string) error {
	result := s.db.Where("uuid = ?", id).Delete(&models.Agent{})
	if result.Error != nil {
		return fmt.Errorf("delete agent: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("agent %q not found", id)
	}
	return nil
}
