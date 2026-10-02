package discovery

import (
	"errors"
	"testing"

	"skillswap/domain"
)

type fakeReader struct {
	offers []domain.Offer
	err    error
}

func (f *fakeReader) GetOffers() ([]domain.Offer, error) {
	return f.offers, f.err
}

func TestIsRelevant(t *testing.T) {
	user := domain.User{
		ID: "user-1",
		Offers: []domain.Skill{
			{Name: "English", Level: domain.LevelBeginner},
		},
		Needs: []domain.Skill{
			{Name: "Go", Level: domain.LevelBeginner},
		},
		Format: domain.FormatOnline,
	}

	tests := []struct {
		name  string
		user  domain.User
		offer domain.Offer
		want  bool
	}{
		{
			name: "relevant",
			user: user,
			offer: domain.Offer{
				ID:            "offer-1",
				AuthorID:      "user-2",
				Skill:         domain.Skill{Name: "Go", Level: domain.LevelAdvanced},
				WantsInReturn: []domain.Skill{{Name: "English"}},
				Format:        domain.FormatOnline,
				Status:        domain.OfferActive,
			},
			want: true,
		},
		{
			name: "wrong skill",
			user: user,
			offer: domain.Offer{
				ID:       "offer-2",
				AuthorID: "user-2",
				Skill:    domain.Skill{Name: "Python"},
				Format:   domain.FormatOnline,
				Status:   domain.OfferActive,
			},
			want: false,
		},
		{
			name: "not active",
			user: user,
			offer: domain.Offer{
				ID:       "offer-3",
				AuthorID: "user-2",
				Skill:    domain.Skill{Name: "Go"},
				Format:   domain.FormatOnline,
				Status:   domain.OfferPaused,
			},
			want: false,
		},
		{
			name: "own offer",
			user: user,
			offer: domain.Offer{
				ID:       "offer-4",
				AuthorID: "user-1",
				Skill:    domain.Skill{Name: "Go"},
				Format:   domain.FormatOnline,
				Status:   domain.OfferActive,
			},
			want: false,
		},
		{
			name: "wrong format",
			user: user,
			offer: domain.Offer{
				ID:       "offer-5",
				AuthorID: "user-2",
				Skill:    domain.Skill{Name: "Go"},
				Format:   domain.FormatOffline,
				Status:   domain.OfferActive,
			},
			want: false,
		},
		{
			name: "any format",
			user: user,
			offer: domain.Offer{
				ID:       "offer-6",
				AuthorID: "user-2",
				Skill:    domain.Skill{Name: "Go"},
				Format:   domain.FormatAny,
				Status:   domain.OfferActive,
			},
			want: true,
		},
		{
			name: "user does not offer wanted skill",
			user: user,
			offer: domain.Offer{
				ID:            "offer-7",
				AuthorID:      "user-2",
				Skill:         domain.Skill{Name: "Go"},
				WantsInReturn: []domain.Skill{{Name: "Python"}},
				Format:        domain.FormatOnline,
				Status:        domain.OfferActive,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsRelevant(tt.user, tt.offer)
			if got != tt.want {
				t.Errorf("IsRelevant() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFindRelevant(t *testing.T) {
	user := domain.User{
		ID:     "user-1",
		Needs:  []domain.Skill{{Name: "Go"}},
		Format: domain.FormatOnline,
	}
	offers := []domain.Offer{
		{
			ID:       "good-1",
			AuthorID: "user-2",
			Skill:    domain.Skill{Name: "Go"},
			Format:   domain.FormatOnline,
			Status:   domain.OfferActive,
		},
		{
			ID:       "bad",
			AuthorID: "user-3",
			Skill:    domain.Skill{Name: "Python"},
			Format:   domain.FormatOnline,
			Status:   domain.OfferActive,
		},
		{
			ID:       "good-2",
			AuthorID: "user-4",
			Skill:    domain.Skill{Name: "Go"},
			Format:   domain.FormatAny,
			Status:   domain.OfferActive,
		},
	}

	got := FindRelevant(user, offers)

	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].ID != "good-1" || got[1].ID != "good-2" {
		t.Errorf("got offers %q and %q", got[0].ID, got[1].ID)
	}
}

func TestServiceFindRelevantStorageError(t *testing.T) {
	svc := NewService(&fakeReader{err: domain.ErrStorageUnavailable})

	_, err := svc.FindRelevant(domain.User{ID: "user-1"})

	if !errors.Is(err, domain.ErrStorageUnavailable) {
		t.Errorf("expected ErrStorageUnavailable, got %v", err)
	}
}
