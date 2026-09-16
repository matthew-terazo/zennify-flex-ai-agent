-- Create prompt_configurations table
CREATE TABLE prompt_configurations (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT NOT NULL UNIQUE,
  system_prompt TEXT,
  metadata JSONB DEFAULT '{}'::jsonb,
  version INT DEFAULT 1,
  is_active BOOLEAN DEFAULT true,
  created_at TIMESTAMPTZ DEFAULT now(),
  updated_at TIMESTAMPTZ DEFAULT now()
);

-- Create knowledge_entries table
CREATE TABLE knowledge_entries (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  config_id UUID NOT NULL REFERENCES prompt_configurations(id) ON DELETE CASCADE,
  category TEXT,
  content TEXT NOT NULL,
  metadata JSONB DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ DEFAULT now(),
  updated_at TIMESTAMPTZ DEFAULT now()
);

-- Create knowledge_versions table for versioning
CREATE TABLE knowledge_versions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  entry_id UUID NOT NULL REFERENCES knowledge_entries(id) ON DELETE CASCADE,
  content_version INT NOT NULL,
  content TEXT NOT NULL,
  changed_at TIMESTAMPTZ DEFAULT now()
);

-- Create indexes
CREATE INDEX idx_prompt_configurations_name ON prompt_configurations(name);
CREATE INDEX idx_prompt_configurations_is_active ON prompt_configurations(is_active);
CREATE INDEX idx_knowledge_entries_config_id ON knowledge_entries(config_id);
CREATE INDEX idx_knowledge_entries_category ON knowledge_entries(config_id, category);
CREATE INDEX idx_knowledge_versions_entry_id ON knowledge_versions(entry_id);
