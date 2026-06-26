package services

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sevenclockseven/zhangyi/internal/models"
	"gorm.io/gorm"
)

// TemplateAccount represents an account in a template file
type TemplateAccount struct {
	Code   string   `json:"code"`
	Name   string   `json:"name"`
	Dir    string   `json:"direction"`
	Level  int      `json:"level"`
	Parent string   `json:"parent"`
	Aux    []string `json:"aux"`
}

// TemplateFile represents a template JSON file
type TemplateFile struct {
	ID                string            `json:"id"`
	Name              string            `json:"name"`
	Version           string            `json:"version"`
	Base              string            `json:"base"`
	Standard          string            `json:"standard"`
	Industry          string            `json:"industry"`
	Taxpayer          string            `json:"taxpayer"`
	Description       string            `json:"description"`
	Accounts          []TemplateAccount `json:"accounts"`
}

// ManifestTemplate represents a template entry in manifest.json
type ManifestTemplate struct {
	ID        string `json:"id"`
	Standard  string `json:"standard"`
	Industry  string `json:"industry"`
	Taxpayer  string `json:"taxpayer"`
	File      string `json:"file"`
}

// Manifest represents the v2 manifest.json
type Manifest struct {
	Version        string                       `json:"version"`
	Standards      map[string]map[string]string  `json:"standards"`
	Industries     map[string]map[string]string  `json:"industries"`
	TaxpayerTypes  map[string]map[string]string  `json:"taxpayer_types"`
	Templates      []ManifestTemplate            `json:"templates"`
}

// LoadTemplate loads a template from the templates directory
func LoadTemplate(dir, id string) (*TemplateFile, error) {
	path := filepath.Join(dir, id+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read template %s: %w", id, err)
	}

	var tpl TemplateFile
	if err := json.Unmarshal(data, &tpl); err != nil {
		return nil, fmt.Errorf("failed to parse template %s: %w", id, err)
	}

	return &tpl, nil
}

// LoadV2Template loads a v2 template by standard, industry, and taxpayer type
func LoadV2Template(dir, standard, industry, taxpayer string) (*TemplateFile, error) {
	// Default to small_business if not specified
	if standard == "" {
		standard = "small_business"
	}
	// Default to general taxpayer if not specified
	if taxpayer == "" {
		taxpayer = "general"
	}

	// Try v2 directory first
	v2Dir := filepath.Join(dir, "v2")
	manifestPath := filepath.Join(v2Dir, "manifest.json")

	if _, err := os.Stat(manifestPath); err == nil {
		// v2 templates exist, find the matching template
		manifestData, err := os.ReadFile(manifestPath)
		if err == nil {
			var manifest Manifest
			if json.Unmarshal(manifestData, &manifest) == nil {
				// Find matching template
				for _, t := range manifest.Templates {
					if t.Standard == standard && t.Industry == industry && t.Taxpayer == taxpayer {
						tplPath := filepath.Join(v2Dir, t.File)
						tplData, err := os.ReadFile(tplPath)
						if err != nil {
							continue
						}
						var tpl TemplateFile
						if json.Unmarshal(tplData, &tpl) == nil {
							return &tpl, nil
						}
					}
				}
			}
		}
	}

	// Fallback to old template system
	return nil, fmt.Errorf("v2 template not found for %s/%s/%s", standard, industry, taxpayer)
}

// LoadAndMergeTemplates loads base template + industry overlays and merges them
func LoadAndMergeTemplates(dir string, industryIDs []string) ([]TemplateAccount, error) {
	// Always load base template first
	base, err := LoadTemplate(dir, "base")
	if err != nil {
		return nil, fmt.Errorf("failed to load base template: %w", err)
	}

	// Build account map (code -> account), base first
	accountMap := make(map[string]TemplateAccount)
	for _, a := range base.Accounts {
		accountMap[a.Code] = a
	}

	// Load and merge industry templates
	for _, id := range industryIDs {
		tpl, err := LoadTemplate(dir, id)
		if err != nil {
			return nil, fmt.Errorf("failed to load template %s: %w", id, err)
		}

		for _, a := range tpl.Accounts {
			if _, exists := accountMap[a.Code]; !exists {
				accountMap[a.Code] = a
			}
		}
	}

	// Convert map to sorted slice
	result := make([]TemplateAccount, 0, len(accountMap))
	for _, a := range accountMap {
		result = append(result, a)
	}

	sortAccounts(result)
	return result, nil
}

// sortAccounts sorts template accounts by code
func sortAccounts(accounts []TemplateAccount) {
	sort.Slice(accounts, func(i, j int) bool {
		return accounts[i].Code < accounts[j].Code
	})
}

