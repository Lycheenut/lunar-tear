package logmcp

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	defaultLimit       = 50
	maxLimit           = 200
	defaultSearchRange = 15 * time.Minute
	maxSearchRange     = time.Hour
	defaultContextSide = 30 * time.Second
	maxContextSide     = 5 * time.Minute
	queryTimeout       = 30 * time.Second
	maxQueryRunes      = 256
	maxMessageRunes    = 4000
	maxMessageBytes    = 100_000
)

var (
	allowedServices = []string{
		"lunar-tear-auth",
		"lunar-tear-cdn",
		"lunar-tear-server",
	}
	bearerPattern = regexp.MustCompile(`(?i)(\bauthorization\s*[:=]\s*bearer\s+)[^\s,;"']+`)
	jwtPattern    = regexp.MustCompile(`\beyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\b`)
	secretPattern = regexp.MustCompile(`(?i)(\b(?:api[_-]?key|access[_-]?token|refresh[_-]?token|password|passwd|secret|token)\s*[:=]\s*)[^\s,;&}"']+`)
)

// RawEntry is the subset of a Cloud Logging entry needed by the MCP server.
type RawEntry struct {
	Timestamp time.Time
	Severity  string
	Payload   any
	InsertID  string
}

// Source provides read-only access to the configured log backend.
type Source interface {
	Query(ctx context.Context, filter string, limit int) ([]RawEntry, error)
	Close() error
}

type Service struct {
	project string
	source  Source
	now     func() time.Time
}

func NewService(project string, source Source) (*Service, error) {
	project = strings.TrimSpace(project)
	if project == "" {
		return nil, fmt.Errorf("Google Cloud project is required")
	}
	if source == nil {
		return nil, fmt.Errorf("log source is required")
	}
	return &Service{project: project, source: source, now: time.Now}, nil
}

type ListServicesInput struct{}

type ServiceInfo struct {
	Name        string `json:"name"`
	Environment string `json:"environment"`
}

type ListServicesOutput struct {
	Services []ServiceInfo `json:"services"`
	LogID    string        `json:"log_id"`
}

func (s *Service) ListServices() ListServicesOutput {
	services := make([]ServiceInfo, 0, len(allowedServices))
	for _, name := range allowedServices {
		services = append(services, ServiceInfo{Name: name, Environment: "production"})
	}
	return ListServicesOutput{Services: services, LogID: "gcplogs-docker-driver"}
}

type SearchLogsInput struct {
	Service   string `json:"service" jsonschema:"Production service to query: lunar-tear-server, lunar-tear-cdn, or lunar-tear-auth"`
	Query     string `json:"query,omitempty" jsonschema:"Optional literal, case-insensitive text to find in the log message; maximum 256 characters"`
	StartTime string `json:"start_time,omitempty" jsonschema:"RFC3339 start time; defaults to 15 minutes before end_time"`
	EndTime   string `json:"end_time,omitempty" jsonschema:"RFC3339 end time; defaults to the current time"`
	Limit     int    `json:"limit,omitempty" jsonschema:"Maximum entries to return; defaults to 50 and cannot exceed 200"`
}

type GetLogContextInput struct {
	Service       string `json:"service" jsonschema:"Production service to query: lunar-tear-server, lunar-tear-cdn, or lunar-tear-auth"`
	Timestamp     string `json:"timestamp" jsonschema:"RFC3339 timestamp around which to retrieve logs"`
	BeforeSeconds int    `json:"before_seconds,omitempty" jsonschema:"Seconds before timestamp; defaults to 30 and cannot exceed 300"`
	AfterSeconds  int    `json:"after_seconds,omitempty" jsonschema:"Seconds after timestamp; defaults to 30 and cannot exceed 300"`
	Limit         int    `json:"limit,omitempty" jsonschema:"Maximum entries to return; defaults to 50 and cannot exceed 200"`
}

type LogRecord struct {
	Timestamp   string `json:"timestamp"`
	Service     string `json:"service"`
	Container   string `json:"container,omitempty"`
	ContainerID string `json:"container_id,omitempty"`
	Severity    string `json:"severity,omitempty"`
	Message     string `json:"message"`
	InsertID    string `json:"insert_id,omitempty"`
}

type SearchLogsOutput struct {
	Entries   []LogRecord `json:"entries"`
	StartTime string      `json:"start_time"`
	EndTime   string      `json:"end_time"`
	Truncated bool        `json:"truncated"`
}

