package awsauth

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/99designs/keyring"
	"github.com/aws/aws-sdk-go-v2/aws"
)

// CacheProvider wraps a aws.CredentialsProvider and caches the sessions.
type CacheProvider struct {
	Provider       aws.CredentialsProvider
	Store          keyring.Keyring
	ProfileName    string
	RotationWindow time.Duration
}

// Retrieve implements aws.CredentialProvider.
func (p *CacheProvider) Retrieve(ctx context.Context) (creds aws.Credentials, err error) {
	item, err := p.Store.Get(p.ProfileName)
	if err != nil && err != keyring.ErrKeyNotFound {
		return aws.Credentials{}, err
	}

	if item.Data != nil {
		if err = json.Unmarshal(item.Data, &creds); err != nil {
			return aws.Credentials{}, fmt.Errorf("invalid data in keyring: %v", err)
		}
	}

	if creds.Expires.Add(-p.RotationWindow).After(time.Now()) {
		// Credentials are still valid and can be used
		return creds, nil
	}

	creds, err = p.Provider.Retrieve(ctx)
	if err != nil {
		return aws.Credentials{}, err
	}

	bytes, err := json.Marshal(creds)
	if err != nil {
		return aws.Credentials{}, err
	}
	err = p.Store.Set(keyring.Item{
		Key:                         p.ProfileName,
		Label:                       fmt.Sprintf("aws-auth cache for %s", p.ProfileName),
		Data:                        bytes,
		KeychainNotTrustApplication: true,
	})
	if err != nil {
		return aws.Credentials{}, err
	}
	return creds, nil
}
