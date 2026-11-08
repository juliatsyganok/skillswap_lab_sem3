package agreement

import (
	"context"
	"errors"
	"fmt"

	"skillswap/domain"
)

var (
	ErrNotParticipant   = errors.New("actor is not an engagement participant")
	ErrOfferMissing     = errors.New("offer is missing in directory")
	ErrOfferUnavailable = errors.New("offer is unavailable in directory")
	ErrDirectoryFailed  = errors.New("offer directory failed")
)

type OfferInfo struct {
	ID     string
	Active bool
}

type OfferDirectory interface {
	OfferInfo(ctx context.Context, offerID string) (OfferInfo, error)
}

type EngagementReader interface {
	GetResponse(id string) (domain.Response, error)
	GetOffer(id string) (domain.Offer, error)
}

type Engagement struct {
	Response domain.Response
	Offer    domain.Offer
}

type Engagements struct {
	reader    EngagementReader
	directory OfferDirectory
	command   *Service
}

func NewEngagements(reader EngagementReader, directory OfferDirectory, command *Service) *Engagements {
	return &Engagements{reader: reader, directory: directory, command: command}
}

func (e *Engagements) Accept(ctx context.Context, actorID, responseID string) (Engagement, bool, error) {
	if err := ctx.Err(); err != nil {
		return Engagement{}, false, err
	}
	current, err := e.load(responseID)
	if err != nil {
		return Engagement{}, false, err
	}
	if current.Offer.AuthorID != actorID {
		return Engagement{}, false, fmt.Errorf("accept response %s: %w", responseID, ErrNotOfferAuthor)
	}

	info, err := e.directory.OfferInfo(ctx, current.Offer.ID)
	switch {
	case err == nil:
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return Engagement{}, false, fmt.Errorf("offer directory %s: %w", current.Offer.ID, err)
	case errors.Is(err, domain.ErrNotFound):
		return Engagement{}, false, fmt.Errorf("offer directory %s: %w: %w", current.Offer.ID, ErrOfferMissing, err)
	default:
		return Engagement{}, false, fmt.Errorf("offer directory %s: %w", current.Offer.ID, err)
	}
	if info.ID != current.Offer.ID {
		return Engagement{}, false, fmt.Errorf("offer directory returned %q for %q: %w", info.ID, current.Offer.ID, ErrDirectoryFailed)
	}
	if !info.Active {
		return Engagement{}, false, fmt.Errorf("offer %s: %w", info.ID, ErrOfferUnavailable)
	}

	resp, changed, err := e.command.AcceptContext(ctx, actorID, responseID)
	if err != nil {
		return Engagement{}, false, err
	}
	current.Response = resp
	return current, changed, nil
}

func (e *Engagements) Get(ctx context.Context, actorID, responseID string) (Engagement, error) {
	if err := ctx.Err(); err != nil {
		return Engagement{}, err
	}
	current, err := e.load(responseID)
	if err != nil {
		return Engagement{}, err
	}
	if actorID != current.Offer.AuthorID && actorID != current.Response.UserID {
		return Engagement{}, fmt.Errorf("read response %s: %w", responseID, ErrNotParticipant)
	}
	return current, nil
}

func (e *Engagements) load(responseID string) (Engagement, error) {
	resp, err := e.reader.GetResponse(responseID)
	if err != nil {
		return Engagement{}, fmt.Errorf("get response %s: %w", responseID, err)
	}
	offer, err := e.reader.GetOffer(resp.OfferID)
	if err != nil {
		return Engagement{}, fmt.Errorf("get offer %s: %w", resp.OfferID, err)
	}
	return Engagement{Response: resp, Offer: offer}, nil
}

type LocalOfferDirectory struct {
	Offers interface {
		GetOffer(id string) (domain.Offer, error)
	}
}

func (d LocalOfferDirectory) OfferInfo(ctx context.Context, offerID string) (OfferInfo, error) {
	if err := ctx.Err(); err != nil {
		return OfferInfo{}, err
	}
	offer, err := d.Offers.GetOffer(offerID)
	if err != nil {
		return OfferInfo{}, err
	}
	return OfferInfo{ID: offer.ID, Active: offer.Status == domain.OfferActive || offer.Status == domain.OfferPaused}, nil
}
