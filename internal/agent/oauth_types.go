package agent

type oauthCredentials struct {
	RefreshToken  string
	AccessToken   string
	ExpiresAtMS   int64
	EnterpriseURL string
	ProjectID     string
	Email         string
}

func (c oauthCredentials) expired(nowMS int64) bool {
	if c.ExpiresAtMS == 0 {
		return true
	}
	return nowMS >= c.ExpiresAtMS
}
