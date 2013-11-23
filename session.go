package dropbox

import (
	"errors"
	"fmt"
	"github.com/garyburd/go-oauth/oauth"
	"net/http"
	"net/url"
)

// Constants to build URLs
const (
	Version = 1
	Scheme  = "https://"
	Prefix  = "/1"

	APIHost     = "api.dropbox.com"
	WWWHost     = "www.dropbox.com"
	ContentHost = "api-content.dropbox.com"

	APIPrefix     = Scheme + APIHost + Prefix
	WWWPrefix     = Scheme + WWWHost + Prefix
	ContentPrefix = Scheme + ContentHost + Prefix
)

// Authorization URLs
const (
	RequestURI       = APIPrefix + "/oauth/request_token"
	AuthorizationURI = WWWPrefix + "/oauth/authorize"
	AccessURI        = APIPrefix + "/oauth/access_token"
)

// An AuthorizationError error represents an error generated while trying to perform OAuth authentication.
type AuthorizationError struct {
	Context string
	Cause   error
}

func (ae *AuthorizationError) Error() string {
	if ae.Cause != nil {
		return fmt.Sprintf("Authorization Error (%s): %v", ae.Context, ae.Cause)
	}
	return fmt.Sprintf("Authorization Error (%s)", ae.Context)
}

// Credentials represents the a set of authentication credentials in the
// Dropbox OAuth v1 API. They are used to sign and authenticate all communication
// with the Dropbox servers.
type Credentials struct {
	Token, Secret string
}

func fromOauth(c *oauth.Credentials) *Credentials {
	return &Credentials{
		Token:  c.Token,
		Secret: c.Secret,
	}
}

func (c *Credentials) oauth() *oauth.Credentials {
	return &oauth.Credentials{
		Token:  c.Token,
		Secret: c.Secret,
	}
}

var oauthClient = oauth.Client{
	TemporaryCredentialRequestURI: RequestURI,
	ResourceOwnerAuthorizationURI: AuthorizationURI,
	TokenRequestURI:               AccessURI,
}

// A Session represents an OAuth connection to dropbox.
type Session struct {
	RequestToken, AccessToken *Credentials
	OauthClient               oauth.Client
	HTTPClient                *http.Client
	Locale                    string
}

func (s *Session) client() *http.Client {
	if s.HTTPClient != nil {
		return s.HTTPClient
	}
	return http.DefaultClient
}

// NewSession creates a new Session object using the given data.
// Both httpClient and accessToken may be nil. In the case of the former, http.DefaultClient
// is used, and in the case of the latter, the full authorization cycle must be performed.
func NewSession(appKey, appSecret string, httpClient *http.Client, accessToken *Credentials) *Session {
	return &Session{
		AccessToken: accessToken,
		HTTPClient:  httpClient,
		OauthClient: oauth.Client{
			TemporaryCredentialRequestURI: RequestURI,
			ResourceOwnerAuthorizationURI: AuthorizationURI,
			TokenRequestURI:               AccessURI,
			Credentials: oauth.Credentials{
				Token:  appKey,
				Secret: appSecret,
			},
		},
	}
}

// Reset the session so that new credentials must be retrieved
// from the Dropbox API server.
func (s *Session) Reset() {
	s.AccessToken = nil
	s.RequestToken = nil
}

// GetRequestToken asks the Dropbox API server for a set of request
// credentials if none exist in the current Session. Upon success
// the credentials are stored in the session and returned. If the
// credentials already exist this method is a no-op.
func (s *Session) GetRequestToken() error {
	if s.RequestToken == nil {
		req, err := s.OauthClient.RequestTemporaryCredentials(s.client(), "", nil)
		if err != nil {
			return err
		}
		s.RequestToken = fromOauth(req)
	}
	return nil
}

func (s *Session) makeParams(locale bool) url.Values {
	p := make(url.Values, 32)
	if locale && s.Locale != "" {
		p.Set("locale", s.Locale)
	}
	return p
}

// GetAuthorizeURL returns a string containing a URL to send the user
// to in a web browser to authorize this Session's request token with
// Dropbox. It's up to the client of this package to determine how to
// do that.
//
// If no request token is present in the Session, a new one will be
// requested, and the potential errors will be returned.
func (s *Session) GetAuthorizeURL(callback string) (string, error) {
	if err := s.GetRequestToken(); err != nil {
		return "", err
	}
	params := s.makeParams(true)
	if callback != "" {
		params.Set("oauth_callback", callback)
	}
	return s.OauthClient.AuthorizationURL(s.RequestToken.oauth(), params), nil
}

// Authorized returns true if the session believes that it has been authorized.
// This simply checks to see if there are access credentials in the Session.
//
// NOTE: The user can still de-authorize a set of access credentials, so this method
// can result in a false positive.
func (s *Session) Authorized() bool {
	return s.AccessToken != nil
}

// GetAccessTokenCallback is to be used by a web-application to handle the return of the
// user from the Dropbox authorization page as long as GetAuthorizeURL was given a callback
// url. The request token from the request and an optional verifier code can be used on
// an initialized session to get an access token. The access credentials produced by the
// dropbox server on success are returned and stored in the session.
func (s *Session) GetAccessTokenCallback(requestToken *Credentials, verifier string) error {
	cred, _, err := s.OauthClient.RequestToken(s.client(), requestToken.oauth(), verifier)
	if err != nil {
		return err
	}
	s.AccessToken = fromOauth(cred)
	return nil
}

// GetAccessToken requests an access token from the server assuming that the request token
// in the session was authorized by the user. If the session is already authorized, (ie:
// there is already an access token), then this method is a no-op.
//
// Should the session be authorized, a copy of the access credentials is returned, as
// well as being stored in the session.
func (s *Session) GetAccessToken() error {
	if s.Authorized() {
		return nil
	}

	if s.RequestToken == nil {
		return errors.New("no request token")
	}

	cred, _, err := s.OauthClient.RequestToken(s.client(), s.RequestToken.oauth(), "")
	if err != nil {
		return err
	}
	s.AccessToken = fromOauth(cred)
	return nil
}

// SignParam adds signing parameters to the params hash given using the Session's
// access token. If the session is not authorized (no access token) this method
// returns an error.
func (s *Session) signParam(method, url string, params url.Values) error {
	if !s.Authorized() {
		return errors.New("session not authorized")
	}

	s.OauthClient.SignParam(s.AccessToken.oauth(), method, url, params)
	return nil
}
