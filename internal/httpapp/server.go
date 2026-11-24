package httpapp

import (
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"skillswap/domain"
	"skillswap/internal/agreement"
	"skillswap/internal/discovery"
	"skillswap/internal/matching"
	"skillswap/internal/storage"
)

// Dependencies is the composition boundary for the team's route registration.
type Dependencies struct {
	Agreement  *agreement.Service
	Responses  agreement.EngagementReader
	Profiles   matching.ProfileReader
	Matching   *matching.Service
	Discovery  *discovery.Service
	Config     Config
	HTTPClient *http.Client
}

func NewServer(cfg Config, logger *slog.Logger, register func(*http.ServeMux, Dependencies)) (*http.Server, error) {
	state := storage.NewInMemoryAgreementStore()
	profiles := storage.NewInMemoryMatchingStore()
	if err := seedDemo(state, profiles); err != nil {
		return nil, fmt.Errorf("seed demo data: %w", err)
	}

	transport := &http.Transport{
		DialContext:           (&net.Dialer{Timeout: cfg.UpstreamDialTimeout}).DialContext,
		ResponseHeaderTimeout: cfg.UpstreamResponseHeaderTimeout,
		TLSHandshakeTimeout:   cfg.UpstreamDialTimeout,
		IdleConnTimeout:       30 * time.Second,
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   cfg.UpstreamTimeout,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok\n"))
	})
	if register != nil {
		register(mux, Dependencies{
			Agreement:  agreement.NewService(state),
			Responses:  state,
			Profiles:   profiles,
			Matching:   matching.NewService(profiles, state),
			Discovery:  discovery.NewService(state, state),
			Config:     cfg,
			HTTPClient: client,
		})
	}

	server := &http.Server{
		Addr:              cfg.Addr,
		Handler:           observeRequests(mux, logger, cfg.CommandTimeout),
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		IdleTimeout:       cfg.IdleTimeout,
		MaxHeaderBytes:    cfg.MaxHeaderBytes,
	}
	server.RegisterOnShutdown(client.CloseIdleConnections)
	return server, nil
}

func seedDemo(state *storage.InMemoryAgreementStore, profiles *storage.InMemoryMatchingStore) error {
	goSkill := domain.Skill{Name: "go"}
	python := domain.Skill{Name: "python"}
	for _, id := range []string{"author", "mentor"} {
		profiles.AddUser(domain.User{ID: id, Offers: []domain.Skill{goSkill}, Needs: []domain.Skill{python}, Format: domain.FormatOnline})
	}
	profiles.AddUser(domain.User{ID: "learner", Offers: []domain.Skill{python}, Needs: []domain.Skill{goSkill}, Format: domain.FormatOnline})

	for i, author := range []string{"author", "mentor", "mentor", "mentor", "mentor"} {
		err := state.SaveOffer(domain.Offer{
			ID: fmt.Sprintf("offer-%d", i+1), AuthorID: author, Skill: goSkill,
			WantsInReturn: []domain.Skill{python}, Format: domain.FormatOnline, Status: domain.OfferActive,
		})
		if err != nil {
			return err
		}
	}
	return state.SaveResponse(domain.Response{
		ID: "response-1", OfferID: "offer-1", UserID: "learner", Status: domain.ResponsePending,
	})
}
