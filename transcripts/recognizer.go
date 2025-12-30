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
}

// NewRecognizer creates a new Recognizer.
func NewRecognizer(ctx context.Context, conn *Conn, handlers ...ServerEventHandler) *Recognizer {
	return &Recognizer{
		ctx:      ctx,
		conn:     conn,
		handlers: handlers,
	}
}

// run runs the recognizer.
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

// StartContinuousRecognitionAsync asynchronously initiates continuous speech recognition operation.
func (r *Recognizer) StartContinuousRecognitionAsync() chan error {
	outcome := make(chan error)
	go func() {
		outcome <- nil
		if err := r.run(); err != nil {
			outcome <- err
			return
		}
	}()
	return outcome
}

// StopContinuousRecognitionAsync asynchronously terminates ongoing continuous speech recognition operation.
func (r *Recognizer) StopContinuousRecognitionAsync() chan error {
	outcome := make(chan error)
	go func() {
		if err := r.conn.Close(); err != nil {
			outcome <- err
			return
		}
		outcome <- nil
	}()
	return outcome
}
