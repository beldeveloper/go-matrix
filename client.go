package gomatrix

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
)

const requestTimeout = time.Minute

type MediaType string

// https://spec.matrix.org/v1.13/client-server-api/#room-events
const (
	File  MediaType = "m.file"
	Image MediaType = "m.image"
	Audio MediaType = "m.audio"
	Video MediaType = "m.video"
)

type Credentials struct {
	Server   string
	User     string
	Password string
}

type Client struct {
	credentials Credentials
	httpClient  *http.Client

	mux            sync.RWMutex
	token          string
	sessionStorage SessionStorage
}

type Config struct {
	Credentials    Credentials
	SessionStorage SessionStorage
	HttpClient     *http.Client
}

func NewClientWithConfig(cfg Config) (*Client, error) {
	if cfg.HttpClient == nil {
		cfg.HttpClient = &http.Client{Timeout: requestTimeout}
	}

	c := &Client{
		credentials:    cfg.Credentials,
		httpClient:     cfg.HttpClient,
		sessionStorage: cfg.SessionStorage,
	}

	if c.sessionStorage != nil {
		sess, err := c.sessionStorage.Get()
		if err != nil {
			return nil, err
		}

		if sess.AccessToken != "" {
			c.token = sess.AccessToken
		}
	}

	if c.token == "" {
		err := c.authenticate(c.token)
		if err != nil {
			return nil, err
		}
	}

	return c, nil
}

func NewClient(cred Credentials) (*Client, error) {
	return NewClientWithConfig(Config{Credentials: cred})
}

func (c *Client) SendText(ctx context.Context, roomID, text string) error {
	return c.sendMessage(ctx, apiSendMsgReq{
		RoomID: roomID,
		Type:   "m.text",
		Body:   text,
	})
}

type Media struct {
	Type     MediaType
	Caption  string
	Filename string
	URI      string
}

func (c *Client) SendMedia(ctx context.Context, roomID string, media Media) error {
	return c.sendMessage(ctx, apiSendMsgReq{
		RoomID:   roomID,
		Type:     string(media.Type),
		Body:     media.Caption,
		Filename: media.Filename,
		URL:      media.URI,
	})
}

func (c *Client) sendMessage(ctx context.Context, msg apiSendMsgReq) error {
	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message payload")
	}

	path := fmt.Sprintf("/_matrix/client/v3/rooms/%s/send/m.room.message/%s", msg.RoomID, uuid.NewString())
	resp, err := c.doRequest(ctx, http.MethodPut, path, payload, func(r *http.Request) {
		r.Header.Set("Content-Type", "application/json")
	}, true)
	if err != nil {
		return fmt.Errorf("failed to send a message: %w", err)
	}
	defer resp.Body.Close()
	return nil
}

func (c *Client) UploadFile(ctx context.Context, contentType string, data []byte) (string, error) {
	resp, err := c.doRequest(ctx, http.MethodPost, "/_matrix/media/v3/upload", data, func(r *http.Request) {
		r.Header.Set("Content-Type", contentType)
		r.Header.Set("Content-Length", strconv.Itoa(len(data)))
	}, true)
	if err != nil {
		return "", fmt.Errorf("failed to upload a file: %w", err)
	}
	defer resp.Body.Close()

	var respData apiUploadResp
	err = json.NewDecoder(resp.Body).Decode(&respData)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal upload file response: %w", err)
	}

	return respData.URI, nil
}

func (c *Client) authenticate(prevToken string) error {
	c.mux.Lock()
	defer c.mux.Unlock()

	if c.token != prevToken {
		return nil
	}

	payload, err := json.Marshal(apiLoginReq{
		Type:     "m.login.password",
		User:     c.credentials.User,
		Password: c.credentials.Password,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal auth payload: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.credentials.Server+"/_matrix/client/v3/login", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to create an auth request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to do an auth request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("auth - unexpected status code: %d; body: %s", resp.StatusCode, respBody)
	}

	var sess Session
	err = json.NewDecoder(resp.Body).Decode(&sess)
	if err != nil {
		return fmt.Errorf("failed to unmarshal auth session: %w", err)
	}

	c.token = sess.AccessToken
	if c.sessionStorage != nil {
		return c.sessionStorage.Set(sess)
	}

	return nil
}

func (c *Client) doRequest(
	ctx context.Context, method, path string, payload []byte, reqFn func(r *http.Request), tryAuth bool,
) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.credentials.Server+path, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to create a request: %w", err)
	}

	token := c.getToken()
	req.Header.Set("Authorization", "Bearer "+token)

	if reqFn != nil {
		reqFn(req)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to do a request: %w", err)
	}

	if resp.StatusCode < 400 {
		return resp, nil
	}

	defer resp.Body.Close()

	if !tryAuth || resp.StatusCode != http.StatusUnauthorized {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code: %d; body: %s", resp.StatusCode, respBody)
	}

	err = c.authenticate(token)
	if err != nil {
		return nil, err
	}

	return c.doRequest(ctx, method, path, payload, reqFn, false)
}

func (c *Client) getToken() string {
	c.mux.RLock()
	defer c.mux.RUnlock()
	return c.token
}
