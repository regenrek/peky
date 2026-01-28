package skills

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/regenrek/peakypanes/internal/cli/output"
	"github.com/regenrek/peakypanes/internal/cli/root"
	"github.com/regenrek/peakypanes/internal/skills"
)

// Register registers skills handlers.
func Register(reg *root.Registry) {
	reg.Register("skills", runList)
	reg.Register("skills.list", runList)
	reg.Register("skills.install", runInstall)
	reg.Register("skills.status", runStatus)
	reg.Register("skills.uninstall", runUninstall)
}

func runList(ctx root.CommandContext) error {
	start := time.Now()
	meta := output.NewMeta("skills.list", ctx.Deps.Version)
	bundle, err := skills.LoadBundle()
	if err != nil {
		return err
	}
	targets, err := parseTargetsDefaultAll(ctx.Cmd.StringSlice("target"))
	if err != nil {
		return err
	}
	status, err := skills.Status(bundle, targets, "")
	if err != nil {
		return err
	}
	list := buildSkillSummaries(bundle, status)
	if ctx.JSON {
		meta = output.WithDuration(meta, start)
		return output.WriteSuccess(ctx.Out, meta, output.SkillsListResponse{Skills: list})
	}
	for _, item := range list {
		if err := writeSkillLine(ctx, item); err != nil {
			return err
		}
	}
	return nil
}

func runStatus(ctx root.CommandContext) error {
	start := time.Now()
	meta := output.NewMeta("skills.status", ctx.Deps.Version)
	bundle, err := skills.LoadBundle()
	if err != nil {
		return err
	}
	targets, err := parseTargetsDefaultAll(ctx.Cmd.StringSlice("target"))
	if err != nil {
		return err
	}
	status, err := skills.Status(bundle, targets, ctx.Cmd.String("dest"))
	if err != nil {
		return err
	}
	list := buildSkillSummaries(bundle, status)
	if ctx.JSON {
		meta = output.WithDuration(meta, start)
		return output.WriteSuccess(ctx.Out, meta, output.SkillsListResponse{Skills: list})
	}
	for _, item := range list {
		if err := writeSkillLine(ctx, item); err != nil {
			return err
		}
	}
	return nil
}

func runInstall(ctx root.CommandContext) error {
	start := time.Now()
	meta := output.NewMeta("skills.install", ctx.Deps.Version)
	bundle, err := skills.LoadBundle()
	if err != nil {
		return err
	}
	targets, err := parseTargetsRequired(ctx.Cmd.StringSlice("target"))
	if err != nil {
		return err
	}
	mode := skills.InstallMode(strings.TrimSpace(ctx.Cmd.String("mode")))
	result, err := skills.Install(bundle, skills.InstallOptions{
		Targets:      targets,
		SkillIDs:     ctx.Cmd.StringSlice("skill"),
		Mode:         mode,
		DestOverride: ctx.Cmd.String("dest"),
		Overwrite:    ctx.Cmd.Bool("overwrite"),
	})
	if err != nil {
		return err
	}
	records := mapInstallRecords(result.Records)
	if ctx.JSON {
		meta = output.WithDuration(meta, start)
		return output.WriteSuccess(ctx.Out, meta, output.SkillsInstallResponse{Records: records})
	}
	for _, record := range records {
		if err := writeLine(ctx.Out, formatRecord(record)); err != nil {
			return err
		}
	}
	return nil
}

func runUninstall(ctx root.CommandContext) error {
	start := time.Now()
	meta := output.NewMeta("skills.uninstall", ctx.Deps.Version)
	bundle, err := skills.LoadBundle()
	if err != nil {
		return err
	}
	targets, err := parseTargetsRequired(ctx.Cmd.StringSlice("target"))
	if err != nil {
		return err
	}
	result, err := skills.Uninstall(bundle, skills.UninstallOptions{
		Targets:      targets,
		SkillIDs:     ctx.Cmd.StringSlice("skill"),
		DestOverride: ctx.Cmd.String("dest"),
	})
	if err != nil {
		return err
	}
	records := mapUninstallRecords(result.Records)
	if ctx.JSON {
		meta = output.WithDuration(meta, start)
		return output.WriteSuccess(ctx.Out, meta, output.SkillsUninstallResponse{Records: records})
	}
	for _, record := range records {
		if err := writeLine(ctx.Out, formatRecord(record)); err != nil {
			return err
		}
	}
	return nil
}

