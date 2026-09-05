package services

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/oauth2"

	"novelhub/internal/dtos/response"
	"novelhub/internal/gen/sqlc"
	"novelhub/internal/models"
	"novelhub/pkg/apperrors"
	"novelhub/pkg/constants"
	"novelhub/pkg/convert"
	"novelhub/pkg/jsonx"
	"novelhub/pkg/netx"
	"novelhub/pkg/oauth"
)

func (a *authService) SigninOrRegisterOAuth(ctx context.Context, provider string, email string, name string, avatarURL string, oauth2ID string) (*response.AuthResponse, error) {
	if email == "" {
		return nil, apperrors.New(apperrors.ErrBadRequest, "OAuth email is required")
	}

	user, err := a.userRepo.GetAuthByEmail(ctx, email)
	if err != nil && !apperrors.IsNotFound(err) {
		return nil, apperrors.New(apperrors.ErrInternalError, "Internal Server Error")
	}

	if user != nil {
		if slices.Contains(models.RolesEntityToRoleConstant(user.Roles), constants.RoleTypeBanned) {
			return nil, apperrors.New(apperrors.ErrForbidden, "User account is banned")
		}

		if user.AuthProvider != "" && !strings.EqualFold(user.AuthProvider, provider) {
			return nil, apperrors.New(apperrors.ErrConflict, fmt.Sprintf("Account was registered with %s authentication", user.AuthProvider))
		}
		if user.Oauth2ID != "" && oauth2ID != "" && user.Oauth2ID != oauth2ID {
			return nil, apperrors.New(apperrors.ErrForbidden, "OAuth account ID mismatch")
		}

		var newName *string
		if name != "" && user.FullName != name {
			newName = &name
		}
		var newAvatar *string
		if avatarURL != "" && user.AvatarUrl != avatarURL {
			newAvatar = &avatarURL
		}
		var newOAuth2ID *string
		if oauth2ID != "" && user.Oauth2ID != oauth2ID {
			newOAuth2ID = &oauth2ID
		}

		if newName != nil || newAvatar != nil || newOAuth2ID != nil {
			params := sqlc.UpdateProfileParams{
				ID: user.ID,
			}
			if newName != nil {
				params.FullName = convert.StrPtrToNullString(newName)
			}
			if newAvatar != nil {
				params.AvatarUrl = convert.StrPtrToNullString(newAvatar)
			}
			if newOAuth2ID != nil {
				params.Oauth2ID = convert.StrPtrToNullString(newOAuth2ID)
			}

			updated, err := a.userRepo.UpdateProfile(ctx, params)
			if err == nil && updated != nil {
				user.FullName = updated.FullName
				user.AvatarUrl = updated.AvatarUrl
				user.Oauth2ID = updated.Oauth2ID
			}
		}

		tokens, err := a.genToken(user)
		if err != nil {
			return nil, err
		}

		refreshDigest := refreshTokenDigest(tokens.RefreshToken)
		if err := a.userRepo.UpdateRefreshToken(ctx, user.ID, &refreshDigest); err != nil {
			return nil, apperrors.New(apperrors.ErrInternalError, "Failed to update refresh token")
		}

		a.userRepo.InvalidateUserCache(ctx, user.ID, user.Email)
		return tokens, nil
	}

	settings, err := a.settings.Public(ctx)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to load settings")
	}
	if !settings.RegistrationEnabled {
		return nil, apperrors.New(apperrors.ErrForbidden, "Public registration is disabled")
	}

	autoIDs, err := a.roleRepo.GetAutoAssignRoleIDs(ctx)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to get auto roles")
	}
	var roles []*models.RoleEntity
	if len(autoIDs) > 0 {
		roles, err = a.roleRepo.GetByIDs(ctx, autoIDs)
		if err != nil {
			return nil, apperrors.New(apperrors.ErrInternalError, "Failed to fetch roles")
		}
	} else {
		role, err := a.roleRepo.GetByName(ctx, constants.RoleTypeUser.String())
		if err != nil {
			return nil, apperrors.New(apperrors.ErrInternalError, "Failed to get default role")
		}
		roles = []*models.RoleEntity{role}
	}

	tx, err := a.txManager.BeginTx(ctx, nil)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to start transaction")
	}
	defer func() {
		_ = tx.Rollback()
	}()

	userRepoTx := a.userRepo.WithTx(tx)
	roleRepoTx := a.roleRepo.WithTx(tx)

	var fullNamePtr *string
	if name != "" {
		fullNamePtr = &name
	}
	var avatarPtr *string
	if avatarURL != "" {
		avatarPtr = &avatarURL
	}
	var oauth2IDPtr *string
	if oauth2ID != "" {
		oauth2IDPtr = &oauth2ID
	}

	newUser, err := userRepoTx.UpsertUser(ctx, sqlc.UpsertUserParams{
		ID:           uuid.Must(uuid.NewV7()).String(),
		Email:        email,
		AuthProvider: provider,
		Oauth2ID:     convert.StrPtrToNullString(oauth2IDPtr),
		FullName:     convert.StrPtrToNullString(fullNamePtr),
		AvatarUrl:    convert.StrPtrToNullString(avatarPtr),
	})
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to create user")
	}

	newUser.Roles = make([]*models.RoleSimple, 0, len(roles))
	for _, role := range roles {
		if err := roleRepoTx.CreateUserRole(ctx, newUser.ID, role.ID); err != nil {
			return nil, apperrors.New(apperrors.ErrInternalError, "Failed to assign roles")
		}
		newUser.Roles = append(newUser.Roles, role.ToRoleSimple())
	}

	if err := tx.Commit(); err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to commit user registration")
	}

	tokens, err := a.genToken(newUser)
	if err != nil {
		return nil, err
	}

	refreshDigest := refreshTokenDigest(tokens.RefreshToken)
	if err := a.userRepo.UpdateRefreshToken(ctx, newUser.ID, &refreshDigest); err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to update refresh token")
	}

	return tokens, nil
}

