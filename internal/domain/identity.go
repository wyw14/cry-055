package domain

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
	"time"
)

type ID string

func NewID(prefix string) ID {
	var data [8]byte
	if _, err := rand.Read(data[:]); err != nil {
		stamp := time.Now().UTC().Format("20060102150405.000000000")
		return ID(strings.ToLower(prefix) + "_" + strings.ReplaceAll(stamp, ".", ""))
	}
	return ID(strings.ToLower(prefix) + "_" + hex.EncodeToString(data[:]))
}

func (id ID) Empty() bool { return strings.TrimSpace(string(id)) == "" }

type Version int64

func (v Version) Next() Version { return v + 1 }

type Actor struct {
	ID    ID
	Name  string
	Roles []string
}

func (a Actor) HasRole(role string) bool {
	for _, item := range a.Roles {
		if item == role {
			return true
		}
	}
	return false
}
