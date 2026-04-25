package agent

import (
	"context"
	"errors"
	"sync"
)

type FakeAgent struct {
	mu        sync.Mutex
	Responses []Response
	Calls     []Request
	Editor    func(req Request) error
}

func (f *FakeAgent) Run(_ context.Context, req Request) (Response, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Calls = append(f.Calls, req)
	if len(f.Responses) == 0 {
		return Response{}, errors.New("FakeAgent: responses exhausted")
	}
	r := f.Responses[0]
	f.Responses = f.Responses[1:]
	if f.Editor != nil {
		if err := f.Editor(req); err != nil {
			return r, err
		}
	}
	return r, nil
}