type GetLogContextOutput struct {
	Entries   []LogRecord `json:"entries"`
	Timestamp string      `json:"timestamp"`
	StartTime string      `json:"start_time"`
	EndTime   string      `json:"end_time"`
	Truncated bool        `json:"truncated"`
}

func (s *Service) SearchLogs(ctx context.Context, input SearchLogsInput) (SearchLogsOutput, error) {
	service, err := validateService(input.Service)
	if err != nil {
		return SearchLogsOutput{}, err
	}
	limit, err := validateLimit(input.Limit)
	if err != nil {
		return SearchLogsOutput{}, err
	}
	start, end, err := s.searchRange(input.StartTime, input.EndTime)
	if err != nil {
		return SearchLogsOutput{}, err
	}

	query := strings.TrimSpace(input.Query)
	if utf8.RuneCountInString(query) > maxQueryRunes {
		return SearchLogsOutput{}, fmt.Errorf("query cannot exceed %d characters", maxQueryRunes)
	}
	filter := s.buildFilter(service, start, end, query)
	entries, truncated, err := s.query(ctx, filter, service, limit)
	if err != nil {
		return SearchLogsOutput{}, err
	}
	return SearchLogsOutput{
		Entries:   entries,
		StartTime: start.Format(time.RFC3339Nano),
		EndTime:   end.Format(time.RFC3339Nano),
		Truncated: truncated,
	}, nil
}

func (s *Service) GetLogContext(ctx context.Context, input GetLogContextInput) (GetLogContextOutput, error) {
	service, err := validateService(input.Service)
	if err != nil {
		return GetLogContextOutput{}, err
	}
	target, err := parseRequiredTime("timestamp", input.Timestamp)
	if err != nil {
		return GetLogContextOutput{}, err
	}
	before, err := contextDuration("before_seconds", input.BeforeSeconds)
	if err != nil {
		return GetLogContextOutput{}, err
	}
	after, err := contextDuration("after_seconds", input.AfterSeconds)
	if err != nil {
		return GetLogContextOutput{}, err
	}
	limit, err := validateLimit(input.Limit)
	if err != nil {
		return GetLogContextOutput{}, err
	}

	start, end := target.Add(-before), target.Add(after)
	entries, truncated, err := s.query(ctx, s.buildFilter(service, start, end, ""), service, limit)
	if err != nil {
		return GetLogContextOutput{}, err
	}
	slices.Reverse(entries)
	return GetLogContextOutput{
		Entries:   entries,
		Timestamp: target.Format(time.RFC3339Nano),
		StartTime: start.Format(time.RFC3339Nano),
		EndTime:   end.Format(time.RFC3339Nano),
		Truncated: truncated,
	}, nil
}

func (s *Service) searchRange(startValue, endValue string) (time.Time, time.Time, error) {
	end := s.now().UTC()
	var err error
	if strings.TrimSpace(endValue) != "" {
		end, err = parseRequiredTime("end_time", endValue)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
	}
	start := end.Add(-defaultSearchRange)
	if strings.TrimSpace(startValue) != "" {
		start, err = parseRequiredTime("start_time", startValue)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
	}
	if !start.Before(end) {
		return time.Time{}, time.Time{}, fmt.Errorf("start_time must be before end_time")
	}
	if end.Sub(start) > maxSearchRange {
		return time.Time{}, time.Time{}, fmt.Errorf("time range cannot exceed %s", maxSearchRange)
	}
	return start.UTC(), end.UTC(), nil
}

func (s *Service) buildFilter(service string, start, end time.Time, query string) string {
	parts := []string{
		fmt.Sprintf(`logName=%s`, quoteFilterValue(fmt.Sprintf("projects/%s/logs/gcplogs-docker-driver", s.project))),
		`jsonPayload.container.metadata.environment="production"`,
		fmt.Sprintf(`jsonPayload.container.metadata.service=%s`, quoteFilterValue(service)),
		fmt.Sprintf(`timestamp>=%s`, quoteFilterValue(start.Format(time.RFC3339Nano))),
		fmt.Sprintf(`timestamp<=%s`, quoteFilterValue(end.Format(time.RFC3339Nano))),
	}
	if query != "" {
		parts = append(parts, fmt.Sprintf(`jsonPayload.message=~%s`, quoteFilterValue("(?i)"+regexp.QuoteMeta(query))))
	}
	return strings.Join(parts, " AND ")
}

