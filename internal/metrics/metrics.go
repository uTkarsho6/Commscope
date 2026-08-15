package metrics

import (
	"slices"
	"sync"
	"time"
)

// This represents the snapshot that is converted into JSON and sent to the dashboard
// whole purpose is to send data to react front-end; so that it can display charts, counters etc..
type ProtocolMetrics struct {
	Protocol          string
	ActiveConnections int
	MessageCount      int64
	P50LatencyMs      float64
	P99LatencyMs      float64
	Throughput        float64
}

// telemetry actually stored here,
// requires sliding window of recent latency duration and slice of timestamps, for msgs processed in last rolling window
type ProtocolTracker struct {
	mu           sync.Mutex
	protocol     string
	latencies    []time.Duration
	messageTimes []time.Time //timestamps of last N messages to calculate throughput
}

const maxLatencyWindow = 1000

// for proper initialization of tracker
func NewProtocolTracker(protocol string) *ProtocolTracker {
	return &ProtocolTracker{
		protocol: protocol,
	}
}

// Writes telemetry to the tracker; can be called by multiple goroutines at the same time

func (pt *ProtocolTracker) Record(latency time.Duration) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	if len(pt.latencies) >= maxLatencyWindow {
		pt.latencies = pt.latencies[1:]
	}
	pt.latencies = append(pt.latencies, latency)

	pt.messageTimes = append(pt.messageTimes, time.Now())

	//Pruning ensures we only hold the last 10 seconds of messages, capping our memory usage to a tiny, constant size.

	cutoff := time.Now().Add(-10 * time.Second) // cutoff := now - 10 sec
	idx := 0
	for idx < len(pt.messageTimes) && pt.messageTimes[idx].Before(cutoff) {
		idx++
	}
	if idx > 0 {
		pt.messageTimes = pt.messageTimes[idx:]
	}

}

// Reads telemetry
// Calculates latency percentiles and throughput for the rolling window
func (pt *ProtocolTracker) Snapshot(activeConnections int, totalMessages int64) ProtocolMetrics {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	// 1. Prune timestamps older than 10 seconds for accurate throughput
	cutoff := time.Now().Add(-10 * time.Second)
	idx := 0
	for idx < len(pt.messageTimes) && pt.messageTimes[idx].Before(cutoff) {
		idx++
	}
	if idx > 0 {
		pt.messageTimes = pt.messageTimes[idx:]
	}

	throughput := float64(len(pt.messageTimes)) / 10.0

	// 2. Declare variables in outer function scope
	var p50Ms, p99Ms float64

	// 3. Only calculate latencies if we have records
	if len(pt.latencies) > 0 {
		sorted := make([]time.Duration, len(pt.latencies))
		copy(sorted, pt.latencies)

		// sort.Slice(sorted, func(i, j int) bool {
		// 	return sorted[i] < sorted[j]
		// })

		slices.Sort(sorted)

		p50Idx := len(sorted) / 2
		p99Idx := int(float64(len(sorted)) * 0.99)

		// Prevent out-of-bounds if slice is small
		if p99Idx >= len(sorted) {
			p99Idx = len(sorted) - 1
		}

		// Convert time.Duration to millisecond floats
		// time.Duration is bydefault in ns
		p50Ms = float64(sorted[p50Idx].Microseconds()) / 1000.0
		p99Ms = float64(sorted[p99Idx].Microseconds()) / 1000.0
	}

	// 4. Return the populated metrics payload
	return ProtocolMetrics{
		Protocol:          pt.protocol,
		ActiveConnections: activeConnections,
		MessageCount:      totalMessages,
		P50LatencyMs:      p50Ms,
		P99LatencyMs:      p99Ms,
		Throughput:        throughput,
	}
}
