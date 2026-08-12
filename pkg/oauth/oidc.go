package oauth

import "golang.org/x/oauth2"

func NewOidcProvider(clientID, clientSecret, redirectURL string, authURL, tokenURL string, scopes []string) *oauth2.Config {
	if len(scopes) == 0 {
		scopes = []string{"openid", "profile", "email"}
	}
	return &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:  authURL,
			TokenURL: tokenURL,
		},
	}
}
