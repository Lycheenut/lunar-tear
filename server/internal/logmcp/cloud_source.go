package logmcp

import (
	"context"

	"cloud.google.com/go/logging"
	"cloud.google.com/go/logging/logadmin"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

type CloudSource struct {
	client *logadmin.Client
}

func NewCloudSource(ctx context.Context, project string) (*CloudSource, error) {
	client, err := logadmin.NewClient(ctx, project, option.WithScopes(logging.ReadScope))
	if err != nil {
		return nil, err
	}
	return &CloudSource{client: client}, nil
}

func (s *CloudSource) Query(ctx context.Context, filter string, limit int) ([]RawEntry, error) {
	entries := make([]RawEntry, 0, limit)
	it := s.client.Entries(
		ctx,
		logadmin.Filter(filter),
		logadmin.NewestFirst(),
		logadmin.PageSize(int32(limit)),
	)
	for len(entries) < limit {
		entry, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		entries = append(entries, RawEntry{
			Timestamp: entry.Timestamp,
			Severity:  entry.Severity.String(),
			Payload:   entry.Payload,
			InsertID:  entry.InsertID,
		})
	}
	return entries, nil
}

func (s *CloudSource) Close() error {
	return s.client.Close()
}
