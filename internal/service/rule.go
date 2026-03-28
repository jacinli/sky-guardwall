package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"regexp"
	"strconv"
	"strings"

	"github.com/jacinli/sky-guardwall/internal/model"
	"gorm.io/gorm"
)

type RuleService struct {
	db *gorm.DB
}

func NewRuleService(db *gorm.DB) *RuleService {
	return &RuleService{db: db}
}

type AddRuleRequest struct {
	Description string `json:"description"`
	Chain       string `json:"chain"`
	SrcIP       string `json:"src_ip"`
	Protocol    string `json:"protocol"`
	DstPort     int    `json:"dst_port"`
	Target      string `json:"target"`
}

var allowedChains = map[string]bool{
	"INPUT": true, "OUTPUT": true, "FORWARD": true,
}
var allowedTargets = map[string]bool{
	"ACCEPT": true, "DROP": true, "REJECT": true,
}
var allowedProtocols = map[string]bool{
	"tcp": true, "udp": true, "all": true, "icmp": true, "": true,
}
var chainRe = regexp.MustCompile(`^[A-Z][A-Z0-9\-]{0,28}$`)

func (s *RuleService) Validate(req *AddRuleRequest) error {
	req.Chain = strings.ToUpper(strings.TrimSpace(req.Chain))
	req.Target = strings.ToUpper(strings.TrimSpace(req.Target))
	req.Protocol = strings.ToLower(strings.TrimSpace(req.Protocol))
	req.SrcIP = strings.TrimSpace(req.SrcIP)

	if !chainRe.MatchString(req.Chain) {
		return fmt.Errorf("invalid chain: %q", req.Chain)
	}
	if !allowedTargets[req.Target] {
		return fmt.Errorf("invalid target: %q (must be ACCEPT/DROP/REJECT)", req.Target)
	}
	if !allowedProtocols[req.Protocol] {
		return fmt.Errorf("invalid protocol: %q", req.Protocol)
	}
	if req.SrcIP != "" {
		if err := validateIPorCIDR(req.SrcIP); err != nil {
			return err
		}
	}
	if req.DstPort < 0 || req.DstPort > 65535 {
		return fmt.Errorf("invalid port: %d", req.DstPort)
	}
	if req.DstPort > 0 && req.Protocol == "all" {
		req.Protocol = "tcp" // port requires a protocol
	}
	return nil
}

// AddRule validates, executes iptables -I, and persists the rule.
func (s *RuleService) AddRule(ctx context.Context, req AddRuleRequest) (*model.ManagedRule, error) {
	if err := s.Validate(&req); err != nil {
		return nil, err
	}

	args := buildArgs(req)
	argsJSON, _ := json.Marshal(args)

	if err := execIptables(ctx, "-I", args); err != nil {
		return nil, fmt.Errorf("iptables write failed: %w", err)
	}

	slog.Info("iptables rule added",
		"chain", req.Chain, "src_ip", req.SrcIP,
		"protocol", req.Protocol, "dst_port", req.DstPort,
		"target", req.Target,
	)

	rule := &model.ManagedRule{
		Description:  req.Description,
		Chain:        req.Chain,
		SrcIP:        req.SrcIP,
		Protocol:     req.Protocol,
		DstPort:      req.DstPort,
		Target:       req.Target,
		IptablesArgs: string(argsJSON),
		IsApplied:    true,
	}
	if err := s.db.Create(rule).Error; err != nil {
		return nil, fmt.Errorf("db save failed: %w", err)
	}
	return rule, nil
}

// DeleteRule removes the rule from iptables and the database.
func (s *RuleService) DeleteRule(ctx context.Context, id uint) error {
	var rule model.ManagedRule
	if err := s.db.First(&rule, id).Error; err != nil {
		return fmt.Errorf("rule %d not found: %w", id, err)
	}

	if rule.IsApplied {
		var args []string
		if err := json.Unmarshal([]byte(rule.IptablesArgs), &args); err != nil {
			return fmt.Errorf("invalid stored args: %w", err)
		}
		if err := execIptables(ctx, "-D", args); err != nil {
			slog.Warn("iptables delete failed (rule may already be gone)", "err", err)
		} else {
			slog.Info("iptables rule deleted", "rule_id", id, "chain", rule.Chain)
		}
	}

	return s.db.Delete(&rule).Error
}

// ListRules returns all managed rules.
func (s *RuleService) ListRules() ([]model.ManagedRule, error) {
	var rules []model.ManagedRule
	if err := s.db.Order("created_at DESC").Find(&rules).Error; err != nil {
		return nil, err
	}
	return rules, nil
}

// buildArgs constructs the iptables arg slice (without -I/-D prefix).
func buildArgs(req AddRuleRequest) []string {
	args := []string{req.Chain}
	if req.SrcIP != "" {
		args = append(args, "-s", req.SrcIP)
	}
	if req.Protocol != "" && req.Protocol != "all" {
		args = append(args, "-p", req.Protocol)
	}
	if req.DstPort > 0 {
		args = append(args, "--dport", strconv.Itoa(req.DstPort))
	}
	args = append(args, "-j", req.Target)
	return args
}

func execIptables(ctx context.Context, flag string, args []string) error {
	full := append([]string{flag}, args...)
	_, err := runCmd(ctx, "iptables", full...)
	return err
}

func validateIPorCIDR(s string) error {
	if _, _, err := net.ParseCIDR(s); err == nil {
		return nil
	}
	if net.ParseIP(s) != nil {
		return nil
	}
	return fmt.Errorf("invalid IP or CIDR: %q", s)
}
