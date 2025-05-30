// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package v9

import (
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/elastic/elastic-agent-libs/logp"

	"github.com/elastic/beats/v7/x-pack/filebeat/input/netflow/decoder/config"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/netflow/decoder/template"
)

// SessionKey is the key used to lookup sessions: exporter address + port
// + source ID.
type SessionKey struct {
	Addr     string
	SourceID uint32
}

// MakeSessionKey returns a session key.
func MakeSessionKey(addr net.Addr, sourceID uint32, shared bool) SessionKey {
	if shared {
		// If templates are shared, do not store the addr.
		return SessionKey{SourceID: sourceID}
	}
	return SessionKey{addr.String(), sourceID}
}

// TemplateKey is the type of key used to lookup templates.
type TemplateKey uint16

// TemplateWrapper wraps a template with an expiration flag.
type TemplateWrapper struct {
	Template *template.Template
	Delete   atomic.Bool
}

// SessionState holds the state for a single session (observation domain).
type SessionState struct {
	mutex        sync.RWMutex
	Templates    map[TemplateKey]*TemplateWrapper
	lastSequence uint32
	logger       *logp.Logger
	Delete       atomic.Bool
}

// NewSession creates a new session.
func NewSession(logger *logp.Logger) *SessionState {
	return &SessionState{
		logger:    logger,
		Templates: make(map[TemplateKey]*TemplateWrapper),
	}
}

// AddTemplate adds the passed template.
func (s *SessionState) AddTemplate(t *template.Template) {
	s.logger.Debugf("state %p addTemplate %d %p", s, t.ID, t)
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.Templates[TemplateKey(t.ID)] = &TemplateWrapper{Template: t}
}

// GetTemplate returns a template by ID.
func (s *SessionState) GetTemplate(id uint16) (template *template.Template) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	wrapper, found := s.Templates[TemplateKey(id)]
	if found {
		template = wrapper.Template
		wrapper.Delete.Store(false)
	}
	return template
}

// ExpireTemplates will remove those templates that have not been used
// since the last call to ExpireTemplates.
func (s *SessionState) ExpireTemplates() (alive int, removed int) {
	var toDelete []TemplateKey
	s.mutex.RLock()
	for id, template := range s.Templates {
		if !template.Delete.CompareAndSwap(false, true) {
			toDelete = append(toDelete, id)
		}
	}
	total := len(s.Templates)
	s.mutex.RUnlock()
	if len(toDelete) > 0 {
		s.mutex.Lock()
		total = len(s.Templates)
		for _, id := range toDelete {
			if template, found := s.Templates[id]; found && template.Delete.Load() {
				s.logger.Debugf("expired template %v", id)
				delete(s.Templates, id)
				removed++
			}
		}
		s.mutex.Unlock()
	}
	return total - removed, removed
}

// CheckReset returns if the session must be reset after the receipt of the
// given sequence number.
func (s *SessionState) CheckReset(seqNum uint32) (prev uint32, reset bool) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	prev = s.lastSequence
	if reset = !isValidSequence(prev, seqNum); reset {
		s.Templates = make(map[TemplateKey]*TemplateWrapper)
	}
	s.lastSequence = seqNum
	return prev, reset
}

func isValidSequence(current, next uint32) bool {
	return next-current < MaxSequenceDifference || current-next < MaxSequenceDifference
}

// SessionMap manages all the sessions for a collector.
type SessionMap struct {
	mutex    sync.RWMutex
	Sessions map[SessionKey]*SessionState
	logger   *logp.Logger
	metric   config.ActiveSessionsMetric
}

// NewSessionMap returns a new SessionMap.
func NewSessionMap(logger *logp.Logger, metric config.ActiveSessionsMetric) SessionMap {
	return SessionMap{
		logger:   logger,
		Sessions: make(map[SessionKey]*SessionState),
		metric:   metric,
	}
}

func (m *SessionMap) decreaseActiveSessions() {
	if m.metric == nil {
		return
	}

	m.metric.Dec()
}

func (m *SessionMap) increaseActiveSessions() {
	if m.metric == nil {
		return
	}

	m.metric.Inc()
}

// GetOrCreate looks up the given session key and returns an existing session
// or creates a new one.
func (m *SessionMap) GetOrCreate(key SessionKey) *SessionState {
	m.mutex.RLock()
	session, found := m.Sessions[key]
	if found {
		session.Delete.Store(false)
	}
	m.mutex.RUnlock()
	if !found {
		m.mutex.Lock()
		if session, found = m.Sessions[key]; !found {
			session = NewSession(m.logger)
			m.Sessions[key] = session
			m.increaseActiveSessions()
		}
		m.mutex.Unlock()
	}
	return session
}

func (m *SessionMap) cleanup() (aliveSession int, removedSession int, aliveTemplates int, removedTemplates int) {
	var toDelete []SessionKey
	m.mutex.RLock()
	total := len(m.Sessions)
	for key, session := range m.Sessions {
		a, r := session.ExpireTemplates()
		aliveTemplates += a
		removedTemplates += r
		if !session.Delete.CompareAndSwap(false, true) {
			toDelete = append(toDelete, key)
		}
	}
	m.mutex.RUnlock()
	if len(toDelete) > 0 {
		m.mutex.Lock()
		total = len(m.Sessions)
		for _, key := range toDelete {
			if session, found := m.Sessions[key]; found && session.Delete.Load() {
				delete(m.Sessions, key)
				removedSession++
				m.decreaseActiveSessions()
			}
		}
		m.mutex.Unlock()
	}
	return total - removedSession, removedSession, aliveTemplates, removedTemplates
}

// CleanupLoop will expire the sessions that have been inactive for the given
// interval.
func (m *SessionMap) CleanupLoop(interval time.Duration, done <-chan struct{}) {
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-done:
			return

		case <-t.C:
			aliveS, removedS, aliveT, removedT := m.cleanup()
			if removedS > 0 || removedT > 0 {
				m.logger.Debugf("Expired %d sessions (%d remain) / %d templates (%d remain)", removedS, aliveS, removedT, aliveT)
			}
		}
	}
}
