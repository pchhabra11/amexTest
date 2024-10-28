package main

import (
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

// JSON structures remain unchanged
type Response struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
	Data    Data   `json:"data"`
}

type Data struct {
	Containers []Container `json:"containers"`
}

type Container struct {
	ParentEntityID string  `json:"parent_entity_id"`
	ContainerName  string  `json:"container_name"`
	Graphs         []Graph `json:"graphs"`
}

type Graph struct {
	GraphName     string      `json:"graph_name"`
	GraphMetadata []GraphMeta `json:"graph_metadata"`
}

type GraphMeta struct {
	LegendName     string         `json:"legend_name"`
	EntityID       string         `json:"entity_id"`
	MetricID       string         `json:"metric_id"`
	MetadataLayout MetadataLayout `json:"metadata_layout"`
}

type MetadataLayout struct {
	Containers []Container `json:"containers"`
}

// YAML structures
type Config struct {
	Source Source `yaml:"source"`
}

type Source struct {
	DefaultConfig DefaultConfig `yaml:"defaultConfig"`
	Entity        Entity        `yaml:"entity"`
}

type DefaultConfig struct {
	EmailConfigName            string   `yaml:"emailConfigName"`
	SlackConfigName            string   `yaml:"slackConfigName"`
	IncidentSevTwoConfigName   string   `yaml:"incidentSevTwoConfigName"`
	IncidentSevThreeConfigName string   `yaml:"incidentSevThreeConfigName"`
	IncidentSevFourConfigName  string   `yaml:"incidentSevFourConfigName"`
	Incident                   Incident `yaml:"incident"`
}

type Incident struct {
	Severity string `yaml:"severity"`
	Enabled  bool   `yaml:"enabled"`
}

type Entity struct {
	Name             string            `yaml:"name"`
	ID               string            `yaml:"id"`
	Ignore           EntityIDs         `yaml:"ignore"`
	Whitelist        EntityIDs         `yaml:"whitelist"`
	MetricThresholds []MetricThreshold `yaml:"metricThresholds"`
}

type EntityIDs struct {
	EntityIds []string `yaml:"entityIds"`
}

type MetricThreshold struct {
	EntityID       string   `yaml:"entityId"`
	MetricID       string   `yaml:"metricId"`
	ParentEntityID string   `yaml:"parentEntityId"`
	ContainerName  string   `yaml:"containerName"`
	GraphName      string   `yaml:"graphName"`
	LegendName     string   `yaml:"legendName"`
	Min            *float64 `yaml:"min,omitempty"`
	Max            *float64 `yaml:"max,omitempty"`
	Incident       string   `yaml:"incident,omitempty"`
}

// Function to create directory structure and generate YAML files
func createStructureAndYaml(basePath string, containers []Container, yamlConfig Config) error {
	for _, container := range containers {
		sanitizedName := sanitizeFolderName(container.ContainerName)
		currentPath := filepath.Join(basePath, sanitizedName)

		if err := os.MkdirAll(currentPath, 0755); err != nil {
			return fmt.Errorf("error creating directory %s: %v", currentPath, err)
		}

		// Create YAML file for this container
		containerYaml := createContainerYaml(yamlConfig, container)
		yamlData, err := yaml.Marshal(containerYaml)
		if err != nil {
			return fmt.Errorf("error marshaling YAML for %s: %v", container.ContainerName, err)
		}

		yamlPath := filepath.Join(currentPath, "config.yaml")
		if err := ioutil.WriteFile(yamlPath, yamlData, 0644); err != nil {
			return fmt.Errorf("error writing YAML file %s: %v", yamlPath, err)
		}

		// Process nested containers
		for _, graph := range container.Graphs {
			for _, meta := range graph.GraphMetadata {
				if meta.MetadataLayout.Containers != nil {
					if err := createStructureAndYaml(currentPath, meta.MetadataLayout.Containers, yamlConfig); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

// Creates a YAML configuration tailored to a specific container
func createContainerYaml(config Config, container Container) Config {
	newConfig := Config{
		Source: Source{
			DefaultConfig: config.Source.DefaultConfig,
			Entity: Entity{
				Name:      config.Source.Entity.Name,
				ID:        config.Source.Entity.ID,
				Ignore:    config.Source.Entity.Ignore,
				Whitelist: config.Source.Entity.Whitelist,
			},
		},
	}

	// Deduplicate based solely on entityId and metricId combinations
	uniqueThresholds := make(map[string]MetricThreshold)

	for _, graph := range container.Graphs {
		for _, meta := range graph.GraphMetadata {
			for _, threshold := range config.Source.Entity.MetricThresholds {
				if threshold.EntityID == meta.EntityID && threshold.MetricID == meta.MetricID {
					key := threshold.EntityID + "-" + threshold.MetricID

					// Only add if this unique combination of entityId and metricId has not been added before
					if _, exists := uniqueThresholds[key]; !exists {
						uniqueThresholds[key] = threshold
					}
				}
			}
		}
	}

	// Append the unique thresholds to newConfig
	for _, threshold := range uniqueThresholds {
		newConfig.Source.Entity.MetricThresholds = append(newConfig.Source.Entity.MetricThresholds, threshold)
	}

	return newConfig
}

// Sanitizes folder names to ensure compatibility with file system restrictions
func sanitizeFolderName(name string) string {
	invalid := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	result := name
	for _, char := range invalid {
		result = strings.ReplaceAll(result, char, "_")
	}
	return result
}

func main() {
	// Read JSON file
	jsonFile, err := os.ReadFile("test-1.json")
	if err != nil {
		fmt.Printf("Error reading JSON file: %v\n", err)
		return
	}

	// Read YAML file
	yamlFile, err := os.ReadFile("test-2.yaml")
	if err != nil {
		fmt.Printf("Error reading YAML file: %v\n", err)
		return
	}

	// Parse JSON
	var response Response
	if err := json.Unmarshal(jsonFile, &response); err != nil {
		fmt.Printf("Error parsing JSON: %v\n", err)
		return
	}

	// Parse YAML using the updated Config struct
	var yamlConfig Config
	if err := yaml.Unmarshal(yamlFile, &yamlConfig); err != nil {
		fmt.Printf("Error parsing YAML: %v\n", err)
		return
	}

	// Create base directory
	basePath := "monitoring_structure"
	if err := os.MkdirAll(basePath, 0755); err != nil {
		fmt.Printf("Error creating base directory: %v\n", err)
		return
	}

	// Create folder structure and YAML files
	if err := createStructureAndYaml(basePath, response.Data.Containers, yamlConfig); err != nil {
		fmt.Printf("Error creating structure: %v\n", err)
		return
	}

	fmt.Println("Folder structure and YAML files created successfully!")
}
