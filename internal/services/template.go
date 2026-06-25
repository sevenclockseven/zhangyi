package services

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Version     string            `json:"version"`
	Base        string            `json:"base"`
	Description string            `json:"description"`
	Accounts    []TemplateAccount `json:"accounts"`
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
			// Industry template adds new accounts or overrides existing
			// If same code exists, keep base version (industry won't override base)
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

	// Sort by code
	sortAccounts(result)

	return result, nil
}

// sortAccounts sorts template accounts by code
func sortAccounts(accounts []TemplateAccount) {
	for i := 0; i < len(accounts); i++ {
		for j := i + 1; j < len(accounts); j++ {
			if accounts[j].Code < accounts[i].Code {
				accounts[i], accounts[j] = accounts[j], accounts[i]
			}
		}
	}
}

// ApplyTemplateToBook creates Account records from templates for a given book
func ApplyTemplateToBook(db *gorm.DB, bookID uint, dir string, industryIDs []string) error {
	templates, err := LoadAndMergeTemplates(dir, industryIDs)
	if err != nil {
		return err
	}

	// Build parent code map for determining level
	codeSet := make(map[string]bool)
	for _, a := range templates {
		codeSet[a.Code] = true
	}

	for _, tpl := range templates {
		// Determine parent code
		parentCode := tpl.Parent
		if parentCode == "" && tpl.Level > 1 {
			// Auto-detect parent: remove last segment
			parentCode = detectParent(tpl.Code)
		}

		// Determine if leaf: no other account has this as parent
		isLeaf := true
		for _, other := range templates {
			if other.Parent == tpl.Code || (other.Parent == "" && detectParent(other.Code) == tpl.Code) {
				isLeaf = false
				break
			}
		}

		// Determine level
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

	// Get existing account codes
	var existing []models.Account
	db.Where("book_id = ?", bookID).Find(&existing)
	existingCodes := make(map[string]bool)
	for _, a := range existing {
		existingCodes[a.Code] = true
	}

	// Only add new accounts
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
