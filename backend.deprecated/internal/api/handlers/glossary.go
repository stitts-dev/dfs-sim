package handlers

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jstittsworth/dfs-optimizer/internal/models"
	"github.com/jstittsworth/dfs-optimizer/internal/services"
	"github.com/jstittsworth/dfs-optimizer/pkg/database"
	"github.com/jstittsworth/dfs-optimizer/pkg/utils"
	"gorm.io/gorm"
)

type GlossaryHandler struct {
	db    *database.DB
	cache *services.CacheService
}

func NewGlossaryHandler(db *database.DB, cache *services.CacheService) *GlossaryHandler {
	return &GlossaryHandler{
		db:    db,
		cache: cache,
	}
}

// GetGlossaryTerms returns all glossary terms with optional filters
// GET /api/glossary?category=general&difficulty=beginner&sport=nfl
func (h *GlossaryHandler) GetGlossaryTerms(c *gin.Context) {
	// Get query parameters
	category := c.Query("category")
	difficulty := c.Query("difficulty")
	sport := c.Query("sport")

	// Validate category if provided
	if category != "" {
		validCategories := []string{"general", "sport-specific", "strategy", "platform"}
		isValid := false
		for _, valid := range validCategories {
			if category == valid {
				isValid = true
				break
			}
		}
		if !isValid {
			utils.SendValidationError(c, "Invalid category", "Category must be one of: general, sport-specific, strategy, platform")
			return
		}
	}

	// Validate difficulty if provided
	if difficulty != "" {
		validDifficulties := []string{"beginner", "intermediate", "advanced"}
		isValid := false
		for _, valid := range validDifficulties {
			if difficulty == valid {
				isValid = true
				break
			}
		}
		if !isValid {
			utils.SendValidationError(c, "Invalid difficulty", "Difficulty must be one of: beginner, intermediate, advanced")
			return
		}
	}

	// Get terms from database
	terms, err := models.GetGlossaryTerms(h.db, category, difficulty, sport)
	if err != nil {
		utils.SendInternalError(c, "Failed to fetch glossary terms")
		return
	}

	utils.SendSuccess(c, terms)
}

// SearchGlossaryTerms searches glossary terms by term or definition
// GET /api/glossary/search?q=stack
func (h *GlossaryHandler) SearchGlossaryTerms(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		utils.SendValidationError(c, "Search query required", "Please provide a search query using the 'q' parameter")
		return
	}

	// Minimum query length
	if len(query) < 2 {
		utils.SendValidationError(c, "Query too short", "Search query must be at least 2 characters")
		return
	}

	// Search terms
	terms, err := models.SearchGlossaryTerms(h.db, query)
	if err != nil {
		utils.SendInternalError(c, "Failed to search glossary terms")
		return
	}

	utils.SendSuccess(c, terms)
}

// GetGlossaryTerm returns a specific glossary term by ID or term name
// GET /api/glossary/:term (can be ID or term name)
func (h *GlossaryHandler) GetGlossaryTerm(c *gin.Context) {
	termParam := c.Param("term")

	var term models.GlossaryTerm
	var err error

	// Try to parse as ID first
	if id, parseErr := strconv.ParseUint(termParam, 10, 32); parseErr == nil {
		// It's a numeric ID
		err = h.db.Where("id = ?", id).First(&term).Error
	} else {
		// It's a term name - case insensitive search
		err = h.db.Where("LOWER(term) = LOWER(?)", strings.TrimSpace(termParam)).First(&term).Error
	}

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.SendNotFound(c, "Glossary term not found")
			return
		}
		utils.SendInternalError(c, "Failed to fetch glossary term")
		return
	}

	utils.SendSuccess(c, term)
}
