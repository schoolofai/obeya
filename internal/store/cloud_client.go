package store

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/niladribose/obeya/internal/domain"
)

// APIResponse is the standard envelope from the Obeya Cloud API.
type APIResponse struct {
	OK    bool            `json:"ok"`
	Data  interface{}     `json:"data,omitempty"`
	Error *APIError       `json:"error,omitempty"`
	Meta  json.RawMessage `json:"meta,omitempty"`
}

// APIError represents an error from the Cloud API.
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// CloudClient handles HTTP communication with the Obeya Cloud API.
type CloudClient struct {
	apiURL     string
	token      string
	httpClient *http.Client
}

// NewCloudClient creates a new CloudClient with the given API base URL and auth token.
func NewCloudClient(apiURL, token string) *CloudClient {
	return &CloudClient{
		apiURL: apiURL,
		token:  token,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ExportBoard fetches the full board as a domain.Board from the export endpoint.
func (c *CloudClient) ExportBoard(boardID string) (*domain.Board, error) {
	url := fmt.Sprintf("%s/boards/%s/export", c.apiURL, boardID)

	body, err := c.doRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("export board request failed: %w", err)
	}

	var board domain.Board
	if err := json.Unmarshal(body, &board); err != nil {
		return nil, fmt.Errorf("failed to parse exported board: %w", err)
	}

	initBoardNilMaps(&board)
	return &board, nil
}

// CreateItem sends a POST to create an item on the given board.
func (c *CloudClient) CreateItem(boardID string, item *domain.Item) error {
	url := fmt.Sprintf("%s/boards/%s/items", c.apiURL, boardID)

	payload, err := json.Marshal(item)
	if err != nil {
		return fmt.Errorf("failed to marshal item: %w", err)
	}

	_, err = c.doRequest(http.MethodPost, url, payload)
	if err != nil {
		return fmt.Errorf("create item request failed: %w", err)
	}

	return nil
}

// UpdateItem sends a PATCH to update an existing item.
func (c *CloudClient) UpdateItem(item *domain.Item) error {
	url := fmt.Sprintf("%s/items/%s", c.apiURL, item.ID)

	payload, err := json.Marshal(item)
	if err != nil {
		return fmt.Errorf("failed to marshal item: %w", err)
	}

	_, err = c.doRequest(http.MethodPatch, url, payload)
	if err != nil {
		return fmt.Errorf("update item request failed: %w", err)
	}

	return nil
}

// MoveItem sends a POST to change an item's status column.
func (c *CloudClient) MoveItem(itemID, status string) error {
	url := fmt.Sprintf("%s/items/%s/move", c.apiURL, itemID)

	payload, err := json.Marshal(map[string]string{"status": status})
	if err != nil {
		return fmt.Errorf("failed to marshal move payload: %w", err)
	}

	_, err = c.doRequest(http.MethodPost, url, payload)
	if err != nil {
		return fmt.Errorf("move item request failed: %w", err)
	}

	return nil
}

// DeleteItem sends a DELETE to remove an item.
func (c *CloudClient) DeleteItem(itemID string) error {
	url := fmt.Sprintf("%s/items/%s", c.apiURL, itemID)

	_, err := c.doRequest(http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("delete item request failed: %w", err)
	}

	return nil
}

// ImportBoard sends the full board.json for server-side migration.
// Returns the new cloud board ID.
func (c *CloudClient) ImportBoard(board *domain.Board, orgID string) (string, error) {
	url := fmt.Sprintf("%s/boards/import", c.apiURL)

	type importPayload struct {
		Board *domain.Board `json:"board"`
		OrgID string        `json:"org_id,omitempty"`
	}

	payload, err := json.Marshal(importPayload{Board: board, OrgID: orgID})
	if err != nil {
		return "", fmt.Errorf("failed to marshal import payload: %w", err)
	}

	body, err := c.doRequest(http.MethodPost, url, payload)
	if err != nil {
		return "", fmt.Errorf("import board request failed: %w", err)
	}

	return extractStringField(body, "board_id")
}

// CreateBoard creates a new empty board on the cloud.
// Returns the new cloud board ID.
func (c *CloudClient) CreateBoard(name string, columns []string, orgID string) (string, error) {
	url := fmt.Sprintf("%s/boards", c.apiURL)

	type createPayload struct {
		Name    string   `json:"name"`
		Columns []string `json:"columns"`
		OrgID   string   `json:"org_id,omitempty"`
	}

	payload, err := json.Marshal(createPayload{Name: name, Columns: columns, OrgID: orgID})
	if err != nil {
		return "", fmt.Errorf("failed to marshal create board payload: %w", err)
	}

	body, err := c.doRequest(http.MethodPost, url, payload)
	if err != nil {
		return "", fmt.Errorf("create board request failed: %w", err)
	}

	return extractStringField(body, "board_id")
}

// GetMe fetches the current user's profile. Returns (userID, username, error).
func (c *CloudClient) GetMe() (string, string, error) {
	url := fmt.Sprintf("%s/auth/me", c.apiURL)

	body, err := c.doRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", "", fmt.Errorf("get me request failed: %w", err)
	}

	userID, err := extractStringField(body, "user_id")
	if err != nil {
		return "", "", err
	}

	username, err := extractStringField(body, "username")
	if err != nil {
		return "", "", err
	}

	return userID, username, nil
}

// doRequest executes an HTTP request with auth headers and parses the API envelope.
// Returns the raw JSON bytes of the "data" field on success.
func (c *CloudClient) doRequest(method, url string, payload []byte) (json.RawMessage, error) {
	var bodyReader io.Reader
	if payload != nil {
		bodyReader = bytes.NewReader(payload)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var apiResp struct {
		OK    bool            `json:"ok"`
		Data  json.RawMessage `json:"data"`
		Error *APIError       `json:"error,omitempty"`
	}
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse API response (status %d): %w", resp.StatusCode, err)
	}

	if !apiResp.OK {
		errCode := "UNKNOWN"
		errMsg := "unknown API error"
		if apiResp.Error != nil {
			errCode = apiResp.Error.Code
			errMsg = apiResp.Error.Message
		}
		return nil, fmt.Errorf("API error [%s]: %s", errCode, errMsg)
	}

	return apiResp.Data, nil
}

// extractStringField extracts a string field from a JSON raw message.
func extractStringField(data json.RawMessage, field string) (string, error) {
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return "", fmt.Errorf("failed to parse response data: %w", err)
	}

	val, ok := m[field]
	if !ok {
		return "", fmt.Errorf("response missing field: %s", field)
	}

	str, ok := val.(string)
	if !ok {
		return "", fmt.Errorf("field %s is not a string", field)
	}

	return str, nil
}

// initBoardNilMaps ensures all map fields on a Board are non-nil.
func initBoardNilMaps(board *domain.Board) {
	if board.Items == nil {
		board.Items = make(map[string]*domain.Item)
	}
	if board.DisplayMap == nil {
		board.DisplayMap = make(map[int]string)
	}
	if board.Users == nil {
		board.Users = make(map[string]*domain.Identity)
	}
	if board.Plans == nil {
		board.Plans = make(map[string]*domain.Plan)
	}
	if board.Projects == nil {
		board.Projects = make(map[string]*domain.LinkedProject)
	}
}