type OAuthState struct {
	State       string `json:"state"`
	RedirectURL string `json:"redirect_url"`
}

type OIDCDiscoveryDoc struct {
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	UserinfoEndpoint      string `json:"userinfo_endpoint"`
	Issuer                string `json:"issuer"`
}

func sanitizeRedirectURL(target string) string {
	target = strings.TrimSpace(target)
	if target == "" {
		return "/"
	}
	if !strings.HasPrefix(target, "/") || strings.HasPrefix(target, "//") || strings.HasPrefix(target, "/\\") || strings.HasPrefix(target, "\\") {
		return "/"
	}
	if strings.ContainsAny(target, "\r\n\t") || strings.Contains(target, "://") {
		return "/"
	}
	parsed, err := url.Parse(target)
	if err != nil || parsed.IsAbs() || parsed.Host != "" {
		return "/"
	}
	return target
}

func (a *authService) BuildOAuthURL(ctx context.Context, provider string, redirect string) (authURL string, stateUUID string, err error) {
	provider = strings.ToLower(provider)
	if provider != "google" && provider != "github" && provider != "discord" && provider != "oidc" {
		return "", "", apperrors.New(apperrors.ErrBadRequest, "Unsupported OAuth provider")
	}

	config, err := a.settings.OAuthProviderConfig(ctx, provider)
	if err != nil || config == nil || !config.Enabled {
		return "", "", apperrors.New(apperrors.ErrForbidden, "Provider is disabled or misconfigured")
	}

	if config.ClientID == "" || config.RedirectURI == "" {
		return "", "", apperrors.New(apperrors.ErrBadRequest, "Provider Client ID or Redirect URI is not configured")
	}

	stateUUID = uuid.New().String()
	redirect = sanitizeRedirectURL(redirect)

	stateData := OAuthState{
		State:       stateUUID,
		RedirectURL: redirect,
	}

	stateBytes, err := jsonx.Marshal(stateData)
	if err != nil {
		return "", "", apperrors.New(apperrors.ErrInternalError, "Failed to generate state")
	}
	encodedState := base64.URLEncoding.EncodeToString(stateBytes)

	var cfg *oauth2.Config

	switch provider {
	case "google":
		cfg = oauth.NewGoogleProvider(config.ClientID, config.ClientSecret, config.RedirectURI)
	case "github":
		cfg = oauth.NewGithubProvider(config.ClientID, config.ClientSecret, config.RedirectURI)
	case "discord":
		cfg = oauth.NewDiscordProvider(config.ClientID, config.ClientSecret, config.RedirectURI)
	case "oidc":
		if config.IssuerURL == "" {
			return "", "", apperrors.New(apperrors.ErrBadRequest, "OIDC Issuer URL is required")
		}
		// Discover endpoints dynamically using SSRF-safe client
		discoveryURL := strings.TrimSuffix(config.IssuerURL, "/") + "/.well-known/openid-configuration"
		discReq, err := http.NewRequestWithContext(ctx, http.MethodGet, discoveryURL, nil)
		if err != nil {
			return "", "", apperrors.New(apperrors.ErrInternalError, "Failed to create OIDC discovery request")
		}

		client := netx.NewSafeHTTPClient(10 * time.Second)
		resp, err := client.Do(discReq)
		if err != nil || resp.StatusCode != http.StatusOK {
			return "", "", apperrors.New(apperrors.ErrInternalError, "Failed to fetch OIDC configuration discovery")
		}
		defer resp.Body.Close()

		var doc OIDCDiscoveryDoc
		docBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", "", apperrors.New(apperrors.ErrInternalError, "Failed to read OIDC configuration discovery")
		}
		if err := jsonx.Unmarshal(docBytes, &doc); err != nil {
			return "", "", apperrors.New(apperrors.ErrInternalError, "Failed to parse OIDC configuration discovery")
		}

		cfg = oauth.NewOidcProvider(config.ClientID, config.ClientSecret, config.RedirectURI, doc.AuthorizationEndpoint, doc.TokenEndpoint, config.Scopes)
	}

	var authOpts []oauth2.AuthCodeOption
	if provider == "google" {
		authOpts = append(authOpts, oauth2.SetAuthURLParam("prompt", "select_account"))
	}

	authURL = cfg.AuthCodeURL(encodedState, authOpts...)
	return authURL, stateUUID, nil
}

