package service

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"sync"
	"time"

	"github.com/jacinli/sky-guardwall/internal/model"
	"github.com/jacinli/sky-guardwall/internal/service/parser"
	"gorm.io/gorm"
)

type IptablesService struct {
	db      *gorm.DB
	mu      sync.Mutex
	syncing bool
}

func NewIptablesService(db *gorm.DB) *IptablesService {
	return &IptablesService{db: db}
}

type SyncResult struct {
	SyncedAt  time.Time `json:"synced_at"`
	RuleCount int       `json:"rule_count"`
	HasError  bool      `json:"has_error"`
	ErrorMsg  string    `json:"error_msg,omitempty"`
}

// Sync runs `iptables -S`, replaces all stored rules, and records sync meta.
func (s *IptablesService) Sync(ctx context.Context) *SyncResult {
	s.mu.Lock()
	if s.syncing {
		s.mu.Unlock()
		slog.Debug("iptables sync already in progress, skipping")
		return nil
	}
	s.syncing = true
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		s.syncing = false
		s.mu.Unlock()
	}()

	now := time.Now()
	result := &SyncResult{SyncedAt: now}

	out, err := runCmd(ctx, "iptables", "-S")
	if err != nil {
		result.HasError = true
		result.ErrorMsg = err.Error()
		slog.Error("iptables sync failed", "err", err)
		s.saveMeta(now, 0, true, err.Error())
		return result
	}

	rules := parser.ParseOutput(out, now)
	result.RuleCount = len(rules)

	txErr := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("1 = 1").Delete(&model.IptablesRule{}).Error; err != nil {
			return fmt.Errorf("delete old rules: %w", err)
		}
		if len(rules) > 0 {
			if err := tx.CreateInBatches(rules, 200).Error; err != nil {
				return fmt.Errorf("insert rules: %w", err)
			}
		}
		return nil
	})
	if txErr != nil {
		result.HasError = true
		result.ErrorMsg = txErr.Error()
		slog.Error("iptables db update failed", "err", txErr)
		s.saveMeta(now, 0, true, txErr.Error())
		return result
	}

	slog.Info("iptables sync completed", "rules", len(rules))
	s.saveMeta(now, len(rules), false, "")
	return result
}

// LastSync returns the most recent sync meta record.
func (s *IptablesService) LastSync() (*model.SyncMeta, error) {
	var meta model.SyncMeta
	if err := s.db.Order("synced_at DESC").First(&meta).Error; err != nil {
		return nil, err
	}
	return &meta, nil
}

// GetRules returns all currently stored iptables rules with optional chain filter.
func (s *IptablesService) GetRules(chain string) ([]model.IptablesRule, error) {
	var rules []model.IptablesRule
	q := s.db.Order("id ASC")
	if chain != "" {
		q = q.Where("chain = ?", chain)
	}
	if err := q.Find(&rules).Error; err != nil {
		return nil, err
	}
	return rules, nil
}

// GetChains returns distinct chain names from stored rules.
func (s *IptablesService) GetChains() ([]string, error) {
	var chains []string
	if err := s.db.Model(&model.IptablesRule{}).
		Distinct("chain").
		Order("chain ASC").
		Pluck("chain", &chains).Error; err != nil {
		return nil, err
	}
	return chains, nil
}

func (s *IptablesService) saveMeta(at time.Time, count int, hasErr bool, errMsg string) {
	meta := model.SyncMeta{
		SyncedAt:  at,
		RuleCount: count,
		HasError:  hasErr,
		ErrorMsg:  errMsg,
	}
	if err := s.db.Create(&meta).Error; err != nil {
		slog.Error("failed to save sync meta", "err", err)
	}
}

func runCmd(ctx context.Context, name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%s %v: %w — %s", name, args, err, stderr.String())
	}
	return stdout.String(), nil
}
