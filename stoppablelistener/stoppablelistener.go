package stoppablelistener

import (
	"errors"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	ErrClosedConnection = errors.New("use of closed network connection")
	ErrStopped          = errors.New("listener stopped")
	ErrUnableToWrap     = errors.New("unable to wrap listener")
)

type Listener struct {
	*net.TCPListener

	stop   chan struct{}
	closed bool
	once   sync.Once
}

func New(l net.Listener) (*Listener, error) {
	ln, ok := l.(*net.TCPListener)
	if !ok {
		return nil, ErrUnableToWrap
	}

	return &Listener{
		TCPListener: ln,
		stop:        make(chan struct{}),
	}, nil
}

func (l *Listener) Accept() (net.Conn, error) {
	for {
		if err := l.SetDeadline(time.Now().Add(time.Second)); err != nil {
			return nil, err
		}

		newConn, err := l.TCPListener.Accept()
		if err != nil {
			neterr, ok := err.(net.Error)
			if ok && neterr.Timeout() && neterr.Temporary() {
				// If this is a timeout then continue waiting for connections.
				continue
			}

			if strings.Contains(err.Error(), ErrClosedConnection.Error()) {
				return nil, ErrClosedConnection
			}

			return nil, err
		}

		select {
		case <-l.stop:
			if err == nil {
				if err := newConn.Close(); err != nil {
					return nil, err
				}
			}

			return nil, ErrStopped
		default:
			// channel is still open
		}

		return newConn, nil
	}
}

func (l *Listener) Port() (int, error) {
	port := ""
	addr := l.TCPListener.Addr().String()
	colon := strings.LastIndex(addr, ":")

	switch colon {
	case -1:
	case 4:
	default:
		port = addr[colon+1:]
	}

	return strconv.Atoi(port)
}

func (l *Listener) IsStopped() bool {
	return l.closed
}

// Stop ensures that the listener is only ever stopped once.
func (l *Listener) Stop() {
	l.once.Do(func() {
		l.closed = true
		close(l.stop)
	})
}
