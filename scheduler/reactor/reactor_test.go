package reactor

import (
	"fmt"
	"testing"
	"time"
)

type simpleGenerate struct {
	count int
}

func (s *simpleGenerate) Run() ([]*nuclear, error) {
	if s.count < 30 {
		count := s.count
		s.count++
		id := fmt.Sprintf("nucler#%d", count)
		return []*nuclear{NewNuclear(id, count)}, nil
	}
	return nil, fmt.Errorf("geneateor is down")
}

func TestReactorRun(t *testing.T) {
	geneateor := &simpleGenerate{count: 1}
	np := NewNuclearProductor(geneateor)
	r := NewReactor(5, 3, 2, np.notifyCh)
	time.Sleep(time.Second * 30)
	r.Kill()
	time.Sleep(time.Second)
}
