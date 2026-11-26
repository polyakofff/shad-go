//go:build !solution

package retryupdate

import (
	"errors"

	"github.com/gofrs/uuid"
	"gitlab.com/slon/shad-go/retryupdate/kvapi"
)

func UpdateValue(c kvapi.Client, key string, updateFn func(oldValue *string) (newValue string, err error)) error {
BeforeGet:
	for {
		getReq := kvapi.GetRequest{
			Key: key,
		}
		getResp, err := c.Get(&getReq)

		if err != nil {
			var authErr *kvapi.AuthError
			switch {
			case errors.As(err, &authErr):
				return err
			case errors.Is(err, kvapi.ErrKeyNotFound):
				getResp = nil
			default:
				continue BeforeGet
			}
		}

		var oldValue *string
		var oldVersion uuid.UUID
		if getResp != nil {
			oldValue = &getResp.Value
			oldVersion = getResp.Version
		}

	BeforeSet:
		newValue, err := updateFn(oldValue)

		if err != nil {
			return err
		}

		var prevVersion uuid.UUID
		for {
			setReq := kvapi.SetRequest{
				Key:        key,
				Value:      newValue,
				OldVersion: oldVersion,
				NewVersion: uuid.Must(uuid.NewV4()),
			}

			_, err = c.Set(&setReq)

			if err != nil {
				var authErr *kvapi.AuthError
				var conflictErr *kvapi.ConflictError
				switch {
				case errors.As(err, &authErr):
					return err
				case errors.Is(err, kvapi.ErrKeyNotFound):
					oldValue = nil
					oldVersion = uuid.UUID{}
					goto BeforeSet
				case errors.As(err, &conflictErr):
					if conflictErr.ExpectedVersion == prevVersion {
						return nil
					}
					continue BeforeGet
				default:
					prevVersion = setReq.NewVersion
					continue
				}
			}

			return nil
		}
	}

}
