package api

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"smoi/internal/database"
)

type GenerationJob struct {
	ID                 string `json:"id"`
	ConversationID     string `json:"conversation_id"`
	UserMessageID      string `json:"user_message_id"`
	AssistantMessageID string `json:"assistant_message_id"`
	Status             string `json:"status"`
	Error              string `json:"error"`
	CreatedAt          int64  `json:"created_at"`
	UpdatedAt          int64  `json:"updated_at"`
}

type activeGeneration struct {
	id     string
	cancel context.CancelFunc
}

type requestedGeneration struct {
	id        string
	createdAt int64
}

type JobManager struct {
	DB            *database.DB
	Server        *Server
	mu            sync.Mutex
	active        map[string]activeGeneration
	latest        map[string]requestedGeneration
	workers       map[string]bool
	stopRequested map[string]bool
	cancelReasons map[string]string
}

func NewJobManager(server *Server) *JobManager {
	return &JobManager{
		DB:            server.DB,
		Server:        server,
		active:        make(map[string]activeGeneration),
		latest:        make(map[string]requestedGeneration),
		workers:       make(map[string]bool),
		stopRequested: make(map[string]bool),
		cancelReasons: make(map[string]string),
	}
}

func (m *JobManager) Start() {
	_, err := m.DB.Exec(`
        UPDATE generation_jobs
        SET status = 'failed', error = 'Server restarted during generation', updated_at = ?
        WHERE status = 'running'
    `, time.Now().UnixMicro())
	if err != nil {
		slog.Error("could not recover interrupted generation jobs", "error", err)
	}

	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		m.SchedulePending()
		for range ticker.C {
			m.SchedulePending()
		}
	}()
}

func (m *JobManager) SchedulePending() {
	rows, err := m.DB.Query(`
        SELECT DISTINCT conversation_id FROM generation_jobs WHERE status = 'pending'
    `)
	if err != nil {
		slog.Error("could not inspect pending generations", "error", err)
		return
	}

	var conversationIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			conversationIDs = append(conversationIDs, id)
		}
	}
	if err := rows.Err(); err != nil {
		slog.Error("could not read pending generations", "error", err)
	}
	_ = rows.Close()

	for _, id := range conversationIDs {
		m.Launch(id)
	}
}

func (m *JobManager) Supersede(job GenerationJob) {
	var latestID string
	err := m.DB.QueryRow(`
        SELECT id FROM generation_jobs
        WHERE conversation_id = ?
        ORDER BY created_at DESC, id DESC LIMIT 1
    `, job.ConversationID).Scan(&latestID)
	if err != nil {
		slog.Error("could not identify latest generation", "conversation", job.ConversationID, "error", err)
		m.Launch(job.ConversationID)
		return
	}
	if latestID != job.ID {
		m.Launch(job.ConversationID)
		return
	}

	m.mu.Lock()
	current, exists := m.latest[job.ConversationID]
	if !exists || job.CreatedAt > current.createdAt {
		m.latest[job.ConversationID] = requestedGeneration{id: job.ID, createdAt: job.CreatedAt}
		delete(m.stopRequested, job.ConversationID)

		if active, found := m.active[job.ConversationID]; found && active.id != job.ID {
			m.cancelReasons[active.id] = "Interrupted by a newer message"
			active.cancel()
		}
	}
	m.mu.Unlock()

	m.Launch(job.ConversationID)
}

func (m *JobManager) Launch(conversationID string) {
	m.mu.Lock()
	if m.workers[conversationID] {
		m.mu.Unlock()
		return
	}
	m.workers[conversationID] = true
	m.mu.Unlock()

	go m.runConversation(conversationID)
}

