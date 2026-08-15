package registry

import (
	"fmt"
	"sync"
	"testing"
	"time"
)
//This test verifies the logical correctness of the registry in a single-threaded context.
func TestConnectionRegistry_Basic(t *testing.T) {
	reg := NewConnectionRegistry()

	client := &Client{
		ID:          "client1",
		Protocol:    "websocket",
		ConnectedAt: time.Now(),
	}
	reg.Add(client)
	if reg.Count() != 1 {
		t.Errorf("Expected 1 connect but got %d", reg.Count())
	}

	reg.RecordMessage("client1")
	clients := reg.List()
	if len(clients) != 1 || clients[0].MessageCount != 1 {
		t.Errorf("expected message count to be 1, but got %d", clients[0].MessageCount)
	}

	reg.Remove("client1")

	if reg.Count() != 0 {
		t.Errorf("Expected 0 connect but got %d", reg.Count())
	}

}

//This test verifies the stability of the registry under concurrent load
func TestConnectionRegistry_Concurrency(t *testing.T) {
	reg := NewConnectionRegistry()
	var wg sync.WaitGroup
	numOps := 100
	for i := 0; i < numOps; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			client := &Client{
				ID:       fmt.Sprintf("client%d", id),
				Protocol: "http",
			}
			reg.Add(client)

		}(i)

	}
// if this code block runs before adding the client, it cannot find the connections so it does nothing, instead of panicking..
	for i := 0; i < numOps; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			reg.RecordMessage(fmt.Sprintf("client%d", id))
			
			_ = reg.Count()
			_ = reg.List()

		}(i)
	}
	wg.Wait()
}
