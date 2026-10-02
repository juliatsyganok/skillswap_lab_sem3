package domain

type Level string

const (
	LevelBeginner     Level = "beginner"
	LevelIntermediate Level = "intermediate"
	LevelAdvanced     Level = "advanced"
)

type Format string

const (
	FormatOnline  Format = "online"
	FormatOffline Format = "offline"
	FormatAny     Format = "any"
)

type Skill struct {
	Name  string
	Level Level
}

type User struct {
	ID     string
	Offers []Skill
	Needs  []Skill
	Format Format
}

type OfferStatus string

const (
	OfferActive OfferStatus = "active"
	OfferPaused OfferStatus = "paused"
	OfferClosed OfferStatus = "closed"
)

type Offer struct {
	ID            string
	AuthorID      string
	Skill         Skill
	WantsInReturn []Skill
	Format        Format
	Status        OfferStatus
}

type ResponseStatus string

const (
	ResponsePending  ResponseStatus = "pending"
	ResponseAccepted ResponseStatus = "accepted"
	ResponseRejected ResponseStatus = "rejected"
)

type Response struct {
	ID      string
	OfferID string
	UserID  string
	Status  ResponseStatus
}
