package aumgr

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	EpicOAuthHost        = "account-public-service-prod03.ol.epicgames.com"
	LauncherClientId     = "34a02cf8f4414e29b15921876da36f9a"
	LauncherClientSecret = "daafbccc737745039dffe53d94fc76cf"
	EpicUserAgent        = "UELauncher/11.0.1-14469634+++Portal+Release-Live Windows/10.0.19041.1.256.64bit"
)

type EpicSession struct {
	AccessToken  string    `json:"access_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	RefreshToken string    `json:"refresh_token"`
	RefreshAt    time.Time `json:"refresh_at"`
	AccountId    string    `json:"account_id"`
	DisplayName  string    `json:"display_name"`
}

type EpicApi struct {
	client *http.Client
}

func NewEpicApi() *EpicApi {
	return &EpicApi{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (a *EpicApi) GetAuthUrl() string {
	return fmt.Sprintf("https://www.epicgames.com/id/login?redirectUrl=https%%3A%%2F%%2Fwww.epicgames.com%%2Fid%%2Fapi%%2Fredirect%%3FclientId%%3D%s%%26responseType%%3Dcode", LauncherClientId)
}

func (a *EpicApi) getBasicAuth() string {
	auth := LauncherClientId + ":" + LauncherClientSecret
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))
}

type oauthTokenResponse struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	RefreshIn    int    `json:"refresh_in"`
	AccountId    string `json:"account_id"`
	DisplayName  string `json:"display_name"`
}

func (a *EpicApi) LoginWithAuthCode(code string) (*EpicSession, error) {
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)

	return a.oauthRequest(data)
}

func (a *EpicApi) RefreshSession(refreshToken string) (*EpicSession, error) {
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", refreshToken)

	return a.oauthRequest(data)
}

func (a *EpicApi) oauthRequest(data url.Values) (*EpicSession, error) {
	req, err := http.NewRequest("POST", "https://"+EpicOAuthHost+"/account/api/oauth/token", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", a.getBasicAuth())
	req.Header.Set("User-Agent", EpicUserAgent)

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("epic oauth request failed: %s", resp.Status)
	}

	var tokenResp oauthTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, err
	}

	now := time.Now()
	return &EpicSession{
		AccessToken:  tokenResp.AccessToken,
		ExpiresAt:    now.Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
		RefreshToken: tokenResp.RefreshToken,
		RefreshAt:    now.Add(time.Duration(tokenResp.RefreshIn) * time.Second),
		AccountId:    tokenResp.AccountId,
		DisplayName:  tokenResp.DisplayName,
	}, nil
}

type exchangeCodeResponse struct {
	Code string `json:"code"`
}

func (a *EpicApi) GetExchangeCode(accessToken string) (string, error) {
	req, err := http.NewRequest("GET", "https://"+EpicOAuthHost+"/account/api/oauth/exchange", nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("User-Agent", EpicUserAgent)

	resp, err := a.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("epic exchange code request failed: %s", resp.Status)
	}

	var exchangeResp exchangeCodeResponse
	if err := json.NewDecoder(resp.Body).Decode(&exchangeResp); err != nil {
		return "", err
	}

	return exchangeResp.Code, nil
}
