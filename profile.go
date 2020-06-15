package awsauth

import (
	"fmt"
	"io/ioutil"
	"strings"

	"gopkg.in/ini.v1"
)

// Profile ...
type Profile struct {
	Name              string `ini:"-"`
	SSOStartURL       string `ini:"sso_start_url,omitempty"`
	SSORegion         string `ini:"sso_region,omitempty"`
	SSOAccountID      string `ini:"sso_account_id,omitempty"`
	SSORoleName       string `ini:"sso_role_name,omitempty"`
	CredentialProcess string `ini:"credential_process,omitempty"`
}

// IsSSOProfile returns true if the profile uses SSO.
func (p *Profile) IsSSOProfile() bool {
	if p.SSOStartURL != "" {
		return true
	}
	return false
}

// GetAWSConfigProfile ...
func GetAWSConfigProfile(path, profileName string) (*Profile, error) {
	f, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	i, err := ini.LoadSources(ini.LoadOptions{AllowNestedValues: true}, f)
	if err != nil {
		return nil, err
	}
	section, err := i.GetSection(fmt.Sprintf("profile %s", profileName))
	if err != nil {
		if strings.Contains(err.Error(), "does not exist") {
			section, err = i.GetSection(profileName)
		}
		if err != nil {
			return nil, err
		}
	}
	profile := &Profile{Name: profileName}
	if err := section.MapTo(profile); err != nil {
		return nil, fmt.Errorf("parsing profile %q: %s", profileName, err)
	}
	return profile, nil
}

// // ErrInvalidSSOProfile is returned when the SharedConfigProfile is not valid for use with SSO.
// var ErrInvalidSSOProfile = errors.New("invalid sso profile")

// // ResolveCredentialsWithSSO implements resolution for SSO Credentials.
// func ResolveCredentialsWithSSO(cfg *aws.Config, configs external.Configs) error {
// 	_, found, err := external.GetCredentialsProvider(configs)
// 	if err != nil {
// 		return err
// 	}
// 	if found {
// 		// Use external.WithCredentialsProvider if specified.
// 		return external.ResolveCredentials(cfg, configs)
// 	}
// 	profile, found, err := ResolveSSOProfile(configs)
// 	if err != nil {
// 		return err
// 	}
// 	if !found {
// 		// Defer to built-in resolver if SSO is not configured.
// 		return external.ResolveCredentials(cfg, configs)
// 	}
// 	ssoCfg := cfg.Copy()
// 	ssoCfg.Credentials = aws.AnonymousCredentials

// 	store, err := keyring.Open(keyring.Config{
// 		ServiceName:      "aws-auth",
// 		AllowedBackends:  []keyring.BackendType{keyring.BackendType("file")},
// 		FileDir:          "~/.awsauth/keys/",
// 		FilePasswordFunc: fileKeyringPassphrasePrompt,
// 	})
// 	if err != nil {
// 		return fmt.Errorf("open keyring: %v", err)
// 	}

// 	cfg.Credentials = &SSOProvider{
// 		OIDCClient:     ssooidc.New(ssoCfg),
// 		SSOClient:      sso.New(ssoCfg),
// 		Profile:        profile,
// 		Store:          store,
// 		NoBrowser:      false,
// 		RotationWindow: 30 * time.Minute,
// 	}
// 	return nil
// }

// // ResolveSSOProfile attempts to retrive a profile matching the requirements for SSOProfile.
// func ResolveSSOProfile(configs external.Configs) (*SSOProfile, bool, error) {
// 	profileName, found, err := external.GetSharedConfigProfile(configs)
// 	if err != nil {
// 		return nil, false, err
// 	}
// 	if !found {
// 		return nil, false, errors.New("missing aws profile name")
// 	}
// 	files, found, err := external.GetSharedConfigFiles(configs)
// 	if err != nil {
// 		return nil, false, err
// 	}
// 	if !found {
// 		files = []string{external.DefaultSharedConfigFilename()}
// 	}
// 	profile, err := GetSSOProfileFromConfig(profileName, files[0])
// 	if err != nil {
// 		return nil, false, err
// 	}
// 	if len(profile.SSOStartURL) == 0 {
// 		return nil, false, nil
// 	}
// 	return profile, true, nil
// }

// // GetSSOProfileFromConfig reads an SSOProfile from the shared AWS config, which
// // is necessary because the AWS SDK for Go does not support SSO profiles.
// func GetSSOProfileFromConfig(profileName, path string) (*SSOProfile, error) {
// 	f, err := ioutil.ReadFile(path)
// 	if err != nil {
// 		return nil, err
// 	}
// 	i, err := ini.LoadSources(ini.LoadOptions{AllowNestedValues: true}, f)
// 	if err != nil {
// 		return nil, err
// 	}
// 	section, err := i.GetSection(fmt.Sprintf("profile %s", profileName))
// 	if err != nil {
// 		return nil, err
// 	}
// 	profile := &SSOProfile{}
// 	if err := section.MapTo(profile); err != nil {
// 		return nil, fmt.Errorf("parsing profile %q: %s", profileName, err)
// 	}
// 	return profile, nil
// }

// func fileKeyringPassphrasePrompt(prompt string) (string, error) {
// 	fmt.Fprintf(os.Stderr, "%s: ", prompt)
// 	b, err := terminal.ReadPassword(int(os.Stdin.Fd()))
// 	if err != nil {
// 		return "", err
// 	}
// 	fmt.Println()
// 	return string(b), nil
// }
