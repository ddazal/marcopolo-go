package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// GetHolidaysInput represents the input parameters for get_holidays tool
type GetHolidaysInput struct {
	Year        string `json:"year"`
	CountryCode string `json:"countryCode"`
}

// Holiday represents a public holiday
type Holiday struct {
	Date        string   `json:"date"`
	LocalName   string   `json:"localName"`
	Name        string   `json:"name"`
	CountryCode string   `json:"countryCode"`
	Fixed       bool     `json:"fixed"`
	Global      bool     `json:"global"`
	Counties    []string `json:"counties,omitempty"`
	LaunchYear  *int     `json:"launchYear,omitempty"`
	Types       []string `json:"types"`
}

func init() {
	getHolidaysTool := ToolDefinition{
		Name:        "get_holidays",
		Description: "Retrieve the list of all public holidays for the specified year and country",
		Parameters: &Parameters{
			Properties: map[string]ParameterProperty{
				"year": {
					Type:        "string",
					Description: "The target year for which public holidays should be retrieved. If not specific\n  year is asked, default to the current year",
				},
				"countryCode": {
					Type:        "string",
					Description: "A valid ISO 3166-1 alpha-2 country code.",
				},
			},
			Required: []string{"year", "countryCode"},
		},
	}
	Register(getHolidaysTool)
}

// GetHolidays retrieves public holidays for a specific year and country
func GetHolidays(ctx context.Context, input GetHolidaysInput) ([]Holiday, error) {
	year := input.Year
	if year == "" {
		year = strconv.Itoa(time.Now().Year())
	}

	url := fmt.Sprintf("https://date.nager.at/api/v3/PublicHolidays/%s/%s",
		input.Year,
		strings.ToLower(input.CountryCode))

	request, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch holidays: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// TODO: handle error properly
		return []Holiday{}, nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var holidays []Holiday
	if err := json.Unmarshal(body, &holidays); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return holidays, nil
}