func (s *Service) query(ctx context.Context, filter, service string, limit int) ([]LogRecord, bool, error) {
	queryCtx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	raw, err := s.source.Query(queryCtx, filter, limit+1)
	if err != nil {
		return nil, false, fmt.Errorf("querying Cloud Logging: %w", err)
	}
	truncated := len(raw) > limit
	if truncated {
		raw = raw[:limit]
	}

	entries := make([]LogRecord, 0, len(raw))
	remaining := maxMessageBytes
	for _, entry := range raw {
		record := toLogRecord(entry, service)
		if len(record.Message) > remaining {
			record.Message = truncateUTF8Bytes(record.Message, remaining)
			truncated = true
		}
		if record.Message == "" && remaining == 0 {
			break
		}
		remaining -= len(record.Message)
		entries = append(entries, record)
		if remaining == 0 {
			break
		}
	}
	return entries, truncated, nil
}

func validateService(value string) (string, error) {
	value = strings.TrimSpace(value)
	if !slices.Contains(allowedServices, value) {
		return "", fmt.Errorf("service must be one of %s", strings.Join(allowedServices, ", "))
	}
	return value, nil
}

func validateLimit(value int) (int, error) {
	if value == 0 {
		return defaultLimit, nil
	}
	if value < 1 || value > maxLimit {
		return 0, fmt.Errorf("limit must be between 1 and %d", maxLimit)
	}
	return value, nil
}

func parseRequiredTime(name, value string) (time.Time, error) {
	parsed, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(value))
	if err != nil {
		return time.Time{}, fmt.Errorf("%s must be an RFC3339 timestamp: %w", name, err)
	}
	return parsed.UTC(), nil
}

func contextDuration(name string, seconds int) (time.Duration, error) {
	if seconds == 0 {
		return defaultContextSide, nil
	}
	if seconds < 1 || seconds > int(maxContextSide/time.Second) {
		return 0, fmt.Errorf("%s must be between 1 and %d", name, int(maxContextSide/time.Second))
	}
	return time.Duration(seconds) * time.Second, nil
}

func quoteFilterValue(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `"`, `\"`)
	return `"` + value + `"`
}

type dockerPayload struct {
	Container struct {
		Name     string            `json:"name"`
		ID       string            `json:"id"`
		Metadata map[string]string `json:"metadata"`
	} `json:"container"`
	Message string `json:"message"`
}

func toLogRecord(entry RawEntry, serviceHint string) LogRecord {
	payload := decodePayload(entry.Payload)
	service := payload.Container.Metadata["service"]
	if service == "" {
		service = serviceHint
	}
	containerID := payload.Container.ID
	if len(containerID) > 12 {
		containerID = containerID[:12]
	}
	message := payload.Message
	if message == "" {
		message = fmt.Sprint(entry.Payload)
	}
	message = truncateRunes(redact(message), maxMessageRunes)
	return LogRecord{
		Timestamp:   entry.Timestamp.UTC().Format(time.RFC3339Nano),
		Service:     service,
		Container:   payload.Container.Name,
		ContainerID: containerID,
		Severity:    entry.Severity,
		Message:     message,
		InsertID:    entry.InsertID,
	}
}

func decodePayload(payload any) dockerPayload {
	var decoded dockerPayload
	switch value := payload.(type) {
	case string:
		decoded.Message = value
		return decoded
	case []byte:
		if json.Unmarshal(value, &decoded) == nil {
			return decoded
		}
	}
	encoded, err := json.Marshal(payload)
	if err == nil {
		_ = json.Unmarshal(encoded, &decoded)
	}
	return decoded
}

func redact(message string) string {
	message = bearerPattern.ReplaceAllString(message, `${1}[REDACTED]`)
	message = jwtPattern.ReplaceAllString(message, `[REDACTED_JWT]`)
	return secretPattern.ReplaceAllString(message, `${1}[REDACTED]`)
}

func truncateRunes(value string, limit int) string {
	if utf8.RuneCountInString(value) <= limit {
		return value
	}
	runes := []rune(value)
	return string(runes[:limit-1]) + "…"
}

func truncateUTF8Bytes(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	if limit <= 0 {
		return ""
	}
	end := limit
	for end > 0 && !utf8.ValidString(value[:end]) {
		end--
	}
	return value[:end]
}