func (m *JobManager) runConversation(conversationID string) {
	defer func() {
		m.mu.Lock()
		delete(m.workers, conversationID)
		delete(m.latest, conversationID)
		delete(m.stopRequested, conversationID)
		m.mu.Unlock()
	}()

	for {
		job, model, err := m.nextPendingJob(conversationID)
		if errors.Is(err, sql.ErrNoRows) {
			return
		}
		if err != nil {
			slog.Error("generation dispatch failed", "conversation", conversationID, "error", err)
			return
		}

		ctx, cancel := context.WithCancel(context.Background())
		m.mu.Lock()
		m.active[conversationID] = activeGeneration{id: job.ID, cancel: cancel}
		if latest, exists := m.latest[conversationID]; !exists || job.CreatedAt > latest.createdAt {
			m.latest[conversationID] = requestedGeneration{id: job.ID, createdAt: job.CreatedAt}
		}
		if m.stopRequested[conversationID] {
			cancel()
			delete(m.stopRequested, conversationID)
		}
		if latest, exists := m.latest[conversationID]; exists && latest.id != job.ID {
			m.cancelReasons[job.ID] = "Interrupted by a newer message"
			cancel()
		}
		m.mu.Unlock()

		m.process(ctx, job, model)
		cancel()

		m.mu.Lock()
		delete(m.active, conversationID)
		delete(m.cancelReasons, job.ID)
		m.mu.Unlock()
	}
}

func (m *JobManager) nextPendingJob(conversationID string) (GenerationJob, string, error) {
	for {
		var job GenerationJob
		var model string
		err := m.DB.QueryRow(`
            SELECT id, conversation_id, user_message_id, assistant_message_id,
                   model, status, error, created_at, updated_at
            FROM generation_jobs
            WHERE conversation_id = ? AND status = 'pending'
            ORDER BY created_at DESC, id DESC LIMIT 1
        `, conversationID).Scan(
			&job.ID, &job.ConversationID, &job.UserMessageID, &job.AssistantMessageID,
			&model, &job.Status, &job.Error, &job.CreatedAt, &job.UpdatedAt,
		)
		if err != nil {
			return GenerationJob{}, "", err
		}

		result, err := m.DB.Exec(`
            UPDATE generation_jobs SET status = 'running', updated_at = ?
            WHERE id = ? AND status = 'pending'
        `, time.Now().UnixMicro(), job.ID)
		if err != nil {
			return GenerationJob{}, "", err
		}

		changed, err := result.RowsAffected()
		if err != nil {
			return GenerationJob{}, "", err
		}
		if changed == 1 {
			job.Status = "running"
			return job, model, nil
		}
	}
}

func (m *JobManager) process(ctx context.Context, job GenerationJob, model string) {
	history, err := m.DB.Messages(ctx, job.ConversationID)
	if err != nil {
		if errors.Is(ctx.Err(), context.Canceled) {
			m.finish(job, "stopped", m.cancellationReason(job.ID))
		} else {
			m.finish(job, "failed", "Could not load message history")
			slog.Error("generation history failed", "job", job.ID, "error", err)
		}
		return
	}

	contextMessages, skippedAttachments := generationContext(history, job)

	slog.Info("generation started", "conversation", job.ConversationID, "job", job.ID, "model", model)
	slog.Debug("generation context", "job", job.ID, "messages", contextMessages,
		"previous_attachments_not_forwarded", skippedAttachments)

	var content strings.Builder
	lastSave := time.Now()
	streamErr := m.Server.Upstream.Stream(ctx, model, contextMessages, func(delta string) error {
		content.WriteString(delta)
		slog.Debug("upstream response delta", "job", job.ID, "content", delta)

		if time.Since(lastSave) >= 150*time.Millisecond {
			if err := m.saveContent(job, content.String()); err != nil {
				return err
			}
			lastSave = time.Now()
		}
		return nil
	})

	if err := m.saveContent(job, content.String()); err != nil && streamErr == nil {
		streamErr = err
	}

	if streamErr == nil {
		m.finish(job, "completed", "")
		slog.Info("generation completed", "conversation", job.ConversationID, "job", job.ID,
			"response_bytes", content.Len())
		slog.Debug("upstream complete response", "job", job.ID, "content", content.String())
		return
	}

	if errors.Is(ctx.Err(), context.Canceled) {
		reason := m.cancellationReason(job.ID)
		m.finish(job, "stopped", reason)
		slog.Info("generation stopped", "conversation", job.ConversationID, "job", job.ID, "reason", reason)
	} else {
		m.finish(job, "failed", streamErr.Error())
		slog.Error("generation failed", "conversation", job.ConversationID, "job", job.ID,
			"error", streamErr)
	}
}

