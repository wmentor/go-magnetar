package history

import (
	"encoding/json"
	"os"

	"github.com/pkg/errors"
)

const (
	defaultHistoryLimit = 200
	minHistoryLimit     = 10
)

type Storage struct {
	records    []string
	limit      int
	filename   string
	currentIdx int
}

func New(filename string, limit int) *Storage {
	if limit < minHistoryLimit {
		limit = minHistoryLimit
	}

	s := &Storage{
		limit:    limit,
		filename: filename,
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		s.records = make([]string, 0)
		return s
	}

	if err := json.Unmarshal(data, &s.records); err != nil {
		s.records = make([]string, 0)
		return s
	}

	if len(s.records) > limit {
		s.records = s.records[len(s.records)-limit:]
	}

	s.currentIdx = len(s.records)

	return s
}

func (s *Storage) Add(record string) {
	s.records = append(s.records, record)
	if len(s.records) > s.limit {
		s.records = s.records[len(s.records)-s.limit:]
	}
	s.currentIdx = len(s.records)
	s.Save()
}

func (s *Storage) Prev() string {
	if s.currentIdx > 0 {
		s.currentIdx--
		return s.records[s.currentIdx]
	}
	return ""
}

func (s *Storage) Next() string {
	if s.currentIdx >= len(s.records) {
		s.currentIdx = len(s.records)
		return ""
	}
	record := s.records[s.currentIdx]
	s.currentIdx++
	return record
}

func (s *Storage) Save() error {
	data, err := json.Marshal(s.records)
	if err != nil {
		return errors.Wrap(err, "marshal history records")
	}

	if err := os.WriteFile(s.filename, data, 0644); err != nil {
		return errors.Wrapf(err, "write history to %s", s.filename)
	}

	return nil
}
