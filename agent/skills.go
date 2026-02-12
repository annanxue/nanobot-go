package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/nanobotgo/utils"
)

type SkillInfo struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	Source string `json:"source"`
}

type SkillsLoader struct {
	workspace       string
	workspaceSkills string
	builtinSkills   string
}

func NewSkillsLoader(workspace string) *SkillsLoader {
	return &SkillsLoader{
		workspace:       workspace,
		workspaceSkills: filepath.Join(workspace, "skills"),
		builtinSkills:   filepath.Join(filepath.Dir(utils.GetDataPath()), "skills"),
	}
}

func (sl *SkillsLoader) ListSkills(filterUnavailable bool) []SkillInfo {
	var skills []SkillInfo

	if _, err := os.Stat(sl.workspaceSkills); err == nil {
		entries, _ := os.ReadDir(sl.workspaceSkills)
		for _, entry := range entries {
			if entry.IsDir() {
				skillFile := filepath.Join(sl.workspaceSkills, entry.Name(), "SKILL.md")
				if _, err := os.Stat(skillFile); err == nil {
					skills = append(skills, SkillInfo{
						Name:   entry.Name(),
						Path:   skillFile,
						Source: "workspace",
					})
				}
			}
		}
	}

	if _, err := os.Stat(sl.builtinSkills); err == nil {
		entries, _ := os.ReadDir(sl.builtinSkills)
		for _, entry := range entries {
			if entry.IsDir() {
				skillFile := filepath.Join(sl.builtinSkills, entry.Name(), "SKILL.md")
				if _, err := os.Stat(skillFile); err == nil {
					exists := false
					for _, s := range skills {
						if s.Name == entry.Name() {
							exists = true
							break
						}
					}
					if !exists {
						skills = append(skills, SkillInfo{
							Name:   entry.Name(),
							Path:   skillFile,
							Source: "builtin",
						})
					}
				}
			}
		}
	}

	if filterUnavailable {
		var filtered []SkillInfo
		for _, s := range skills {
			meta := sl.getSkillMeta(s.Name)
			if sl.checkRequirements(meta) {
				filtered = append(filtered, s)
			}
		}
		return filtered
	}

	return skills
}

func (sl *SkillsLoader) LoadSkill(name string) string {
	workspaceSkill := filepath.Join(sl.workspaceSkills, name, "SKILL.md")
	if _, err := os.Stat(workspaceSkill); err == nil {
		content, _ := os.ReadFile(workspaceSkill)
		return string(content)
	}

	if _, err := os.Stat(sl.builtinSkills); err == nil {
		builtinSkill := filepath.Join(sl.builtinSkills, name, "SKILL.md")
		if _, err := os.Stat(builtinSkill); err == nil {
			content, _ := os.ReadFile(builtinSkill)
			return string(content)
		}
	}

	return ""
}

func (sl *SkillsLoader) LoadSkillsForContext(skillNames []string) string {
	var parts []string
	for _, name := range skillNames {
		content := sl.LoadSkill(name)
		if content != "" {
			content = sl.stripFrontmatter(content)
			parts = append(parts, fmt.Sprintf("### Skill: %s\n\n%s", name, content))
		}
	}
	return strings.Join(parts, "\n\n---\n\n")
}

func (sl *SkillsLoader) BuildSkillsSummary() string {
	allSkills := sl.ListSkills(false)
	if len(allSkills) == 0 {
		return ""
	}

	var lines []string
	lines = append(lines, "<skills>")

	for _, s := range allSkills {
		name := escapeXML(s.Name)
		path := s.Path
		desc := escapeXML(sl.getSkillDescription(s.Name))
		skillMeta := sl.getSkillMeta(s.Name)
		available := sl.checkRequirements(skillMeta)

		lines = append(lines, fmt.Sprintf(`  <skill available="%t">`, available))
		lines = append(lines, fmt.Sprintf(`    <name>%s</name>`, name))
		lines = append(lines, fmt.Sprintf(`    <description>%s</description>`, desc))
		lines = append(lines, fmt.Sprintf(`    <location>%s</location>`, path))

		if !available {
			missing := sl.getMissingRequirements(skillMeta)
			if missing != "" {
				lines = append(lines, fmt.Sprintf(`    <requires>%s</requires>`, escapeXML(missing)))
			}
		}

		lines = append(lines, "  </skill>")
	}

	lines = append(lines, "</skills>")
	return strings.Join(lines, "\n")
}

