package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"
)

// ============================================================================
// VictoriaMetrics Client
// ============================================================================

// VMClient handles pushing metrics to VictoriaMetrics
type VMClient struct {
	config     *VMConfig
	httpClient *http.Client
}

// NewVMClient creates a new VictoriaMetrics client
func NewVMClient(config *VMConfig) *VMClient {
	log.Printf("[VM] VictoriaMetrics client initialized (URL: %s)", config.URL)
	log.Printf("[VM] Short interval: %v, Long interval: %v", config.ShortInterval, config.LongInterval)

	return &VMClient{
		config: config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// SendMetrics sends aggregated metrics to VictoriaMetrics using Prometheus format
func (c *VMClient) SendMetrics(window *AggregationWindow) error {
	if window == nil || len(window.Interfaces) == 0 {
		return nil
	}

	// Generate Prometheus-format metrics
	metrics := c.generatePrometheusMetrics(window)
	if len(metrics) == 0 {
		return nil
	}

	// Send to VictoriaMetrics with retry
	for attempt := 0; attempt <= c.config.RetryCount; attempt++ {
		if attempt > 0 {
			log.Printf("[VM] Retry attempt %d/%d", attempt, c.config.RetryCount)
			time.Sleep(time.Second * time.Duration(attempt))
		}

		err := c.sendToVM(metrics, window.EndTime)
		if err == nil {
			log.Printf("[VM] Successfully sent metrics for window [%s, %s) - %d interfaces",
				window.StartTime.Format("15:04:05"),
				window.EndTime.Format("15:04:05"),
				len(window.Interfaces),
			)
			return nil
		}

		log.Printf("[VM] Error sending metrics (attempt %d): %v", attempt+1, err)
	}

	return fmt.Errorf("failed after %d retries", c.config.RetryCount)
}

// generatePrometheusMetrics converts aggregation window to Prometheus format
func (c *VMClient) generatePrometheusMetrics(window *AggregationWindow) string {
	var buf bytes.Buffer
	timestamp := window.EndTime.Unix() * 1000 // Milliseconds

	for ifaceName, stats := range window.Interfaces {
		if stats.Count == 0 {
			continue
		}

		// Calculate averages
		rxAvg := stats.RxSum / float64(stats.Count)
		txAvg := stats.TxSum / float64(stats.Count)

		// Interface type label
		intervalLabel := fmt.Sprintf("%ds", int(window.Interval.Seconds()))

		// RX metrics (bytes/second)
		buf.WriteString(fmt.Sprintf("mikrotik_interface_rx_rate_avg{interface=\"%s\",interval=\"%s\"} %.2f %d\n",
			ifaceName, intervalLabel, rxAvg, timestamp))
		buf.WriteString(fmt.Sprintf("mikrotik_interface_rx_rate_peak{interface=\"%s\",interval=\"%s\"} %.2f %d\n",
			ifaceName, intervalLabel, stats.RxPeak, timestamp))
		buf.WriteString(fmt.Sprintf("mikrotik_interface_rx_rate_min{interface=\"%s\",interval=\"%s\"} %.2f %d\n",
			ifaceName, intervalLabel, stats.RxMin, timestamp))

		// TX metrics (bytes/second)
		buf.WriteString(fmt.Sprintf("mikrotik_interface_tx_rate_avg{interface=\"%s\",interval=\"%s\"} %.2f %d\n",
			ifaceName, intervalLabel, txAvg, timestamp))
		buf.WriteString(fmt.Sprintf("mikrotik_interface_tx_rate_peak{interface=\"%s\",interval=\"%s\"} %.2f %d\n",
			ifaceName, intervalLabel, stats.TxPeak, timestamp))
		buf.WriteString(fmt.Sprintf("mikrotik_interface_tx_rate_min{interface=\"%s\",interval=\"%s\"} %.2f %d\n",
			ifaceName, intervalLabel, stats.TxMin, timestamp))

		// Sample count
		buf.WriteString(fmt.Sprintf("mikrotik_interface_sample_count{interface=\"%s\",interval=\"%s\"} %d %d\n",
			ifaceName, intervalLabel, stats.Count, timestamp))
	}

	return buf.String()
}

// sendToVM sends metrics to VictoriaMetrics import API
func (c *VMClient) sendToVM(metrics string, timestamp time.Time) error {
	url := c.config.URL + "/api/v1/import/prometheus"

	req, err := http.NewRequest("POST", url, bytes.NewBufferString(metrics))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "text/plain")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// ============================================================================
// Query Methods
// ============================================================================

// HistoryQueryParams holds parameters for historical data query
type HistoryQueryParams struct {
	Interface string
	Start     time.Time
	End       time.Time
	Interval  string // "10s", "300s", or "auto"
}

// HistoryDataPoint represents a single data point in historical data
type HistoryDataPoint struct {
	Timestamp   time.Time `json:"timestamp"`
	UploadAvg   float64   `json:"upload_avg"`
	DownloadAvg float64   `json:"download_avg"`
	UploadPeak  float64   `json:"upload_peak"`
	DownloadPeak float64  `json:"download_peak"`
}

// HistoryResponse is the response structure for history queries
type HistoryResponse struct {
	Interface  string              `json:"interface"`
	Interval   string              `json:"interval"`
	Start      string              `json:"start"`
	End        string              `json:"end"`
	DataPoints []HistoryDataPoint  `json:"datapoints"`
	Stats      *OverallStats       `json:"stats,omitempty"`
}

// OverallStats holds aggregated statistics for the entire time range
type OverallStats struct {
	UploadAvg    float64 `json:"upload_avg"`
	DownloadAvg  float64 `json:"download_avg"`
	UploadPeak   float64 `json:"upload_peak"`
	DownloadPeak float64 `json:"download_peak"`
}

// QueryHistory queries historical data from VictoriaMetrics
func (c *VMClient) QueryHistory(params HistoryQueryParams) (*HistoryResponse, error) {
	// Determine interval (auto-select based on time range)
	interval := params.Interval
	if interval == "auto" || interval == "" {
		interval = c.autoSelectInterval(params.Start, params.End)
	}

	// Build PromQL queries
	queries := map[string]string{
		"upload_avg":   fmt.Sprintf(`mikrotik_interface_tx_rate_avg{interface="%s",interval="%s"}`, params.Interface, interval),
		"download_avg": fmt.Sprintf(`mikrotik_interface_rx_rate_avg{interface="%s",interval="%s"}`, params.Interface, interval),
		"upload_peak":  fmt.Sprintf(`mikrotik_interface_tx_rate_peak{interface="%s",interval="%s"}`, params.Interface, interval),
		"download_peak": fmt.Sprintf(`mikrotik_interface_rx_rate_peak{interface="%s",interval="%s"}`, params.Interface, interval),
	}

	// Query each metric
	results := make(map[string][]vmDataPoint)
	for metric, query := range queries {
		data, err := c.queryRange(query, params.Start, params.End)
		if err != nil {
			log.Printf("[VM] Warning: Failed to query %s: %v", metric, err)
			continue
		}
		results[metric] = data
	}

	// Query overall statistics (max of peaks for the entire time range)
	overallStats := c.queryOverallStats(params.Interface, interval, params.Start, params.End)

	// Merge results into unified data points
	dataPoints := c.mergeQueryResults(results)

	return &HistoryResponse{
		Interface:  params.Interface,
		Interval:   interval,
		Start:      params.Start.Format(time.RFC3339),
		End:        params.End.Format(time.RFC3339),
		DataPoints: dataPoints,
		Stats:      overallStats,
	}, nil
}

// queryOverallStats queries aggregated statistics for the entire time range using PromQL
func (c *VMClient) queryOverallStats(interfaceName, interval string, start, end time.Time) *OverallStats {
	stats := &OverallStats{}

	// Use PromQL aggregation functions to get true max/avg over the time range
	queries := map[string]string{
		"upload_avg":    fmt.Sprintf(`avg_over_time(mikrotik_interface_tx_rate_avg{interface="%s",interval="%s"}[%ds])`, interfaceName, interval, int(end.Sub(start).Seconds())),
		"download_avg":  fmt.Sprintf(`avg_over_time(mikrotik_interface_rx_rate_avg{interface="%s",interval="%s"}[%ds])`, interfaceName, interval, int(end.Sub(start).Seconds())),
		"upload_peak":   fmt.Sprintf(`max_over_time(mikrotik_interface_tx_rate_peak{interface="%s",interval="%s"}[%ds])`, interfaceName, interval, int(end.Sub(start).Seconds())),
		"download_peak": fmt.Sprintf(`max_over_time(mikrotik_interface_rx_rate_peak{interface="%s",interval="%s"}[%ds])`, interfaceName, interval, int(end.Sub(start).Seconds())),
	}

	for metric, query := range queries {
		value := c.queryInstant(query, end)
		switch metric {
		case "upload_avg":
			stats.UploadAvg = value
		case "download_avg":
			stats.DownloadAvg = value
		case "upload_peak":
			stats.UploadPeak = value
		case "download_peak":
			stats.DownloadPeak = value
		}
	}

	return stats
}

// queryInstant executes an instant query against VictoriaMetrics
func (c *VMClient) queryInstant(query string, timestamp time.Time) float64 {
	baseURL := fmt.Sprintf("%s/api/v1/query", c.config.URL)
	req, err := http.NewRequest("GET", baseURL, nil)
	if err != nil {
		log.Printf("[VM] Error creating instant query request: %v", err)
		return 0
	}

	q := req.URL.Query()
	q.Add("query", query)
	q.Add("time", fmt.Sprintf("%d", timestamp.Unix()))
	req.URL.RawQuery = q.Encode()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Printf("[VM] Error executing instant query: %v", err)
		return 0
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("[VM] Instant query failed (%d): %s", resp.StatusCode, string(body))
		return 0
	}

	var vmResp struct {
		Status string `json:"status"`
		Data   struct {
			ResultType string `json:"resultType"`
			Result     []struct {
				Metric map[string]string `json:"metric"`
				Value  []interface{}     `json:"value"`
			} `json:"result"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&vmResp); err != nil {
		log.Printf("[VM] Error decoding instant query response: %v", err)
		return 0
	}

	if vmResp.Status != "success" || len(vmResp.Data.Result) == 0 {
		return 0
	}

	if len(vmResp.Data.Result[0].Value) >= 2 {
		valueStr := vmResp.Data.Result[0].Value[1].(string)
		var val float64
		fmt.Sscanf(valueStr, "%f", &val)
		return val
	}

	return 0
}

// vmDataPoint is internal structure for VM query results
type vmDataPoint struct {
	Timestamp int64
	Value     float64
}

// queryRange executes a range query against VictoriaMetrics
func (c *VMClient) queryRange(query string, start, end time.Time) ([]vmDataPoint, error) {
	// Calculate appropriate step based on time range
	duration := end.Sub(start)
	var step int
	switch {
	case duration <= 1*time.Hour:
		step = 10 // 10 seconds for short ranges
	case duration <= 6*time.Hour:
		step = 30 // 30 seconds
	case duration <= 24*time.Hour:
		step = 60 // 1 minute
	case duration <= 7*24*time.Hour:
		step = 300 // 5 minutes for week
	default:
		step = 3600 // 1 hour for longer periods
	}

	// Build URL with proper encoding
	baseURL := fmt.Sprintf("%s/api/v1/query_range", c.config.URL)
	req, err := http.NewRequest("GET", baseURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Add query parameters
	q := req.URL.Query()
	q.Add("query", query)
	q.Add("start", fmt.Sprintf("%d", start.Unix()))
	q.Add("end", fmt.Sprintf("%d", end.Unix()))
	q.Add("step", fmt.Sprintf("%d", step))
	req.URL.RawQuery = q.Encode()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var vmResp struct {
		Status string `json:"status"`
		Data   struct {
			ResultType string `json:"resultType"`
			Result     []struct {
				Metric map[string]string `json:"metric"`
				Values [][]interface{}   `json:"values"`
			} `json:"result"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&vmResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if vmResp.Status != "success" {
		return nil, fmt.Errorf("query failed: %s", vmResp.Status)
	}

	// Extract data points
	var dataPoints []vmDataPoint
	if len(vmResp.Data.Result) > 0 {
		for _, value := range vmResp.Data.Result[0].Values {
			if len(value) >= 2 {
				timestamp := int64(value[0].(float64))
				valueStr := value[1].(string)
				var val float64
				fmt.Sscanf(valueStr, "%f", &val)
				dataPoints = append(dataPoints, vmDataPoint{
					Timestamp: timestamp,
					Value:     val,
				})
			}
		}
	}

	return dataPoints, nil
}

// mergeQueryResults merges multiple metric results into unified data points
func (c *VMClient) mergeQueryResults(results map[string][]vmDataPoint) []HistoryDataPoint {
	// Build timestamp index
	timestampMap := make(map[int64]*HistoryDataPoint)

	for metric, points := range results {
		for _, point := range points {
			dp, exists := timestampMap[point.Timestamp]
			if !exists {
				dp = &HistoryDataPoint{
					Timestamp: time.Unix(point.Timestamp, 0),
				}
				timestampMap[point.Timestamp] = dp
			}

			// Assign value to appropriate field
			switch metric {
			case "upload_avg":
				dp.UploadAvg = point.Value
			case "download_avg":
				dp.DownloadAvg = point.Value
			case "upload_peak":
				dp.UploadPeak = point.Value
			case "download_peak":
				dp.DownloadPeak = point.Value
			}
		}
	}

	// Convert to sorted slice
	var dataPoints []HistoryDataPoint
	for _, dp := range timestampMap {
		dataPoints = append(dataPoints, *dp)
	}

	// Sort by timestamp
	for i := 0; i < len(dataPoints)-1; i++ {
		for j := i + 1; j < len(dataPoints); j++ {
			if dataPoints[i].Timestamp.After(dataPoints[j].Timestamp) {
				dataPoints[i], dataPoints[j] = dataPoints[j], dataPoints[i]
			}
		}
	}

	return dataPoints
}

// autoSelectInterval automatically selects appropriate interval based on time range
func (c *VMClient) autoSelectInterval(start, end time.Time) string {
	duration := end.Sub(start)

	switch {
	case duration <= 1*time.Hour:
		return "10s" // Last hour: use 10-second data
	case duration <= 24*time.Hour:
		return "300s" // Last 24 hours: use 5-minute data
	default:
		return "300s" // Longer periods: use 5-minute data
	}
}

// ============================================================================
// Time Window Aggregator
// ============================================================================

// TimeWindowAggregator handles fixed-boundary time window aggregation
type TimeWindowAggregator struct {
	shortInterval time.Duration
	longInterval  time.Duration
	enableShort   bool
	enableLong    bool

	// Current aggregation windows
	currentShortWindow *AggregationWindow
	currentLongWindow  *AggregationWindow

	// Completed windows ready to send
	completedWindows []*AggregationWindow
	mu               sync.Mutex
}

// AggregationWindow represents a fixed time window with aggregated statistics
type AggregationWindow struct {
	StartTime  time.Time
	EndTime    time.Time
	Interval   time.Duration
	Interfaces map[string]*WindowStats
}

// WindowStats holds aggregated statistics for an interface within a window
type WindowStats struct {
	RxSum  float64 // Sum for average calculation
	TxSum  float64
	RxPeak float64 // Peak value
	TxPeak float64
	RxMin  float64 // Minimum value
	TxMin  float64
	Count  int // Number of samples
}

// NewTimeWindowAggregator creates a new time window aggregator
func NewTimeWindowAggregator(shortInterval, longInterval time.Duration, enableShort, enableLong bool) *TimeWindowAggregator {
	log.Printf("[Aggregator] Time window aggregator initialized")
	if enableShort {
		log.Printf("[Aggregator] Short-term window: %v", shortInterval)
	}
	if enableLong {
		log.Printf("[Aggregator] Long-term window: %v", longInterval)
	}

	return &TimeWindowAggregator{
		shortInterval:    shortInterval,
		longInterval:     longInterval,
		enableShort:      enableShort,
		enableLong:       enableLong,
		completedWindows: make([]*AggregationWindow, 0),
	}
}

// AddSample adds a sample to the current aggregation windows
func (a *TimeWindowAggregator) AddSample(timestamp time.Time, interfaceName string, rxRate, txRate float64) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Process short-term window
	if a.enableShort {
		a.currentShortWindow = a.addToWindow(a.currentShortWindow, a.shortInterval, timestamp, interfaceName, rxRate, txRate)
	}

	// Process long-term window
	if a.enableLong {
		a.currentLongWindow = a.addToWindow(a.currentLongWindow, a.longInterval, timestamp, interfaceName, rxRate, txRate)
	}
}

// addToWindow adds a sample to a specific window, creating new window if needed
func (a *TimeWindowAggregator) addToWindow(window *AggregationWindow, interval time.Duration, timestamp time.Time, ifaceName string, rxRate, txRate float64) *AggregationWindow {
	// Calculate window boundaries (aligned to interval)
	windowStart := timestamp.Truncate(interval)
	windowEnd := windowStart.Add(interval)

	// Create new window if needed
	if window == nil || !timestamp.Before(window.EndTime) {
		// Complete previous window
		if window != nil {
			a.completedWindows = append(a.completedWindows, window)
		}

		// Create new window
		window = &AggregationWindow{
			StartTime:  windowStart,
			EndTime:    windowEnd,
			Interval:   interval,
			Interfaces: make(map[string]*WindowStats),
		}
	}

	// Get or create interface stats
	stats, exists := window.Interfaces[ifaceName]
	if !exists {
		stats = &WindowStats{
			RxMin: rxRate,
			TxMin: txRate,
		}
		window.Interfaces[ifaceName] = stats
	}

	// Update statistics
	stats.RxSum += rxRate
	stats.TxSum += txRate
	stats.Count++

	// Update peak values
	if rxRate > stats.RxPeak {
		stats.RxPeak = rxRate
	}
	if txRate > stats.TxPeak {
		stats.TxPeak = txRate
	}

	// Update min values
	if rxRate < stats.RxMin {
		stats.RxMin = rxRate
	}
	if txRate < stats.TxMin {
		stats.TxMin = txRate
	}

	return window
}

// GetCompletedWindows returns and clears completed windows ready to send to VM
func (a *TimeWindowAggregator) GetCompletedWindows() []*AggregationWindow {
	a.mu.Lock()
	defer a.mu.Unlock()

	if len(a.completedWindows) == 0 {
		return nil
	}

	windows := a.completedWindows
	a.completedWindows = make([]*AggregationWindow, 0)
	return windows
}