func (a *authService) HandleOAuthCallback(ctx context.Context, provider string, code string, stateParam string, cookieState string) (*response.OAuthCallbackResponse, error) {
	provider = strings.ToLower(provider)
	if provider != "google" && provider != "github" && provider != "discord" && provider != "oidc" {
		return nil, apperrors.New(apperrors.ErrBadRequest, "Unsupported OAuth provider")
	}

	stateBytes, err := base64.URLEncoding.DecodeString(stateParam)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrBadRequest, "Invalid state payload")
	}

	var stateData OAuthState
	if err := jsonx.Unmarshal(stateBytes, &stateData); err != nil {
		return nil, apperrors.New(apperrors.ErrBadRequest, "Invalid state structure")
	}

	if cookieState == "" || cookieState != stateData.State {
		return nil, apperrors.New(apperrors.ErrForbidden, "Security check failed (state mismatch)")
	}

	if code == "" {
		return nil, apperrors.New(apperrors.ErrBadRequest, "Authorization code was not returned by provider")
	}

	config, err := a.settings.OAuthProviderConfig(ctx, provider)
	if err != nil || config == nil || !config.Enabled {
		return nil, apperrors.New(apperrors.ErrForbidden, "Provider is disabled or misconfigured")
	}

	// Configure HTTP client for OAuth2 exchange to use the SSRF-safe client
	safeClient := netx.NewSafeHTTPClient(15 * time.Second)
	ctx = context.WithValue(ctx, oauth2.HTTPClient, safeClient)

	var cfg *oauth2.Config
	var userinfoURL string

	switch provider {
	case "google":
		cfg = oauth.NewGoogleProvider(config.ClientID, config.ClientSecret, config.RedirectURI)
		userinfoURL = "https://www.googleapis.com/oauth2/v3/userinfo"
	case "github":
		cfg = oauth.NewGithubProvider(config.ClientID, config.ClientSecret, config.RedirectURI)
		userinfoURL = "https://api.github.com/user"
	case "discord":
		cfg = oauth.NewDiscordProvider(config.ClientID, config.ClientSecret, config.RedirectURI)
		userinfoURL = "https://discord.com/api/users/@me"
	case "oidc":
		discoveryURL := strings.TrimSuffix(config.IssuerURL, "/") + "/.well-known/openid-configuration"
		discReq, err := http.NewRequestWithContext(ctx, http.MethodGet, discoveryURL, nil)
		if err != nil {
			return nil, apperrors.New(apperrors.ErrInternalError, "Failed to create OIDC configuration request")
		}

		resp, err := safeClient.Do(discReq)
		if err != nil || resp.StatusCode != http.StatusOK {
			return nil, apperrors.New(apperrors.ErrInternalError, "Failed to fetch OIDC configuration discovery")
		}
		defer resp.Body.Close()

		var doc OIDCDiscoveryDoc
		docBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, apperrors.New(apperrors.ErrInternalError, "Failed to read OIDC configuration discovery")
		}
		if err := jsonx.Unmarshal(docBytes, &doc); err != nil {
			return nil, apperrors.New(apperrors.ErrInternalError, "Failed to parse OIDC configuration discovery")
		}

		cfg = oauth.NewOidcProvider(config.ClientID, config.ClientSecret, config.RedirectURI, doc.AuthorizationEndpoint, doc.TokenEndpoint, config.Scopes)
		userinfoURL = doc.UserinfoEndpoint
	}

	token, err := cfg.Exchange(ctx, code)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrBadRequest, "Failed to exchange authorization code: "+err.Error())
	}

	userReq, err := http.NewRequestWithContext(ctx, http.MethodGet, userinfoURL, nil)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to build user profile request")
	}
	userReq.Header.Set("Authorization", "Bearer "+token.AccessToken)

	userResp, err := safeClient.Do(userReq)
	if err != nil || userResp.StatusCode != http.StatusOK {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to fetch user profile details")
	}
	defer userResp.Body.Close()

	var email, name, avatarURL, oauth2ID string

	switch provider {
	case "google":
		var googleProfile struct {
			Sub     string `json:"sub"`
			Email   string `json:"email"`
			Name    string `json:"name"`
			Picture string `json:"picture"`
		}
		profileBytes, err := io.ReadAll(userResp.Body)
		if err != nil {
			return nil, apperrors.New(apperrors.ErrInternalError, "Failed to read Google user profile")
		}
		if err := jsonx.Unmarshal(profileBytes, &googleProfile); err != nil {
			return nil, apperrors.New(apperrors.ErrInternalError, "Failed to decode Google user profile")
		}
		email = googleProfile.Email
		name = googleProfile.Name
		avatarURL = googleProfile.Picture
		oauth2ID = googleProfile.Sub

	case "github":
		var githubProfile struct {
			ID        int64  `json:"id"`
			Login     string `json:"login"`
			Name      string `json:"name"`
			AvatarURL string `json:"avatar_url"`
			Email     string `json:"email"`
		}
		profileBytes, err := io.ReadAll(userResp.Body)
		if err != nil {
			return nil, apperrors.New(apperrors.ErrInternalError, "Failed to read GitHub user profile")
		}
		if err := jsonx.Unmarshal(profileBytes, &githubProfile); err != nil {
			return nil, apperrors.New(apperrors.ErrInternalError, "Failed to decode GitHub user profile")
		}
		email = githubProfile.Email
		name = githubProfile.Name
		if name == "" {
			name = githubProfile.Login
		}
		avatarURL = githubProfile.AvatarURL
		oauth2ID = strconv.FormatInt(githubProfile.ID, 10)

		if email == "" {
			emailsReq, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/user/emails", nil)
			if err == nil {
				emailsReq.Header.Set("Authorization", "Bearer "+token.AccessToken)
				emailsResp, err := safeClient.Do(emailsReq)
				if err == nil && emailsResp.StatusCode == http.StatusOK {
					defer emailsResp.Body.Close()
					var githubEmails []struct {
						Email    string `json:"email"`
						Primary  bool   `json:"primary"`
						Verified bool   `json:"verified"`
					}
					emailsBytes, err := io.ReadAll(emailsResp.Body)
					if err == nil {
						if err := jsonx.Unmarshal(emailsBytes, &githubEmails); err == nil {
							for _, gEmail := range githubEmails {
								if gEmail.Verified && gEmail.Primary {
									email = gEmail.Email
									break
								}
							}
							if email == "" {
								for _, gEmail := range githubEmails {
									if gEmail.Verified {
										email = gEmail.Email
										break
									}
								}
							}
						}
					}
				}
			}
		}

	case "discord":
		var discordProfile struct {
			ID            string `json:"id"`
			Username      string `json:"username"`
			Discriminator string `json:"discriminator"`
			Avatar        string `json:"avatar"`
			Email         string `json:"email"`
		}
		profileBytes, err := io.ReadAll(userResp.Body)
		if err != nil {
			return nil, apperrors.New(apperrors.ErrInternalError, "Failed to read Discord user profile")
		}
		if err := jsonx.Unmarshal(profileBytes, &discordProfile); err != nil {
			return nil, apperrors.New(apperrors.ErrInternalError, "Failed to decode Discord user profile")
		}
		email = discordProfile.Email
		name = discordProfile.Username
		if discordProfile.Avatar != "" {
			avatarURL = fmt.Sprintf("https://cdn.discordapp.com/avatars/%s/%s.png", discordProfile.ID, discordProfile.Avatar)
		}
		oauth2ID = discordProfile.ID

	case "oidc":
		var oidcProfile struct {
			Sub               string `json:"sub"`
			Email             string `json:"email"`
			Name              string `json:"name"`
			PreferredUsername string `json:"preferred_username"`
			Picture           string `json:"picture"`
		}
		profileBytes, err := io.ReadAll(userResp.Body)
		if err != nil {
			return nil, apperrors.New(apperrors.ErrInternalError, "Failed to read OIDC user profile")
		}
		if err := jsonx.Unmarshal(profileBytes, &oidcProfile); err != nil {
			return nil, apperrors.New(apperrors.ErrInternalError, "Failed to decode OIDC user profile")
		}
		email = oidcProfile.Email
		name = oidcProfile.Name
		if name == "" {
			name = oidcProfile.PreferredUsername
		}
		if name == "" {
			name = oidcProfile.Sub
		}
		avatarURL = oidcProfile.Picture
		oauth2ID = oidcProfile.Sub
	}

	if email == "" {
		return nil, apperrors.New(apperrors.ErrBadRequest, "Could not retrieve a verified email address from the OAuth provider")
	}

	res, signinErr := a.SigninOrRegisterOAuth(ctx, strings.ToUpper(provider), email, name, avatarURL, oauth2ID)
	if signinErr != nil {
		return nil, signinErr
	}

	return &response.OAuthCallbackResponse{
		AccessToken:  res.AccessToken,
		RefreshToken: res.RefreshToken,
		RedirectURL:  sanitizeRedirectURL(stateData.RedirectURL),
	}, nil
}
