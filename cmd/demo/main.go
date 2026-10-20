package main

import (
	"fmt"

	"skillswap/domain"
	"skillswap/internal/agreement"
	"skillswap/internal/response"
	"skillswap/internal/storage"
)

var _ response.Repo = (*storage.InMemoryOfferStore)(nil)

func main() {
	store := storage.NewInMemoryAgreementStore()
	offer := domain.Offer{ID: "offer-1", AuthorID: "author", Status: domain.OfferActive}
	resp := domain.Response{ID: "response-1", OfferID: offer.ID, UserID: "learner", Status: domain.ResponsePending}
	if err := store.SaveOffer(offer); err != nil {
		panic(err)
	}
	if err := store.SaveResponse(resp); err != nil {
		panic(err)
	}
	svc := agreement.NewService(store)

	printState("до подтверждения", store)
	accepted, err := svc.Accept("author", resp.ID)
	if err != nil {
		panic(err)
	}
	printResult("подтверждение", accepted, store)
	accepted, err = svc.Accept("author", resp.ID)
	if err != nil {
		panic(err)
	}
	printResult("повтор (no-op)", accepted, store)
	read, err := store.GetResponse(resp.ID)
	if err != nil {
		panic(err)
	}
	fmt.Printf("чтение: %s\n", read.Status)
}

func printState(label string, store *storage.InMemoryAgreementStore) {
	r, err := store.GetResponse("response-1")
	if err != nil {
		panic(err)
	}
	fmt.Printf("%s: learner=%s, author=%s\n", label, r.Status, r.Status)
}

func printResult(label string, r domain.Response, store *storage.InMemoryAgreementStore) {
	stored, err := store.GetResponse(r.ID)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%s: learner=%s, author=%s (записей 1; в хранилище %s)\n", label, r.Status, stored.Status, stored.Status)
}
