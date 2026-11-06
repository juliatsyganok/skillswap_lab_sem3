package storage

import (
	"testing"

	"skillswap/domain"
)

func TestMatchingStoreIsolatesProfileSkills(t *testing.T) {
	store := NewInMemoryMatchingStore()
	profile := domain.User{
		ID: "u1", Offers: []domain.Skill{{Name: "go"}}, Needs: []domain.Skill{{Name: "python"}},
	}
	store.AddUser(profile)
	profile.Offers[0].Name = "changed after AddUser"
	fromGet, err := store.Get("u1")
	if err != nil {
		t.Fatal(err)
	}
	fromGet.Needs[0].Name = "changed after Get"
	fromList, err := store.List()
	if err != nil || len(fromList) != 1 {
		t.Fatalf("list: %+v, %v", fromList, err)
	}
	if fromList[0].Offers[0].Name != "go" || fromList[0].Needs[0].Name != "python" {
		t.Fatalf("stored profile changed through caller data: %+v", fromList[0])
	}
	fromList[0].Offers[0].Name = "changed after List"
	stored, err := store.Get("u1")
	if err != nil || stored.Offers[0].Name != "go" {
		t.Fatalf("stored profile changed through list: %+v, %v", stored, err)
	}
}
