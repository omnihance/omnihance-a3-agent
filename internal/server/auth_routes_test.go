package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/omnihance/omnihance-a3-agent/internal/constants"
	"github.com/omnihance/omnihance-a3-agent/internal/logger"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

func TestConcurrentFirstSignupsCreateOnlyOneSuperAdmin(t *testing.T) {
	internalDB := newTestInternalDB(t)
	server := &Server{
		internalDB: internalDB,
		log:        logger.NewZerologLogger(zerolog.Nop(), "test", zerolog.Disabled),
	}

	emails := []string{"first@example.com", "second@example.com"}
	var wg sync.WaitGroup
	errs := make(chan error, len(emails))
	for _, email := range emails {
		wg.Add(1)
		go func(email string) {
			defer wg.Done()

			body, err := json.Marshal(AuthRequest{Email: email, Password: "secret1"})
			if err != nil {
				errs <- err
				return
			}

			req := httptest.NewRequest(http.MethodPost, "/api/auth/sign-up", bytes.NewReader(body))
			rr := httptest.NewRecorder()
			server.signUpHandler(rr, req)
			if rr.Code != http.StatusOK {
				errs <- fmt.Errorf("signup %s returned %d: %s", email, rr.Code, rr.Body.String())
			}
		}(email)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}

	users, err := internalDB.GetUsers()
	require.NoError(t, err)
	require.Len(t, users, 2)

	superAdminCount := 0
	activeSuperAdminCount := 0
	for _, user := range users {
		if user.Roles == constants.RoleSuperAdmin {
			superAdminCount++
			if user.Status == constants.UserStatusActive {
				activeSuperAdminCount++
			}
		}
	}

	require.Equal(t, 1, superAdminCount)
	require.Equal(t, 1, activeSuperAdminCount)
}
