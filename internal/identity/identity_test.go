package identity_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/driif/go-vibe-starter/internal/identity"
)

func TestHasRole(t *testing.T) {
	tests := []struct {
		name string
		p    *identity.Principal
		role string
		want bool
	}{
		{"a nil principal has no role", nil, "admin", false},
		{"a principal without roles has no role", &identity.Principal{}, "admin", false},
		{"a role the principal has", &identity.Principal{Roles: []string{"user", "admin"}}, "admin", true},
		{"a role the principal lacks", &identity.Principal{Roles: []string{"user"}}, "admin", false},
		{"role names are case-sensitive", &identity.Principal{Roles: []string{"Admin"}}, "admin", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.p.HasRole(tt.role))
		})
	}
}

func TestFromContext(t *testing.T) {
	assert.Nil(t, identity.FromContext(context.Background()), "no principal stored")

	p := &identity.Principal{Subject: "s1"}
	assert.Same(t, p, identity.FromContext(identity.WithPrincipal(context.Background(), p)))
}

func TestDevVerifier(t *testing.T) {
	tests := []struct {
		name  string
		roles []string
		token string
	}{
		{"an empty token is accepted", []string{"admin"}, ""},
		{"any token is accepted", []string{"admin", "user"}, "not-a-jwt"},
		{"no roles configured", nil, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := identity.DevVerifier{Roles: tt.roles}

			p, err := v.Verify(context.Background(), tt.token)

			require.NoError(t, err)
			require.NotNil(t, p)
			assert.Equal(t, "local-dev", p.Subject)
			assert.Equal(t, "local-dev", p.Username)
			assert.Empty(t, p.Email)
			assert.Empty(t, p.Name)
			assert.Equal(t, tt.roles, p.Roles)
		})
	}

	t.Run("each call returns its own principal and roles", func(t *testing.T) {
		v := identity.DevVerifier{Roles: []string{"admin"}}

		first, err := v.Verify(context.Background(), "")
		require.NoError(t, err)
		first.Roles[0] = "changed"
		second, err := v.Verify(context.Background(), "")
		require.NoError(t, err)

		assert.NotSame(t, first, second)
		assert.Equal(t, []string{"admin"}, second.Roles)
		assert.Equal(t, []string{"admin"}, v.Roles)
	})
}
