package domain

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
)

// GenerateID creates an 8-character hex ID from 4 random bytes.
func GenerateID() string {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("failed to generate random ID: %v", err))
	}
	return hex.EncodeToString(b)
}

// ResolveID resolves a user-provided reference (display number or hash prefix) to a canonical item ID.
func (b *Board) ResolveID(ref string) (string, error) {
	// Try exact match first
	if _, ok := b.Items[ref]; ok {
		return ref, nil
	}

	// Try as display number
	if num, err := strconv.Atoi(ref); err == nil {
		if id, ok := b.DisplayMap[num]; ok {
			return id, nil
		}
		return "", fmt.Errorf("no item with display number %d", num)
	}

	// Try as hash prefix
	return b.resolveByPrefix(ref, b.itemIDs())
}

// ResolveUserID resolves a user reference (name or hash prefix) to a user ID.
func (b *Board) ResolveUserID(ref string) (string, error) {
	// Exact match
	if _, ok := b.Users[ref]; ok {
		return ref, nil
	}

	// Try by name (case-insensitive)
	for id, u := range b.Users {
		if strings.EqualFold(u.Name, ref) {
			return id, nil
		}
	}

	// Try as hash prefix
	return b.resolveByPrefix(ref, b.userIDs())
}

func (b *Board) itemIDs() []string {
	ids := make([]string, 0, len(b.Items))
	for id := range b.Items {
		ids = append(ids, id)
	}
	return ids
}

func (b *Board) userIDs() []string {
	ids := make([]string, 0, len(b.Users))
	for id := range b.Users {
		ids = append(ids, id)
	}
	return ids
}

func (b *Board) resolveByPrefix(ref string, ids []string) (string, error) {
	var matches []string
	for _, id := range ids {
		if strings.HasPrefix(id, ref) {
			matches = append(matches, id)
		}
	}

	switch len(matches) {
	case 0:
		return "", fmt.Errorf("no item found matching %q", ref)
	case 1:
		return matches[0], nil
	default:
		return "", fmt.Errorf("ambiguous reference %q matches %d items: %s", ref, len(matches), strings.Join(matches, ", "))
	}
}
