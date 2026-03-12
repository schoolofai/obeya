package realtime

// EventAction represents the type of change on a document.
type EventAction string

const (
	ActionCreate EventAction = "create"
	ActionUpdate EventAction = "update"
	ActionDelete EventAction = "delete"
)

// BoardEvent represents a parsed realtime event scoped to a board.
type BoardEvent struct {
	Action       EventAction
	CollectionID string
	DocumentID   string
	Payload      map[string]interface{}
}

// SubscriptionConfig holds the parameters for a realtime subscription.
type SubscriptionConfig struct {
	AppwriteEndpoint string
	ProjectID        string
	APIToken         string
	DatabaseID       string
	BoardID          string
}

// websocketURL converts the Appwrite REST endpoint to a WebSocket URL.
func websocketURL(endpoint string) string {
	url := endpoint

	if len(url) > 8 && url[:8] == "https://" {
		url = "wss://" + url[8:]
	} else if len(url) > 7 && url[:7] == "http://" {
		url = "ws://" + url[7:]
	}

	if url[len(url)-1] == '/' {
		url += "realtime"
	} else {
		url += "/realtime"
	}

	return url
}

// parseEventAction extracts the action from an Appwrite realtime event string.
func parseEventAction(eventStr string) (EventAction, bool) {
	if eventStr == "" {
		return "", false
	}

	lastDot := -1
	for i := len(eventStr) - 1; i >= 0; i-- {
		if eventStr[i] == '.' {
			lastDot = i
			break
		}
	}
	if lastDot < 0 || lastDot >= len(eventStr)-1 {
		return "", false
	}

	suffix := eventStr[lastDot+1:]
	switch suffix {
	case "create":
		return ActionCreate, true
	case "update":
		return ActionUpdate, true
	case "delete":
		return ActionDelete, true
	default:
		return "", false
	}
}

// parseCollectionID extracts the collection ID from an Appwrite realtime event.
func parseCollectionID(eventStr string) string {
	start := 0
	partIndex := 0
	collectionsIdx := -1

	for i := 0; i <= len(eventStr); i++ {
		if i == len(eventStr) || eventStr[i] == '.' {
			if partIndex > 0 && collectionsIdx == partIndex-1 {
				return eventStr[start:i]
			}
			if i-start == 11 && eventStr[start:i] == "collections" {
				collectionsIdx = partIndex
			}
			partIndex++
			start = i + 1
		}
	}
	return ""
}

// parseRealtimeMessage parses a raw Appwrite realtime WebSocket message.
// Returns nil if the event is not relevant to the specified board.
func parseRealtimeMessage(msg map[string]interface{}, boardID string) *BoardEvent {
	eventsRaw, ok := msg["events"]
	if !ok {
		return nil
	}
	events, ok := eventsRaw.([]interface{})
	if !ok || len(events) == 0 {
		return nil
	}

	payloadRaw, ok := msg["payload"]
	if !ok {
		return nil
	}
	payload, ok := payloadRaw.(map[string]interface{})
	if !ok {
		return nil
	}

	payloadBoardID, _ := payload["board_id"].(string)
	if payloadBoardID != boardID {
		return nil
	}

	var action EventAction
	var collectionID string
	for _, e := range events {
		eventStr, ok := e.(string)
		if !ok {
			continue
		}
		a, valid := parseEventAction(eventStr)
		if valid {
			action = a
			collectionID = parseCollectionID(eventStr)
			break
		}
	}
	if action == "" {
		return nil
	}

	docID, _ := payload["$id"].(string)

	return &BoardEvent{
		Action:       action,
		CollectionID: collectionID,
		DocumentID:   docID,
		Payload:      payload,
	}
}