// ApplyTemplateToBook creates Account records from templates for a given book
func ApplyTemplateToBook(db *gorm.DB, bookID uint, dir string, industryIDs []string, taxpayerType, accountingStandard string) error {
	var templates []TemplateAccount

	// Try v2 templates first
	if len(industryIDs) > 0 {
		v2Tpl, err := LoadV2Template(dir, accountingStandard, industryIDs[0], taxpayerType)
		if err == nil {
			templates = v2Tpl.Accounts
		}
	}

	// Fallback to old template system
	if templates == nil {
		var err error
		templates, err = LoadAndMergeTemplates(dir, industryIDs)
		if err != nil {
			return err
		}
	}

	// Build parent code map for determining level
	codeSet := make(map[string]bool)
	for _, a := range templates {
		codeSet[a.Code] = true
	}

	for _, tpl := range templates {
		parentCode := tpl.Parent
		if parentCode == "" && tpl.Level > 1 {
			parentCode = detectParent(tpl.Code)
		}

		isLeaf := true
		for _, other := range templates {
			if other.Parent == tpl.Code || (other.Parent == "" && detectParent(other.Code) == tpl.Code) {
				isLeaf = false
				break
			}
		}

		level := tpl.Level
		if level == 0 {
			level = calcLevel(tpl.Code)
		}

		account := models.Account{
			BookID:     bookID,
			Code:       tpl.Code,
			Name:       tpl.Name,
			ParentCode: parentCode,
			Direction:  normalizeDirection(tpl.Dir),
			Level:      level,
			IsLeaf:     isLeaf,
			IsSystem:   true,
			IsActive:   true,
			AuxTypes:   strings.Join(tpl.Aux, ","),
		}

		if err := db.Create(&account).Error; err != nil {
			return fmt.Errorf("failed to create account %s: %w", tpl.Code, err)
		}
	}

	return nil
}

// detectParent returns the parent code by removing the last segment
func detectParent(code string) string {
	parts := strings.Split(code, ".")
	if len(parts) <= 1 {
		return ""
	}
	return strings.Join(parts[:len(parts)-1], ".")
}

// calcLevel calculates the level from the code
func calcLevel(code string) int {
	return len(strings.Split(code, "."))
}

// normalizeDirection normalizes direction to Chinese
func normalizeDirection(d string) string {
	switch strings.ToLower(d) {
	case "debit", "借":
		return "借"
	case "credit", "贷":
		return "贷"
	default:
		return "借"
	}
}

// SyncTemplateUpdates syncs template updates to an existing book
// Only adds new accounts, never modifies existing ones
func SyncTemplateUpdates(db *gorm.DB, bookID uint, dir string, industryIDs []string) error {
	templates, err := LoadAndMergeTemplates(dir, industryIDs)
	if err != nil {
		return err
	}

	var existing []models.Account
	db.Where("book_id = ?", bookID).Find(&existing)
	existingCodes := make(map[string]bool)
	for _, a := range existing {
		existingCodes[a.Code] = true
	}

	for _, tpl := range templates {
		if existingCodes[tpl.Code] {
			continue
		}

		parentCode := tpl.Parent
		if parentCode == "" && tpl.Level > 1 {
			parentCode = detectParent(tpl.Code)
		}

		level := tpl.Level
		if level == 0 {
			level = calcLevel(tpl.Code)
		}

		account := models.Account{
			BookID:     bookID,
			Code:       tpl.Code,
			Name:       tpl.Name,
			ParentCode: parentCode,
			Direction:  normalizeDirection(tpl.Dir),
			Level:      level,
			IsLeaf:     true,
			IsSystem:   true,
			IsActive:   true,
			AuxTypes:   strings.Join(tpl.Aux, ","),
		}

		if err := db.Create(&account).Error; err != nil {
			return fmt.Errorf("failed to create account %s: %w", tpl.Code, err)
		}
	}

	// Update is_leaf flags
	var allAccounts []models.Account
	db.Where("book_id = ?", bookID).Find(&allAccounts)
	for i, a := range allAccounts {
		isLeaf := true
		for _, other := range allAccounts {
			if other.ParentCode == a.Code {
				isLeaf = false
				break
			}
		}
		if allAccounts[i].IsLeaf != isLeaf {
			db.Model(&allAccounts[i]).Update("is_leaf", isLeaf)
		}
	}

	return nil
}

// GetManifest loads and returns the v2 manifest
func GetManifest(dir string) (*Manifest, error) {
	manifestPath := filepath.Join(dir, "v2", "manifest.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, err
	}
	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}
	return &manifest, nil
}
