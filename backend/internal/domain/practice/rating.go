package practice

// CardRating mirrors the smallint codes used over the wire.
type CardRating int16

// CardRating values per spec.
const (
	RatingDontKnow      CardRating = 0
	RatingStillLearning CardRating = 1
	RatingKnowKnow      CardRating = 2
)

// IsValid reports whether the rating is one of the documented values.
func (r CardRating) IsValid() bool {
	return r == RatingDontKnow || r == RatingStillLearning || r == RatingKnowKnow
}

// SessionMode mirrors the documented practice modes.
type SessionMode string

// SessionMode values per spec.
const (
	ModeTracked   SessionMode = "tracked"
	ModeUntracked SessionMode = "untracked"
)

// IsValid reports whether the mode is one of the documented values.
func (m SessionMode) IsValid() bool {
	return m == ModeTracked || m == ModeUntracked
}

// SessionStatus mirrors the documented states.
type SessionStatus string

// SessionStatus values per spec.
const (
	StatusInProgress SessionStatus = "in_progress"
	StatusCompleted  SessionStatus = "completed"
	StatusAbandoned  SessionStatus = "abandoned"
)
