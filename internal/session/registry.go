package session

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const sessionsSubdir = "sessions"

var (
	ErrNotFound    = errors.New("session not found")
	ErrAmbiguous   = errors.New("ambiguous session alias")
	ErrUnsupported = errors.New("background sessions are not supported on this platform")
)

type State string

const (
	StateOK        State = "ok"
	StateUnhealthy State = "unhealthy"
)

type Record struct {
	ID               string    `json:"id"`
	Alias            string    `json:"alias"`
	PID              int       `json:"pid"`
	ProcessStartedAt time.Time `json:"process_started_at"`
	StartedAt        time.Time `json:"started_at"`
	LocalPort        int       `json:"local_port"`
	Host             string    `json:"host"`
	RemotePort       int       `json:"remote_port"`
	BastionID        string    `json:"bastion_id"`
	Region           string    `json:"region"`
	Profile          string    `json:"profile,omitempty"`
	UseEnv           bool      `json:"use_env,omitempty"`
	LogPath          string    `json:"log_path"`
}

type Entry struct {
	Record
	State         State `json:"state"`
	UptimeSeconds int64 `json:"uptime_seconds"`
}

type Registry struct {
	dir string
}

func SessionsDir(burrowDir string) string {
	return filepath.Join(burrowDir, sessionsSubdir)
}

func Open(burrowDir string) (*Registry, error) {
	dir := SessionsDir(burrowDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create sessions dir: %w", err)
	}
	return &Registry{dir: dir}, nil
}

func NewID() (string, error) {
	var b [2]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return time.Now().UTC().Format("20060102T150405Z") + "-" + hex.EncodeToString(b[:]), nil
}

func (r *Registry) Add(rec Record) error {
	if rec.ID == "" {
		return fmt.Errorf("session id is required")
	}
	path := r.path(rec.ID)
	data, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write session: %w", err)
	}
	return nil
}

func (r *Registry) Get(id string) (Record, error) {
	rec, err := r.read(id)
	if err != nil {
		return Record{}, err
	}
	return rec, nil
}

func (r *Registry) Delete(id string) error {
	path := r.path(id)
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return ErrNotFound
		}
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

func (r *Registry) List() ([]Entry, error) {
	records, err := r.loadAll()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	out := make([]Entry, 0, len(records))
	for _, rec := range records {
		if !IsAlive(rec) {
			_ = r.Delete(rec.ID)
			continue
		}
		entry := Entry{
			Record:        rec,
			UptimeSeconds: int64(now.Sub(rec.StartedAt).Seconds()),
		}
		if PortListening(rec.LocalPort) {
			entry.State = StateOK
		} else {
			entry.State = StateUnhealthy
		}
		out = append(out, entry)
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].StartedAt.Before(out[j].StartedAt)
	})
	return out, nil
}

func (r *Registry) FindByAlias(alias string) ([]Record, error) {
	records, err := r.loadAll()
	if err != nil {
		return nil, err
	}
	var matches []Record
	for _, rec := range records {
		if rec.Alias == alias && IsAlive(rec) {
			matches = append(matches, rec)
		}
	}
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].StartedAt.Before(matches[j].StartedAt)
	})
	return matches, nil
}

func (r *Registry) Resolve(ref string) (Record, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return Record{}, fmt.Errorf("session reference is required")
	}

	if rec, err := r.Get(ref); err == nil {
		if IsAlive(rec) {
			return rec, nil
		}
		_ = r.Delete(rec.ID)
		return Record{}, ErrNotFound
	} else if !errors.Is(err, ErrNotFound) {
		return Record{}, err
	}

	matches, err := r.FindByAlias(ref)
	if err != nil {
		return Record{}, err
	}
	switch len(matches) {
	case 0:
		return Record{}, ErrNotFound
	case 1:
		return matches[0], nil
	default:
		return Record{}, fmt.Errorf("%w: alias %q matches %d sessions (%s)",
			ErrAmbiguous, ref, len(matches), formatIDs(matches))
	}
}

func (r *Registry) loadAll() ([]Record, error) {
	entries, err := os.ReadDir(r.dir)
	if err != nil {
		return nil, fmt.Errorf("read sessions dir: %w", err)
	}

	var records []Record
	for _, ent := range entries {
		if ent.IsDir() || !strings.HasSuffix(ent.Name(), ".json") {
			continue
		}
		id := strings.TrimSuffix(ent.Name(), ".json")
		rec, err := r.read(id)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				continue
			}
			return nil, err
		}
		if !IsAlive(rec) {
			_ = r.Delete(rec.ID)
			continue
		}
		records = append(records, rec)
	}
	return records, nil
}

func (r *Registry) read(id string) (Record, error) {
	path := r.path(id)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Record{}, ErrNotFound
		}
		return Record{}, fmt.Errorf("read session: %w", err)
	}
	var rec Record
	if err := json.Unmarshal(data, &rec); err != nil {
		return Record{}, fmt.Errorf("parse session %q: %w", id, err)
	}
	if rec.ID == "" {
		rec.ID = id
	}
	return rec, nil
}

func (r *Registry) path(id string) string {
	return filepath.Join(r.dir, id+".json")
}

func formatIDs(records []Record) string {
	ids := make([]string, len(records))
	for i, rec := range records {
		ids[i] = rec.ID
	}
	return strings.Join(ids, ", ")
}