func parseTargetsDefaultAll(values []string) ([]skills.Target, error) {
	if len(values) == 0 {
		return skills.Targets(), nil
	}
	return skills.ParseTargets(values)
}

func parseTargetsRequired(values []string) ([]skills.Target, error) {
	if len(values) == 0 {
		return nil, fmt.Errorf("target is required")
	}
	return skills.ParseTargets(values)
}

func buildSkillSummaries(bundle skills.Bundle, status []skills.StatusRecord) []output.SkillSummary {
	statusBySkill := make(map[string][]skills.StatusRecord)
	for _, record := range status {
		statusBySkill[record.SkillID] = append(statusBySkill[record.SkillID], record)
	}
	out := make([]output.SkillSummary, 0, len(bundle.Skills))
	for _, skill := range bundle.Skills {
		targets := make([]string, 0, len(skill.Targets))
		for _, target := range skill.Targets {
			targets = append(targets, string(target))
		}
		sort.Strings(targets)
		summary := output.SkillSummary{
			ID:          skill.ID,
			Name:        skill.Name,
			Description: skill.Description,
			Targets:     targets,
		}
		if records := statusBySkill[skill.ID]; len(records) > 0 {
			summary.Status = mapStatusRecords(records)
		}
		out = append(out, summary)
	}
	return out
}

func mapStatusRecords(records []skills.StatusRecord) []output.SkillTargetStatus {
	out := make([]output.SkillTargetStatus, 0, len(records))
	for _, record := range records {
		out = append(out, output.SkillTargetStatus{
			Target:   string(record.Target),
			Path:     record.Path,
			Present:  record.Present,
			Matches:  record.Matches,
			ErrorMsg: record.ErrorMsg,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Target < out[j].Target })
	return out
}

func mapInstallRecords(records []skills.InstallRecord) []output.SkillsInstallRecord {
	out := make([]output.SkillsInstallRecord, 0, len(records))
	for _, record := range records {
		out = append(out, output.SkillsInstallRecord{
			SkillID: record.SkillID,
			Target:  string(record.Target),
			Path:    record.Path,
			Status:  record.Status,
			Message: record.Message,
		})
	}
	return out
}

func mapUninstallRecords(records []skills.UninstallRecord) []output.SkillsInstallRecord {
	out := make([]output.SkillsInstallRecord, 0, len(records))
	for _, record := range records {
		out = append(out, output.SkillsInstallRecord{
			SkillID: record.SkillID,
			Target:  string(record.Target),
			Path:    record.Path,
			Status:  record.Status,
			Message: record.Message,
		})
	}
	return out
}

func writeSkillLine(ctx root.CommandContext, item output.SkillSummary) error {
	line := item.ID
	if strings.TrimSpace(item.Name) != "" {
		line = fmt.Sprintf("%s (%s)", line, item.Name)
	}
	if len(item.Targets) > 0 {
		line = fmt.Sprintf("%s targets=%s", line, strings.Join(item.Targets, ","))
	}
	return writeLine(ctx.Out, line)
}

func formatRecord(record output.SkillsInstallRecord) string {
	parts := []string{record.SkillID, record.Target, record.Status}
	if strings.TrimSpace(record.Path) != "" {
		parts = append(parts, record.Path)
	}
	if strings.TrimSpace(record.Message) != "" {
		parts = append(parts, record.Message)
	}
	return strings.Join(parts, " ")
}

func writeLine(out io.Writer, line string) error {
	_, err := fmt.Fprintln(out, line)
	return err
}