func (sl *SkillsLoader) getMissingRequirements(skillMeta map[string]interface{}) string {
	var missing []string

	requires, ok := skillMeta["requires"].(map[string]interface{})
	if !ok {
		return ""
	}

	bins, _ := requires["bins"].([]interface{})
	for _, b := range bins {
		if bin, ok := b.(string); ok {
			if _, err := os.Stat(bin); os.IsNotExist(err) {
				missing = append(missing, "CLI: "+bin)
			}
		}
	}

	envs, _ := requires["env"].([]interface{})
	for _, e := range envs {
		if env, ok := e.(string); ok {
			if os.Getenv(env) == "" {
				missing = append(missing, "ENV: "+env)
			}
		}
	}

	return strings.Join(missing, ", ")
}

func (sl *SkillsLoader) getSkillDescription(name string) string {
	meta := sl.GetSkillMetadata(name)
	if desc, ok := meta["description"].(string); ok && desc != "" {
		return desc
	}
	return name
}

func (sl *SkillsLoader) stripFrontmatter(content string) string {
	if strings.HasPrefix(content, "---") {
		re := regexp.MustCompile(`(?s)^---\n.*?\n---\n`)
		match := re.FindString(content)
		if match != "" {
			return strings.TrimSpace(content[len(match):])
		}
	}
	return content
}

func (sl *SkillsLoader) parseNanobotMetadata(raw string) map[string]interface{} {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		return map[string]interface{}{}
	}

	if nanobot, ok := data["nanobot"].(map[string]interface{}); ok {
		return nanobot
	}
	return map[string]interface{}{}
}

func (sl *SkillsLoader) checkRequirements(skillMeta map[string]interface{}) bool {
	requires, ok := skillMeta["requires"].(map[string]interface{})
	if !ok {
		return true
	}

	bins, _ := requires["bins"].([]interface{})
	for _, b := range bins {
		if bin, ok := b.(string); ok {
			if _, err := os.Stat(bin); os.IsNotExist(err) {
				return false
			}
		}
	}

	envs, _ := requires["env"].([]interface{})
	for _, e := range envs {
		if env, ok := e.(string); ok {
			if os.Getenv(env) == "" {
				return false
			}
		}
	}

	return true
}

func (sl *SkillsLoader) getSkillMeta(name string) map[string]interface{} {
	meta := sl.GetSkillMetadata(name)
	if metadata, ok := meta["metadata"].(string); ok {
		return sl.parseNanobotMetadata(metadata)
	}
	return map[string]interface{}{}
}

func (sl *SkillsLoader) GetAlwaysSkills() []string {
	var result []string
	for _, s := range sl.ListSkills(true) {
		meta := sl.GetSkillMetadata(s.Name)
		skillMeta := map[string]interface{}{}
		if metadata, ok := meta["metadata"].(string); ok {
			skillMeta = sl.parseNanobotMetadata(metadata)
		}

		if always, ok := skillMeta["always"].(bool); ok && always {
			result = append(result, s.Name)
		} else if always, ok := meta["always"].(bool); ok && always {
			result = append(result, s.Name)
		}
	}
	return result
}

func (sl *SkillsLoader) GetSkillMetadata(name string) map[string]interface{} {
	content := sl.LoadSkill(name)
	if content == "" {
		return map[string]interface{}{}
	}

	if strings.HasPrefix(content, "---") {
		re := regexp.MustCompile(`(?s)^---\n(.*?)\n---`)
		match := re.FindStringSubmatch(content)
		if len(match) > 1 {
			metadata := make(map[string]interface{})
			lines := strings.Split(match[1], "\n")
			for _, line := range lines {
				if strings.Contains(line, ":") {
					parts := strings.SplitN(line, ":", 2)
					if len(parts) == 2 {
						key := strings.TrimSpace(parts[0])
						value := strings.TrimSpace(parts[1])
						value = strings.Trim(value, "\"'")
						metadata[key] = value
					}
				}
			}
			return metadata
		}
	}

	return map[string]interface{}{}
}

func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}
