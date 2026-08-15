package metrics
import(
	"testing"
	"time"
)

func TestProtocolTracker( t * testing.T){
	pt:= NewProtocolTracker("http")
	pt.Record(10 * time.Millisecond)
    pt.Record(20 * time.Millisecond)
	pt.Record(30 * time.Millisecond)
	pt.Record(40 * time.Millisecond)
	pt.Record(50 * time.Millisecond)

	metrics := pt.Snapshot(2,5)
	if metrics.ActiveConnections != 2 {
		t.Errorf("Expected 2 active connections, got %d", metrics.ActiveConnections)
	}
	if metrics.MessageCount != 5 {
		t.Errorf("Expected 5 total messages, got %d", metrics.MessageCount)
	}
	if metrics.P50LatencyMs != 30.0 {
		t.Errorf("Expected p50 latency to be 30.0ms, got %.1fms", metrics.P50LatencyMs)
	}
	if metrics.P99LatencyMs != 50.0 {
		t.Errorf("Expected p99 latency to be 50.0ms, got %.1fms", metrics.P99LatencyMs)
	}
	if metrics.Throughput != 0.5 {
		t.Errorf("Expected throughput to be 0.5 msg/sec, got %.1f msg/sec", metrics.Throughput)
	}
}


