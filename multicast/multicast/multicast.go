package multicast

import (
	"errors"
	"net"
	"sync"

	tomb "gopkg.in/tomb.v1"
)

const (
	UDP_PROTOCOL    string = "udp"
	maxDatagramSize int    = 8192
)

var (
	ErrorMulticastServerNotFound  error = errors.New("multicast: server not found")
	ErrorMulticastHandlerExists   error = errors.New("multicast: handler already exists")
	ErrorMulticastHandlerNotFound error = errors.New("multicast: handler not found")
)

// default
var _srv *multicast

func init() {
	_srv = new(multicast)
	_srv.srv = make(map[string]*server)
}

// multicast package
type multicast struct {
	mutex sync.Mutex // protecting remaining fields

	srv map[string]*server
}

func (m *multicast) registerMulticast(addr string, key HandlerKey, hnd Handler) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if srv, ok := m.srv[addr]; ok {
		return srv.RegisterHandler(key, hnd)
	}
	srv, err := NewServer(addr)
	if err != nil {
		return err
	}
	m.srv[srv.Addr()] = srv
	return srv.RegisterHandler(key, hnd)
}

func (m *multicast) unregisterMulticast(addr string, key HandlerKey) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	srv, ok := m.srv[addr]
	if ok {
		return ErrorMulticastServerNotFound
	}
	delete(m.srv, addr)
	srv.Close()
	return nil
}

func RegisterMulticast(addr string, key HandlerKey, hnd Handler) error {
	return _srv.registerMulticast(addr, key, hnd)
}

func UnregisterMulticast(addr string, key HandlerKey) error {
	return _srv.unregisterMulticast(addr, key)
}

// handler
type Handler func(src *net.UDPAddr, count int, data []byte)
type HandlerKey string

type server struct {
	tomb *tomb.Tomb

	addr     *net.UDPAddr
	handlers map[HandlerKey]Handler

	listener *net.UDPConn
}

func NewServer(a string) (*server, error) {
	addr, err := net.ResolveUDPAddr(UDP_PROTOCOL, a)
	if err != nil {
		return nil, err
	}
	s := new(server)
	s.tomb = new(tomb.Tomb)
	s.addr = addr
	s.handlers = make(map[HandlerKey]Handler)
	// serve
	go func() {
		defer s.tomb.Done()
		s.tomb.Kill(s.serve())
	}()
	return s, nil
}

func (s *server) Close() {
	s.tomb.Kill(nil)
	s.cleanAll()
}

func (s *server) Addr() string {
	return s.addr.String()
}

func (s *server) RegisterHandler(key HandlerKey, hnd Handler) error {
	if _, ok := s.handlers[key]; ok {
		return ErrorMulticastHandlerExists
	}
	s.handlers[key] = hnd
	return nil
}

func (s *server) UnregisterHandler(key HandlerKey) error {
	if _, ok := s.handlers[key]; !ok {
		return ErrorMulticastHandlerNotFound
	}
	delete(s.handlers, key)
	return nil
}

func (s *server) cleanAll() {
	s.handlers = nil
}

func (s *server) serve() error {
	l, err := net.ListenMulticastUDP(UDP_PROTOCOL, nil, s.addr)
	if err != nil {
		return err
	}
	l.SetReadBuffer(maxDatagramSize)

	for {
		b := make([]byte, maxDatagramSize)
		n, src, err := l.ReadFromUDP(b)
		if err != nil {
			return err
		}

		// dispatch
		for _, hnd := range s.handlers {
			go hnd(src, n, b)
		}

		// judge
		select {
		case <-s.tomb.Dying():
			return nil
		default:
			// continue
		}
	}
}

// broadcast
func Notify(a string, data []byte) (int, error) {
	addr, err := net.ResolveUDPAddr(UDP_PROTOCOL, a)
	if err != nil {
		return 0, err
	}
	c, err := net.DialUDP(UDP_PROTOCOL, nil, addr)
	if err != nil {
		return 0, err
	}
	n, err := c.Write(data)
	if err != nil {
		return 0, err
	}
	return n, nil
}