func generationContext(history []database.Message, job GenerationJob) ([]database.Message, int) {
	contextMessages := make([]database.Message, 0, len(history))
	skippedAttachments := 0

	for _, message := range history {
		if message.ID == job.AssistantMessageID {
			break
		}
		if message.Role == "assistant" && message.Content == "" {
			continue
		}
		if message.Role == "user" && message.ID != job.UserMessageID && len(message.Attachments) > 0 {
			filenames := make([]string, 0, len(message.Attachments))
			for _, attachment := range message.Attachments {
				filenames = append(filenames, attachment.Filename)
			}

			skippedAttachments += len(message.Attachments)
			message.Attachments = nil
			if message.Content != "" {
				message.Content += "\n\n"
			}
			message.Content += "[Earlier attachments not sent again: " + strings.Join(filenames, ", ") + "]"
		}

		contextMessages = append(contextMessages, message)
	}

	return contextMessages, skippedAttachments
}

func (m *JobManager) cancellationReason(jobID string) string {
	m.mu.Lock()
	defer m.mu.Unlock()

	if reason := m.cancelReasons[jobID]; reason != "" {
		return reason
	}
	return "Stopped by user"
}

func (m *JobManager) saveContent(job GenerationJob, content string) error {
	_, err := m.DB.Exec(`
        UPDATE messages SET content = ? WHERE id = ? AND conversation_id = ?
    `, content, job.AssistantMessageID, job.ConversationID)
	if err == nil {
		_, err = m.DB.Exec(`
            UPDATE generation_jobs SET updated_at = ? WHERE id = ?
        `, time.Now().UnixMicro(), job.ID)
	}
	return err
}

func (m *JobManager) finish(job GenerationJob, status, description string) {
	_, err := m.DB.Exec(`
        UPDATE generation_jobs SET status = ?, error = ?, updated_at = ? WHERE id = ?
    `, status, description, time.Now().UnixMicro(), job.ID)
	if err != nil {
		slog.Error("could not save generation status", "job", job.ID, "error", err)
	}
}

func (m *JobManager) Stop(conversationID string) bool {
	m.mu.Lock()
	active, found := m.active[conversationID]
	if found {
		m.cancelReasons[active.id] = "Stopped by user"
		active.cancel()
		m.mu.Unlock()
		return true
	}
	if m.workers[conversationID] {
		m.stopRequested[conversationID] = true
		m.mu.Unlock()
		return true
	}
	m.mu.Unlock()

	result, err := m.DB.Exec(`
        UPDATE generation_jobs
        SET status = 'stopped', error = 'Stopped by user', updated_at = ?
        WHERE conversation_id = ? AND status = 'pending'
    `, time.Now().UnixMicro(), conversationID)
	if err != nil {
		slog.Error("could not stop pending generation", "conversation", conversationID, "error", err)
		return false
	}
	affected, err := result.RowsAffected()
	return err == nil && affected > 0
}

func (s *Server) stopGeneration(w http.ResponseWriter, r *http.Request, user database.User) {
	conversation, ok := s.ownedConversation(w, r, user)
	if !ok {
		return
	}

	stopped := s.Jobs.Stop(conversation.ID)
	slog.Info("generation stop requested", "conversation", conversation.ID, "user", user.ID, "stopped", stopped)
	jsonResponse(w, http.StatusOK, map[string]bool{"stopped": stopped})
}

func (s *Server) conversationJobs(ctx context.Context, conversationID string) ([]GenerationJob, error) {
	rows, err := s.DB.QueryContext(ctx, `
        SELECT id, conversation_id, user_message_id, assistant_message_id,
               status, error, created_at, updated_at
        FROM generation_jobs
        WHERE conversation_id = ?
        ORDER BY created_at, id
    `, conversationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	jobs := make([]GenerationJob, 0)
	for rows.Next() {
		var job GenerationJob
		if err := rows.Scan(
			&job.ID, &job.ConversationID, &job.UserMessageID, &job.AssistantMessageID,
			&job.Status, &job.Error, &job.CreatedAt, &job.UpdatedAt,
		); err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}
	return jobs, rows.Err()
}
