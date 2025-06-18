return fmt.Errorf("base URL is required for provider %s", cfg.AI.Provider)
		}
		if cfg.AI.Model == "" {
			return fmt.Errorf("model is required for providerpackage config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Zachacious/presto/pkg/types"
	"gopkg.in/yaml.v3"
)

// Config represents the top-level application configuration structure.
// It includes settings for AI interaction, default processing options, and file filtering.
type Config struct {
	AI       types.APIConfig `yaml:"ai"`       // AI-related configuration, including API key, base URL, and model.
	Defaults DefaultsConfig  `yaml:"defaults"` // Default processing options for file transformations.
	Filters  FiltersConfig   `yaml:"filters