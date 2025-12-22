package dto

type HealthResponse struct {
	Status struct {
		StatusCode int `json:"status_code"`
	} `json:"status"`
	Memory struct {
		AllocMB      uint64 `json:"allocated_heap_objects_MB"`
		TotalAllocMB uint64 `json:"cumulative_allocated_MB"`
		SysMB        uint64 `json:"total_memory_from_OS_MB"`
		NumGC        uint32 `json:"gc_cycles"`
		NumGoroutine int    `json:"num_goroutines"`
	} `json:"memory"`
}
