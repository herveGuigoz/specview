package catalog

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// NextAction is guidance to copy, never a command to execute in the viewer.
type NextAction struct {
	Label   string `json:"label"`
	Command string `json:"command"`
	Target  string `json:"target"`
	Reason  string `json:"reason"`
}

type changeMetadata struct {
	Schema    string `yaml:"schema"`
	SkipSpecs bool   `yaml:"skip_specs"`
}

func suggestNext(e Entry, schema string, skipSpecs bool, uncertain bool) *NextAction {
	if e.Category != "changes" {
		return nil
	}
	status := func(reason string) *NextAction {
		return &NextAction{Label: "Vérifier la prochaine étape", Command: "openspec status --change " + shellQuote(e.Name), Target: "terminal", Reason: reason}
	}
	if uncertain || schema != "spec-driven" {
		return status("Consulter le workflow du changement pour déterminer la prochaine étape.")
	}
	has := func(kind string) bool {
		for _, d := range e.Documents {
			if d.Kind == kind && d.Error == "" {
				return true
			}
		}
		return false
	}
	instructions := func(artifact, label, reason string) *NextAction {
		return &NextAction{Label: label, Command: fmt.Sprintf("openspec instructions %s --change %s", artifact, shellQuote(e.Name)), Target: "terminal", Reason: reason}
	}
	for _, d := range e.Documents {
		if d.Error != "" {
			return status("Un artefact est illisible ; vérifier l’état du changement.")
		}
	}
	if !has("proposal") {
		return instructions("proposal", "Préparer la proposition", "La proposition n’est pas encore présente.")
	}
	if !skipSpecs && !has("delta") {
		return instructions("specs", "Compléter les spécifications", "Aucune spécification delta n’est présente.")
	}
	// Design is conditional in spec-driven: its absence alone cannot block apply.
	if !has("tasks") {
		return instructions("tasks", "Préparer les tâches", "Le fichier de tâches n’est pas encore présent.")
	}
	if e.Progress.Total == 0 {
		return status("Aucune tâche mesurable ; vérifier les instructions du workflow.")
	}
	if e.Progress.Done < e.Progress.Total {
		return &NextAction{Label: "Poursuivre l’implémentation", Command: "/openspec-apply-change " + agentArgument(e.Name), Target: "agent", Reason: "Des tâches restent à réaliser."}
	}
	return &NextAction{Label: "Vérifier et archiver", Command: "/openspec-archive-change " + agentArgument(e.Name), Target: "agent", Reason: "Toutes les tâches sont cochées. Vérifier le changement avant de l’archiver."}
}
func shellQuote(value string) string { return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'" }
func agentArgument(value string) string {
	if value != "" && strings.IndexFunc(value, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '-' && r != '_' }) < 0 {
		return value
	}
	return strconv.Quote(value)
}
