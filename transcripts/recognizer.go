package transcripts

import (
	"context"
	"encoding/base64"
	"errors"
)

type ServerEventHandler func(ctx context.Context, event ServerEvent)

type Recognizer struct {
	ctx      context.Context
	conn     *Conn
	handlers []ServerEventHandler
	errCh    chan error
}

// NewRecognizer creates a new Recognizer.
func NewRecognizer(ctx context.Context, conn *Conn, handlers ...ServerEventHandler) *Recognizer {
	return &Recognizer{
		ctx:      ctx,
		conn:     conn,
		handlers: handlers,
		errCh:    make(chan error, 1),
	}
}

// Err returns a channel that receives errors from the ConnHandler.
// This could be used to wait for the goroutine to exit.
// If you don't need to wait for the goroutine to exit, there's no need to call this.
// This must be called after the connection is closed, otherwise it will block indefinitely.
func (r *Recognizer) Err() <-chan error {
	return r.errCh
}

// Start the recognizer.
func (r *Recognizer) Start() {
	go func() {
		err := r.run()
		if err != nil {
			r.errCh <- err
		}
		close(r.errCh)
	}()
}

func (r *Recognizer) Send(pcm []byte) error {
	return r.conn.SendMessage(r.ctx, InputAudioChunkEvent{
		Audio: base64.StdEncoding.EncodeToString(pcm),
	})
}

func (r *Recognizer) Commit() error {
	return r.conn.SendMessage(r.ctx, InputAudioChunkEvent{
		Commit: true,
	})
}

func (r *Recognizer) Stop() error {
	return r.conn.Close()
}

func (r *Recognizer) run() error {
	defer r.conn.logger.Debugf("conn handler exited")
	for {
		select {
		case <-r.ctx.Done():
			return r.ctx.Err()
		default:
		}

		msg, err := r.conn.ReadMessage(r.ctx)
		if err != nil {
			var permanent *PermanentError
			if errors.As(err, &permanent) {
				return permanent.Err
			}
			r.conn.logger.Warnf("read message temporary error: %+v", err)
			continue
		}
		for _, handler := range r.handlers {
			handler(r.ctx, msg)
		}
	}
}
