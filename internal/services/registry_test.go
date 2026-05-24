package services

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
)

type stubProvider struct{ name string }

func (p stubProvider) Name() string { return p.name }
func (p stubProvider) ListResources(context.Context, aws.Config) ([]Resource, error) {
	return nil, nil
}

func TestAllReturnsCopy(t *testing.T) {
	before := len(All())
	Register(stubProvider{name: "copy-test"})

	all := All()
	if len(all) != before+1 {
		t.Fatalf("expected %d providers, got %d", before+1, len(all))
	}
	all[before] = nil
	if All()[before] == nil {
		t.Fatal("All should return a defensive copy")
	}
}

func TestRegistryConcurrentAccess(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			Register(stubProvider{name: fmt.Sprintf("test-%d", n)})
			_ = All()
		}(i)
	}
	wg.Wait()
}
