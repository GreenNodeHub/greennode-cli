package auth

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

// UserAgent identifies IAM token requests.
var UserAgent = "grn-cli"

// MachineTokenProvider uses IAM v2 client_credentials; access tokens stay in memory.
type MachineTokenProvider struct {
	cfg        clientcredentials.Config
	baseCtx    context.Context
	httpClient *http.Client

	mu          sync.Mutex
	accessToken string
	expiresAt   time.Time
}

// NewMachineTokenProvider uses the caller's IAM v2 endpoint.
func NewMachineTokenProvider(clientID, clientSecret, tokenURL string) *MachineTokenProvider {
	return &MachineTokenProvider{
		baseCtx:    context.Background(),
		httpClient: newTokenHTTPClient(30 * time.Second),
		cfg: clientcredentials.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			TokenURL:     tokenURL,
			AuthStyle:    oauth2.AuthStyleInHeader,
		},
	}
}

// GetToken returns a cached token or refreshes it before expiry.
func (p *MachineTokenProvider) GetToken() (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.accessToken != "" && time.Now().Before(p.expiresAt) {
		return p.accessToken, nil
	}
	return p.mint()
}

// RefreshToken mints a token regardless of cache state.
func (p *MachineTokenProvider) RefreshToken() (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.mint()
}

// SetToken supplies a cached token for tests.
func (p *MachineTokenProvider) SetToken(token string, expiresAt time.Time) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.accessToken = token
	p.expiresAt = expiresAt
}

// mint fetches a new access token via the client_credentials grant. Caller holds p.mu.
func (p *MachineTokenProvider) mint() (string, error) {
	tok, err := p.cfg.Token(context.WithValue(p.baseCtx, oauth2.HTTPClient, p.httpClient))
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return "", context.Canceled
		}
		if errors.Is(err, context.DeadlineExceeded) {
			return "", context.DeadlineExceeded
		}
		return "", errors.New("IAM machine token request failed")
	}
	p.accessToken = tok.AccessToken
	exp := tok.Expiry
	if exp.IsZero() {
		exp = time.Now().Add(noExpiryFallback)
	}
	p.expiresAt = exp.Add(-refreshExpirySkew)
	return p.accessToken, nil
}

// SetBaseContext supplies cancellation for subsequent IAM requests.
func (p *MachineTokenProvider) SetBaseContext(ctx context.Context) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if ctx == nil {
		ctx = context.Background()
	}
	p.baseCtx = ctx
}

// SetHTTPTimeout sets the total IAM request timeout; zero disables it.
func (p *MachineTokenProvider) SetHTTPTimeout(timeout time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.httpClient = newTokenHTTPClient(timeout)
}

func newTokenHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout:   timeout,
		Transport: userAgentTransport{base: http.DefaultTransport},
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

type userAgentTransport struct {
	base http.RoundTripper
}

func (t userAgentTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	request := req.Clone(req.Context())
	request.Header.Set("User-Agent", UserAgent)
	return t.base.RoundTrip(request)
}

// Check the consumer's token-provider contract without an import cycle.
var _ interface {
	GetToken() (string, error)
	RefreshToken() (string, error)
} = (*MachineTokenProvider)(nil)
