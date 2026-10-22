package middleware

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHasUserPermissionAllowsDriver(t *testing.T) {
	t.Parallel()

	claims := KeycloakClaims{
		ShareTrip: ShareTripPermissions{
			Permissions: []string{PermissionShareTripDriver},
		},
	}

	require.True(t, claims.HasUserPermission(PermissionShareTripDriver))
	require.False(t, claims.HasUserPermission(PermissionShareTripAdmin))
}

func TestHasUserPermissionAdminCanDoEverything(t *testing.T) {
	t.Parallel()

	claims := KeycloakClaims{
		ShareTrip: ShareTripPermissions{
			Permissions: []string{PermissionShareTripAdmin},
		},
	}

	require.True(t, claims.HasUserPermission(PermissionShareTripDriver))
	require.True(t, claims.HasUserPermission(PermissionShareTripDirector))
	require.True(t, claims.HasUserPermission(PermissionShareTripAdmin))
}

func TestHasUserPermissionAllowsDirector(t *testing.T) {
	t.Parallel()

	claims := KeycloakClaims{
		ShareTrip: ShareTripPermissions{
			Permissions: []string{PermissionShareTripDirector},
		},
	}

	require.True(t, claims.HasUserPermission(PermissionShareTripDirector))
	require.False(t, claims.HasUserPermission(PermissionShareTripDriver))
}

func TestHasUserPermissionAllowsPassenger(t *testing.T) {
	t.Parallel()

	claims := KeycloakClaims{
		ShareTrip: ShareTripPermissions{
			Permissions: []string{PermissionShareTripPassenger},
		},
	}

	require.True(t, claims.HasUserPermission(PermissionShareTripPassenger))
	require.False(t, claims.HasUserPermission(PermissionShareTripDriver))
}

func TestHasExactPermissionDoesNotBypassAdmin(t *testing.T) {
	t.Parallel()

	claims := KeycloakClaims{
		ShareTrip: ShareTripPermissions{
			Permissions: []string{PermissionShareTripAdmin},
		},
	}

	require.True(t, claims.HasExactPermission(PermissionShareTripAdmin))
	require.False(t, claims.HasExactPermission(PermissionShareTripDriver))
	require.False(t, claims.HasExactPermission(PermissionShareTripPassenger))
}
