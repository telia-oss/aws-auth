package awsauth

// // ResolverFunc is the function singature used by AWS resolver functions.
// type ResolverFunc func(*aws.Config, external.Configs) error

// // LoadWithSSOResolver loads the AWS Config with support for SSO Credentials.
// func LoadWithSSOResolver(configs ...external.Config) (aws.Config, error) {
// 	var cfgs external.Configs
// 	cfgs = append(cfgs, configs...)

// 	cfgs, err := cfgs.AppendFromLoaders(external.DefaultConfigLoaders)
// 	if err != nil {
// 		return aws.Config{}, err
// 	}

// 	resolvers := []external.AWSConfigResolver{
// 		external.ResolveDefaultAWSConfig,
// 		external.ResolveHandlersFunc,
// 		external.ResolveEndpointResolverFunc,
// 		external.ResolveCustomCABundle,
// 		external.ResolveEnableEndpointDiscovery,

// 		external.ResolveRegion,
// 		external.ResolveEC2Region,
// 		external.ResolveDefaultRegion,

// 		ResolveCredentialsWithSSO,
// 	}
// 	return cfgs.ResolveAWSConfig(resolvers)
// }

// // NewSSOResolverWithFallback providers a resolver for SSO Credentials with a fallback.
// func NewSSOResolverWithFallback(p *SSOProvider, fallbackResolver ResolverFunc) ResolverFunc {
// 	ResolveSSOCredentials := NewSSOResolver(p)

// 	return func(cfg *aws.Config, configs external.Configs) error {
// 		err := ResolveSSOCredentials(cfg, configs)

// 		// Use the fallback resolver if the SSO Profile is not valid.
// 		if err != nil && err == ErrInvalidSSOProfile {
// 			return fallbackResolver(cfg, configs)
// 		}

// 		return err
// 	}
// }

// // NewSSOResolver returns a resolver that can be passed to e.g. external.LoadDefaultAWSConfig() using external.WithCredentialsProvider.
// func NewSSOResolver(p *SSOProvider) ResolverFunc {
// 	return func(cfg *aws.Config, configs external.Configs) error {
// 		profile, found, err := ResolveSSOProfile(configs)
// 		if err != nil {
// 			return err
// 		}
// 		if !found {
// 			return errors.New("invalid sso profile configuration")
// 		}

// 		p.Profile = profile

// 		config := defaults.Config()
// 		config.Region = profile.SSORegion

// 		if p.OIDCClient == nil {
// 			p.OIDCClient = ssooidc.New(config)
// 		}
// 		if p.SSOClient == nil {
// 			p.SSOClient = sso.New(config)
// 		}
// 		if p.Store == nil {
// 			return errors.New("missing store")
// 		}
// 		if p.RotationWindow == 0 {
// 			p.RotationWindow = 30 * time.Minute
// 		}

// 		cfg.Credentials = p
// 		return nil
// 	}
// }
