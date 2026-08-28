package server

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"

	"foundryprotocol/protocol"
)

type pendingMsg struct {
	sess *Session
	msg  protocol.Message
}

type Session struct {
	srv       *Server
	conn      *websocket.Conn
	inbox     chan protocol.Message
	out       chan protocol.Message
	done      chan struct{}
	closeOnce sync.Once
	closed    atomic.Bool
	resync    atomic.Bool
	playerID  string
}

func newSession(s *Server, conn *websocket.Conn) *Session {
	return &Session{
		srv:   s,
		conn:  conn,
		inbox: make(chan protocol.Message, 128),
		out:   make(chan protocol.Message, 128),
		done:  make(chan struct{}),
	}
}

func (sess *Session) enqueue(m protocol.Message) {
	if sess.closed.Load() {
		return
	}
	select {
	case sess.out <- m:
	default:
		sess.resync.Store(true)
	}
}

func (sess *Session) flushOut() {
	for {
		select {
		case <-sess.out:
		default:
			return
		}
	}
}

func (sess *Session) drainInbox(out *[]pendingMsg) {
	for {
		select {
		case m := <-sess.inbox:
			*out = append(*out, pendingMsg{sess: sess, msg: m})
		default:
			return
		}
	}
}

func (sess *Session) readerLoop() {
	for {
		sess.conn.SetReadDeadline(time.Now().Add(120 * time.Second))
		_, data, err := sess.conn.ReadMessage()
		if err != nil {
			break
		}
		msg, err := protocol.Decode(data)
		if err != nil {
			sess.enqueue(protocol.Message{Type: protocol.TypeSystem, Value: "error", Text: "invalid message"})
			continue
		}
		select {
		case sess.inbox <- msg:
		default:
			sess.enqueue(protocol.Message{Type: protocol.TypeSystem, Value: "error", Text: "message dropped (queue full)"})
		}
	}
	sess.closeAndCleanup()
}

func (sess *Session) writerLoop() {
	for {
		if sess.resync.Swap(false) {
			sess.flushOut()
			if raw, ok := sess.srv.lastSnap.Load().([]byte); ok && len(raw) > 0 {
				if err := sess.write(raw); err != nil {
					sess.closeAndCleanup()
					return
				}
				continue
			}
		}
		select {
		case m := <-sess.out:
			raw, err := m.Encode()
			if err != nil {
				sess.srv.logger.Warn().Err(err).Msg("encode outbound message")
				continue
			}
			if err := sess.write(raw); err != nil {
				sess.closeAndCleanup()
				return
			}
		case <-sess.done:
			return
		case <-time.After(50 * time.Millisecond):
		}
	}
}

func (sess *Session) write(raw []byte) error {
	sess.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	return sess.conn.WriteMessage(websocket.TextMessage, raw)
}

func (sess *Session) closeAndCleanup() {
	sess.closeOnce.Do(func() {
		sess.closed.Store(true)
		close(sess.done)
		_ = sess.conn.Close()
		sess.srv.removeSession(sess)
	})
}
