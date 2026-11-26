package appapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sync"

	"github.com/zalando/go-keyring"

	"github.com/yalcinumutt/zenith-cli/internal/ai"
	"github.com/yalcinumutt/zenith-cli/internal/calendar"
)

// planSettings is the planner's configuration, kept beside the database. It
// holds a private calendar link, so the file is owner-only. API keys are not
// here: they live in the system keychain.
type planSettings struct {
	WorkStart   int    `json:"workStart"`
	WorkEnd     int    `json:"workEnd"`
	CalendarURL string `json:"calendarUrl"`
	AIKind      string `json:"aiKind"`
	AIURL       string `json:"aiUrl"`
	AIModel     string `json:"aiModel"`
}

var defaultPlanSettings = planSettings{WorkStart: 9 * 60, WorkEnd: 18 * 60}

type planSettingsStore struct {
	mu   sync.Mutex
	path string
}

func (p *planSettingsStore) load() planSettings {
	p.mu.Lock()
	defer p.mu.Unlock()
	s := defaultPlanSettings
	if p.path != "" {
		if b, err := os.ReadFile(p.path); err == nil {
			_ = json.Unmarshal(b, &s)
		}
	}
	if s.WorkEnd-s.WorkStart < 30 || s.WorkStart < 0 || s.WorkEnd > 24*60 {
		s.WorkStart, s.WorkEnd = defaultPlanSettings.WorkStart, defaultPlanSettings.WorkEnd
	}
	return s
}

func (p *planSettingsStore) save(s planSettings) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.path == "" {
		return errors.New("planner settings are not available")
	}
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p.path), 0o755); err != nil {
		return err
	}
	tmp := p.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, p.path)
}

// secrets stores API keys by provider kind.
type secrets interface {
	Get(kind string) (string, error) // "" with nil error when unset
	Set(kind, key string) error      // an empty key deletes
}

const keychainService = "zenith-cli"

type keychain struct{}

func (keychain) Get(kind string) (string, error) {
	v, err := keyring.Get(keychainService, "ai-"+kind)
	if errors.Is(err, keyring.ErrNotFound) {
		return "", nil
	}
	return v, err
}

func (keychain) Set(kind, key string) error {
	if key == "" {
		err := keyring.Delete(keychainService, "ai-"+kind)
		if errors.Is(err, keyring.ErrNotFound) {
			return nil
		}
		return err
	}
	return keyring.Set(keychainService, "ai-"+kind, key)
}

// aiConfig assembles the provider settings, reading the key once per kind (the
// keychain can be slow) and remembering it until it changes.
func (s *Service) aiConfig() ai.ProviderConfig {
	st := s.plan.load()
	cfg := ai.ProviderConfig{Kind: st.AIKind, URL: st.AIURL, Model: st.AIModel}
	if cfg.Kind == ai.KindOpenAI || cfg.Kind == ai.KindAnthropic {
		cfg.APIKey = s.key(cfg.Kind)
	}
	return cfg
}

func (s *Service) key(kind string) string {
	s.keyMu.Lock()
	defer s.keyMu.Unlock()
	if k, ok := s.keys[kind]; ok {
		return k
	}
	k, err := s.secrets.Get(kind)
	if err != nil {
		k = "" // a locked or missing keychain reads as "no key"; saving reports the real error
	}
	if s.keys == nil {
		s.keys = map[string]string{}
	}
	s.keys[kind] = k
	return k
}

// ---- bound methods

func (s *Service) settingsDTO(st planSettings) PlanSettingsDTO {
	defURL, defModel := ai.Defaults(st.AIKind)
	return PlanSettingsDTO{
		WorkStart: st.WorkStart, WorkEnd: st.WorkEnd, CalendarURL: st.CalendarURL,
		AIKind: st.AIKind, AIURL: st.AIURL, AIModel: st.AIModel,
		AIHasKey: s.key(st.AIKind) != "", DefaultURL: defURL, DefaultModel: defModel,
	}
}

// PlanSettings returns the planner's settings.
func (s *Service) PlanSettings() PlanSettingsDTO { return s.settingsDTO(s.plan.load()) }

// SavePlanSettings validates and saves the planner's settings.
func (s *Service) SavePlanSettings(in PlanSettingsDTO) (PlanSettingsDTO, error) {
	if in.WorkStart < 0 || in.WorkEnd > 24*60 || in.WorkEnd-in.WorkStart < 30 {
		return PlanSettingsDTO{}, errors.New("working hours must span at least 30 minutes")
	}
	switch in.AIKind {
	case "", ai.KindLocal, ai.KindOpenAI, ai.KindAnthropic:
	default:
		return PlanSettingsDTO{}, fmt.Errorf("unknown AI provider %q", in.AIKind)
	}
	if in.AIURL != "" {
		if u, err := url.Parse(in.AIURL); err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return PlanSettingsDTO{}, errors.New("the AI address must start with http:// or https://")
		}
	}
	calURL, err := calendar.Normalize(in.CalendarURL)
	if err != nil {
		return PlanSettingsDTO{}, err
	}
	st := planSettings{WorkStart: in.WorkStart, WorkEnd: in.WorkEnd, CalendarURL: calURL, AIKind: in.AIKind, AIURL: in.AIURL, AIModel: in.AIModel}
	if err := s.plan.save(st); err != nil {
		return PlanSettingsDTO{}, err
	}
	if err := s.feed.SetURL(calURL); err != nil {
		return PlanSettingsDTO{}, err
	}
	return s.settingsDTO(st), nil
}

// SetAIKey stores (or, when empty, removes) the API key for a cloud provider.
func (s *Service) SetAIKey(kind, key string) error {
	if kind != ai.KindOpenAI && kind != ai.KindAnthropic {
		return errors.New("only cloud providers use an API key")
	}
	if err := s.secrets.Set(kind, key); err != nil {
		return fmt.Errorf("could not save the key to the system keychain: %w", err)
	}
	s.keyMu.Lock()
	if s.keys == nil {
		s.keys = map[string]string{}
	}
	s.keys[kind] = key
	s.keyMu.Unlock()
	return nil
}
