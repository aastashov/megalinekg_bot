package megaline

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const (
	loginURL   = "https://bill.mega.kg/?page=login"
	indexURL   = "https://bill.mega.kg/index.php"
	billingURL = "https://bill.mega.kg/page.php?page=main"
)

type Connector struct {
	client http.Client
}

func NewConnector(client http.Client) *Connector {
	return &Connector{
		client: client,
	}
}

func (that *Connector) Login(ctx context.Context, username, password string) ([]byte, string, error) {
	_, sessionID, err := that.makeRequest(ctx, http.MethodGet, loginURL, "", "")
	if err != nil {
		return nil, "", fmt.Errorf("get session id: %w", err)
	}

	payload := fmt.Sprintf("login=%s&pass=%s&act=login", username, password)
	body, _, err := that.makeRequest(ctx, http.MethodPost, loginURL, sessionID, payload)
	if err != nil {
		return nil, "", fmt.Errorf("login: %w", err)
	}

	return body, sessionID, nil
}

func (that *Connector) GetAccountsDetail(ctx context.Context, session, account string) ([]byte, error) {
	if _, _, err := that.makeRequest(ctx, http.MethodPost, indexURL, session, fmt.Sprintf("ls_change=%s", account)); err != nil {
		return nil, fmt.Errorf("change account: %w", err)
	}

	body, _, err := that.makeRequest(ctx, http.MethodGet, billingURL, session, "")
	if err != nil {
		return nil, fmt.Errorf("get account detail: %w", err)
	}

	return body, nil
}

func (that *Connector) makeRequest(ctx context.Context, method, pageURL, session, requestBody string) ([]byte, string, error) {
	req, err := http.NewRequestWithContext(ctx, method, pageURL, strings.NewReader(requestBody))
	if err != nil {
		return nil, "", fmt.Errorf("create request: %w", err)
	}

	if method == http.MethodPost {
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	}

	if session != "" {
		req.Header.Set("Cookie", fmt.Sprintf("PHPSESSID=%s", session))
	}

	resp, err := that.client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("make request: %w", err)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("read body: %w", err)
	}

	cookieValue := ""
	cookie := strings.Split(resp.Header.Get("Set-Cookie"), ";")
	if len(cookie) > 1 {
		cookieValue = strings.Replace(cookie[0], "PHPSESSID=", "", 1)
	}

	return body, cookieValue, nil
}
