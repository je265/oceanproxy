package domain

import (
	"fmt"
	"sync"
)

// PortPool manages port allocation for a specific plan type
type PortPool struct {
	mu             sync.RWMutex
	planType       string
	portRange      PortRange
	allocatedPorts map[int]string // port -> plan_id
	availablePorts []int
}

// NewPortPool creates a new port pool for a plan type
func NewPortPool(planType string, portRange PortRange) *PortPool {
	pool := &PortPool{
		planType:       planType,
		portRange:      portRange,
		allocatedPorts: make(map[int]string),
		availablePorts: make([]int, 0, portRange.Size()),
	}

	// Initialize available ports
	for port := portRange.Start; port <= portRange.End; port++ {
		pool.availablePorts = append(pool.availablePorts, port)
	}

	return pool
}

// AllocatePort allocates a port for a plan
func (pp *PortPool) AllocatePort(planID string) (int, error) {
	pp.mu.Lock()
	defer pp.mu.Unlock()

	if len(pp.availablePorts) == 0 {
		return 0, fmt.Errorf("no available ports in range %d-%d for plan type %s",
			pp.portRange.Start, pp.portRange.End, pp.planType)
	}

	// Get the first available port
	port := pp.availablePorts[0]
	pp.availablePorts = pp.availablePorts[1:]
	pp.allocatedPorts[port] = planID

	return port, nil
}

// ReleasePort releases a port back to the pool
func (pp *PortPool) ReleasePort(port int) error {
	pp.mu.Lock()
	defer pp.mu.Unlock()

	if !pp.portRange.Contains(port) {
		return fmt.Errorf("port %d is not in range %d-%d", port, pp.portRange.Start, pp.portRange.End)
	}

	if _, exists := pp.allocatedPorts[port]; !exists {
		return fmt.Errorf("port %d is not allocated", port)
	}

	delete(pp.allocatedPorts, port)
	pp.availablePorts = append(pp.availablePorts, port)

	return nil
}

// IsAllocated checks if a port is allocated
func (pp *PortPool) IsAllocated(port int) bool {
	pp.mu.RLock()
	defer pp.mu.RUnlock()

	_, exists := pp.allocatedPorts[port]
	return exists
}

// GetAllocatedPorts returns all allocated ports
func (pp *PortPool) GetAllocatedPorts() map[int]string {
	pp.mu.RLock()
	defer pp.mu.RUnlock()

	result := make(map[int]string)
	for port, planID := range pp.allocatedPorts {
		result[port] = planID
	}

	return result
}

// GetAvailableCount returns the number of available ports
func (pp *PortPool) GetAvailableCount() int {
	pp.mu.RLock()
	defer pp.mu.RUnlock()

	return len(pp.availablePorts)
}

// GetAllocatedCount returns the number of allocated ports
func (pp *PortPool) GetAllocatedCount() int {
	pp.mu.RLock()
	defer pp.mu.RUnlock()

	return len(pp.allocatedPorts)
}
