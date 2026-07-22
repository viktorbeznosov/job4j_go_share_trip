package validators

import (
	"github.com/google/uuid"
)

func IsValidUUID(s string) bool {
    id, err := uuid.Parse(s)
    if err != nil {
        return false
    }

    return id != uuid.Nil
}


