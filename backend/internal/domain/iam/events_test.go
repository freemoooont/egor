package iam_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/micocards/api/internal/domain/iam"
)

func TestEventNamesArePastTensePackagePrefixed(t *testing.T) {
	cases := []iam.Event{
		iam.UserRegistered{UserID: "u", RegisteredAt: time.Now()},
		iam.UserLoggedIn{UserID: "u", LoggedInAt: time.Now()},
		iam.RefreshTokenIssued{UserID: "u", FamilyID: "f", TokenID: "t"},
		iam.RefreshTokenRevoked{UserID: "u", FamilyID: "f", Reason: "rotation"},
	}
	want := []string{
		"iam.UserRegistered",
		"iam.UserLoggedIn",
		"iam.RefreshTokenIssued",
		"iam.RefreshTokenRevoked",
	}
	for i, ev := range cases {
		require.Equal(t, want[i], ev.Name())
	}
}
