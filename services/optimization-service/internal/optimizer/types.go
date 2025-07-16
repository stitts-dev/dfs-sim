package optimizer

import (
	"time"

	"github.com/google/uuid"
)

// OptimizationPlayer represents a player for optimization algorithms
// This type uses concrete values instead of pointers for algorithm compatibility
type OptimizationPlayer struct {
	ID              uuid.UUID `json:"id"`
	ExternalID      string    `json:"external_id"`
	Name            string    `json:"name"`
	Team            string    `json:"team"`
	Opponent        string    `json:"opponent"`
	Position        string    `json:"position"`
	SalaryDK        int       `json:"salary_dk"`
	SalaryFD        int       `json:"salary_fd"`
	ProjectedPoints float64   `json:"projected_points"`
	FloorPoints     float64   `json:"floor_points"`
	CeilingPoints   float64   `json:"ceiling_points"`
	OwnershipDK     float64   `json:"ownership_dk"`
	OwnershipFD     float64   `json:"ownership_fd"`
	GameTime        time.Time `json:"game_time"`
	IsInjured       bool      `json:"is_injured"`
	InjuryStatus    string    `json:"injury_status"`
	ImageURL        string    `json:"image_url"`
	// Golf-specific fields
	TeeTime         string    `json:"tee_time,omitempty"`
	CutProbability  float64   `json:"cut_probability,omitempty"`
	// Metadata
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// GetProjectedPoints implements the expected interface for optimization algorithms
func (p OptimizationPlayer) GetProjectedPoints() float64 {
	return p.ProjectedPoints
}

// GetFloorPoints implements the expected interface for optimization algorithms
func (p OptimizationPlayer) GetFloorPoints() float64 {
	return p.FloorPoints
}

// GetCeilingPoints implements the expected interface for optimization algorithms
func (p OptimizationPlayer) GetCeilingPoints() float64 {
	return p.CeilingPoints
}

// GetOwnershipDK implements the expected interface for optimization algorithms
func (p OptimizationPlayer) GetOwnershipDK() float64 {
	return p.OwnershipDK
}

// GetOwnershipFD implements the expected interface for optimization algorithms
func (p OptimizationPlayer) GetOwnershipFD() float64 {
	return p.OwnershipFD
}

// GetTeam implements the expected interface for optimization algorithms
func (p OptimizationPlayer) GetTeam() string {
	return p.Team
}

// GetOpponent implements the expected interface for optimization algorithms
func (p OptimizationPlayer) GetOpponent() string {
	return p.Opponent
}

// GetPosition implements the expected interface for optimization algorithms
func (p OptimizationPlayer) GetPosition() string {
	return p.Position
}

// GetInjuryStatus implements the expected interface for optimization algorithms
func (p OptimizationPlayer) GetInjuryStatus() string {
	return p.InjuryStatus
}

// GetName implements the expected interface for optimization algorithms
func (p OptimizationPlayer) GetName() string {
	return p.Name
}

// GetID implements the expected interface for optimization algorithms
func (p OptimizationPlayer) GetID() uuid.UUID {
	return p.ID
}

// GetExternalID implements the expected interface for optimization algorithms
func (p OptimizationPlayer) GetExternalID() string {
	return p.ExternalID
}

// GetSalaryDK implements the expected interface for optimization algorithms
func (p OptimizationPlayer) GetSalaryDK() int {
	return p.SalaryDK
}

// GetSalaryFD implements the expected interface for optimization algorithms
func (p OptimizationPlayer) GetSalaryFD() int {
	return p.SalaryFD
}

// GetGameTime implements the expected interface for optimization algorithms
func (p OptimizationPlayer) GetGameTime() time.Time {
	return p.GameTime
}

// IsPlayerInjured implements the expected interface for optimization algorithms
func (p OptimizationPlayer) IsPlayerInjured() bool {
	return p.IsInjured
}

// GetImageURL implements the expected interface for optimization algorithms
func (p OptimizationPlayer) GetImageURL() string {
	return p.ImageURL
}