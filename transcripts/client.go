package transcripts

import (
	"context"
	"net/http"
	"net/url"
)

const (
	// ModelScribeV2Realtime the only supported model at the moment
	ModelScribeV2Realtime string = "scribe_v2_realtime"
)

type Client struct {
	config ClientConfig
}

// NewClient creates new OpenAI Realtime API client for specified auth token.
func NewClient(authToken string) *Client {
	return &Client{
		config: DefaultConfig(authToken),
	}
}

// NewClientWithConfig creates new OpenAI Realtime API client for specified config.
func NewClientWithConfig(config ClientConfig) *Client {
	return &Client{
		config: config,
	}
}

func (c *Client) getURL(queries map[string]string) string {
	query := url.Values{}

	if nil != queries {
		for k, v := range queries {
			query.Set(k, v)
		}
	}

	return c.config.BaseURL + "?" + query.Encode()
}

func (c *Client) getHeaders() http.Header {
	headers := http.Header{}
	headers.Set("xi-api-key", c.config.authToken)
	return headers
}

type connectOption struct {
	dialer  WebSocketDialer
	logger  Logger
	queries map[string]string
}
type ConnectOption func(*connectOption)

// WithQuery sets the query parameters for the connection.
func WithQuery(query map[string]string) ConnectOption {
	return func(opts *connectOption) {
		if nil == opts.queries {
			opts.queries = make(map[string]string)
		}
		for k, v := range query {
			opts.queries[k] = v
		}
	}
}

// WithDialer sets the dialer for the connection.
func WithDialer(dialer WebSocketDialer) ConnectOption {
	return func(opts *connectOption) {
		opts.dialer = dialer
	}
}

// WithLogger sets the logger for the connection.
func WithLogger(logger Logger) ConnectOption {
	return func(opts *connectOption) {
		opts.logger = logger
	}
}

// Connect connects to the Realtime API.
func (c *Client) Connect(ctx context.Context, opts ...ConnectOption) (*Conn, error) {
	connectOpts := connectOption{
		logger: NopLogger{},
		queries: map[string]string{
			"model_id":           ModelScribeV2Realtime,
			"audio_format":       string(AudioFormatPcm_16000),
			"include_timestamps": "true",
		},
	}
	for _, opt := range opts {
		opt(&connectOpts)
	}
	if connectOpts.dialer == nil {
		connectOpts.dialer = DefaultDialer()
	}

	// default headers
	headers := c.getHeaders()

	// get url by model
	uri := c.getURL(connectOpts.queries)

	// dial
	conn, err := connectOpts.dialer.Dial(ctx, uri, headers)
	if err != nil {
		return nil, err
	}

	return &Conn{conn: conn, logger: connectOpts.logger}, nil
}
