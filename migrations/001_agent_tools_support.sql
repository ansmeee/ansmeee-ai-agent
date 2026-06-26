-- ============================================================
-- Migration: 001_agent_tools_support
-- Description: Add tool calling support columns to ai_agent table
-- ============================================================

-- UP
ALTER TABLE ai_agent
    ADD COLUMN tools JSON DEFAULT NULL COMMENT 'Allowed tool names, e.g. ["calculator","datetime"]' AFTER prompt,
    ADD COLUMN model_config JSON DEFAULT NULL COMMENT 'Per-agent model parameter overrides (model/temperature/max_tokens/top_p)' AFTER tools,
    ADD COLUMN max_iterations TINYINT NOT NULL DEFAULT 5 COMMENT 'Max ReAct loop iterations (1-10)' AFTER model_config,
    ADD COLUMN status TINYINT NOT NULL DEFAULT 1 COMMENT '1=enabled, 2=disabled' AFTER max_iterations;

-- Optional: backfill existing agents with default tools (uncomment if needed)
-- UPDATE ai_agent SET tools = '["calculator","datetime"]' WHERE tools IS NULL;

-- DOWN (rollback)
-- ALTER TABLE ai_agent
--     DROP COLUMN tools,
--     DROP COLUMN model_config,
--     DROP COLUMN max_iterations,
--     DROP COLUMN status;
