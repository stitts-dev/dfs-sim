package providers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/jstittsworth/dfs-optimizer/internal/dfs"
)

type DraftKingsProvider struct {
	groupCache sync.Map // map[groupID]cachedGroup
}

// DraftKingsContestInfo represents contest information from DraftKings API
type DraftKingsContestInfo struct {
	ID                int     `json:"id"`
	Name              string  `json:"n"`
	EntryFee          float64 `json:"a"`
	PrizePool         float64 `json:"po"`
	MaxEntries        int     `json:"m"`
	TotalEntries      int     `json:"uc"`
	StartTime         string  `json:"sd"`
	IsMultiEntry      bool    `json:"mec"`
	MaxLineupsPerUser int     `json:"ulc"`
	ContestType       string  `json:"attr"`
	SalaryCap         int     `json:"sc"`
	DraftGroupID      int     `json:"dg"`
	Sport             string  `json:"s"`
	IsActive          bool    `json:"isActive"`
}

type cachedGroup struct {
	lastFetch time.Time
	players   []dfs.PlayerData
}

func NewDraftKingsProvider() *DraftKingsProvider {
	return &DraftKingsProvider{}
}

func (dk *DraftKingsProvider) GetPlayers(sport dfs.Sport, date string) ([]dfs.PlayerData, error) {
	// Map sport to DraftKings API sport param
	var dkSport string
	switch sport {
	case dfs.SportNBA:
		dkSport = "NBA"
	case dfs.SportNFL:
		dkSport = "NFL"
	case dfs.SportMLB:
		dkSport = "MLB"
	case dfs.SportNHL:
		dkSport = "NHL"
	case dfs.SportGolf:
		dkSport = "GOLF"
	case "lol":
		dkSport = "LOL"
	default:
		dkSport = "NBA" // Default fallback
	}

	contestsURL := fmt.Sprintf("https://www.draftkings.com/lobby/getcontests?sport=%s", dkSport)
	resp, err := http.Get(contestsURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var contestsResp map[string]interface{}
	if err := json.Unmarshal(body, &contestsResp); err != nil {
		return nil, err
	}
	contests, ok := contestsResp["Contests"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected contests format")
	}
	groupIDs := make(map[string]struct{})
	for _, c := range contests {
		contest, ok := c.(map[string]interface{})
		if !ok {
			continue
		}
		if dg, ok := contest["dg"].(float64); ok {
			groupIDs[fmt.Sprintf("%.0f", dg)] = struct{}{}
		}
	}
	var players []dfs.PlayerData
	for groupID := range groupIDs {
		// Rate limit: only fetch once per hour per groupID
		if val, ok := dk.groupCache.Load(groupID); ok {
			cache := val.(cachedGroup)
			if time.Since(cache.lastFetch) < time.Hour {
				players = append(players, cache.players...)
				continue
			}
		}
		draftablesURL := fmt.Sprintf("https://api.draftkings.com/draftgroups/v1/draftgroups/%s/draftables", groupID)
		drResp, err := http.Get(draftablesURL)
		if err != nil {
			log.Printf("DraftKings fetch error for group %s: %v", groupID, err)
			continue
		}
		defer drResp.Body.Close()
		drBody, err := ioutil.ReadAll(drResp.Body)
		if err != nil {
			log.Printf("DraftKings read error for group %s: %v", groupID, err)
			continue
		}
		var draftablesData map[string]interface{}
		if err := json.Unmarshal(drBody, &draftablesData); err != nil {
			log.Printf("DraftKings JSON error for group %s: %v", groupID, err)
			continue
		}
		draftables, ok := draftablesData["draftables"].([]interface{})
		if !ok {
			log.Printf("DraftKings draftables format error for group %s", groupID)
			continue
		}
		var groupPlayers []dfs.PlayerData
		for _, d := range draftables {
			draftable, ok := d.(map[string]interface{})
			if !ok {
				continue
			}
			player := dfs.PlayerData{
				ExternalID:  fmt.Sprintf("%v", draftable["playerId"]),
				Name:        fmt.Sprintf("%v", draftable["displayName"]),
				Team:        fmt.Sprintf("%v", draftable["teamAbbreviation"]),
				Position:    fmt.Sprintf("%v", draftable["position"]),
				Stats:       map[string]float64{},
				Source:      "draftkings",
				LastUpdated: time.Now(),
			}
			if salary, ok := draftable["salary"].(float64); ok {
				player.Stats["salary"] = salary
			}
			groupPlayers = append(groupPlayers, player)
		}
		dk.groupCache.Store(groupID, cachedGroup{lastFetch: time.Now(), players: groupPlayers})
		players = append(players, groupPlayers...)
	}
	return players, nil
}

// GetContests fetches available contests from DraftKings API
func (dk *DraftKingsProvider) GetContests(sport dfs.Sport) ([]DraftKingsContestInfo, error) {
	// Map sport to DraftKings API sport param
	var dkSport string
	switch sport {
	case dfs.SportNBA:
		dkSport = "NBA"
	case dfs.SportNFL:
		dkSport = "NFL"
	case dfs.SportMLB:
		dkSport = "MLB"
	case dfs.SportNHL:
		dkSport = "NHL"
	case dfs.SportGolf:
		dkSport = "GOLF"
	case "lol":
		dkSport = "LOL"
	default:
		dkSport = "NBA" // Default fallback
	}

	contestsURL := fmt.Sprintf("https://www.draftkings.com/lobby/getcontests?sport=%s", dkSport)
	resp, err := http.Get(contestsURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch contests: %w", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var contestsResp map[string]interface{}
	if err := json.Unmarshal(body, &contestsResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	contests, ok := contestsResp["Contests"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected contests format in response")
	}

	var contestInfos []DraftKingsContestInfo
	for _, c := range contests {
		contest, ok := c.(map[string]interface{})
		if !ok {
			continue
		}

		contestInfo := DraftKingsContestInfo{
			Sport: dkSport,
		}

		// Extract contest fields with proper type checking
		if id, ok := contest["id"].(float64); ok {
			contestInfo.ID = int(id)
		}
		if name, ok := contest["n"].(string); ok {
			contestInfo.Name = name
		}
		if fee, ok := contest["a"].(float64); ok {
			contestInfo.EntryFee = fee
		}
		if pool, ok := contest["po"].(float64); ok {
			contestInfo.PrizePool = pool
		}
		if maxEntries, ok := contest["m"].(float64); ok {
			contestInfo.MaxEntries = int(maxEntries)
		}
		if totalEntries, ok := contest["uc"].(float64); ok {
			contestInfo.TotalEntries = int(totalEntries)
		}
		if startTime, ok := contest["sd"].(string); ok {
			contestInfo.StartTime = startTime
		}
		// Handle mec field (can be bool or number)
		if multiEntry, ok := contest["mec"].(bool); ok {
			contestInfo.IsMultiEntry = multiEntry
		} else if multiEntry, ok := contest["mec"].(float64); ok {
			contestInfo.IsMultiEntry = multiEntry != 0
		}
		if maxLineups, ok := contest["ulc"].(float64); ok {
			contestInfo.MaxLineupsPerUser = int(maxLineups)
		}
		// Handle attr field (can be string or object)
		if attr, ok := contest["attr"].(string); ok {
			contestInfo.ContestType = attr
		} else if _, ok := contest["attr"].(map[string]interface{}); ok {
			// For object format, we'll derive contest type from the contest name or use a default
			contestInfo.ContestType = "gpp" // Default to GPP for object format
		}
		if salaryCap, ok := contest["sc"].(float64); ok {
			contestInfo.SalaryCap = int(salaryCap)
		}
		if draftGroup, ok := contest["dg"].(float64); ok {
			contestInfo.DraftGroupID = int(draftGroup)
		}
		// Handle isActive field (can be bool or number)
		if isActive, ok := contest["isActive"].(bool); ok {
			contestInfo.IsActive = isActive
		} else if isActive, ok := contest["isActive"].(float64); ok {
			contestInfo.IsActive = isActive != 0
		} else {
			// Default to active if not specified
			contestInfo.IsActive = true
		}

		contestInfos = append(contestInfos, contestInfo)
	}

	log.Printf("DraftKings: Found %d contests for sport %s", len(contestInfos), dkSport)
	return contestInfos, nil
}

func (dk *DraftKingsProvider) GetPlayer(sport dfs.Sport, externalID string) (*dfs.PlayerData, error) {
	return nil, nil // Not implemented
}

func (dk *DraftKingsProvider) GetTeamRoster(sport dfs.Sport, teamID string) ([]dfs.PlayerData, error) {
	return []dfs.PlayerData{}, nil // Not implemented
}
