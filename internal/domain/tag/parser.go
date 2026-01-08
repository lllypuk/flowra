package tag

import "strings"

const (
	// maxTagParts is the maximum number of parts when splitting tag (name and value)
	maxTagParts = 2
	// hashAndSpaceOffset is the offset for space and hash characters
	hashAndSpaceOffset = 2
)

// Parser parsit tags from text messages
type Parser struct {
	knownTags map[string]Definition
}

// NewParser creates New parser s zaregistrirovannymi sistemnymi tegami
func NewParser() *Parser {
	parser := &Parser{
		knownTags: make(map[string]Definition),
	}

	// Entity Creation Tags
	parser.registerTag(Definition{
		Name:          "task",
		RequiresValue: true,
		ValueType:     ValueTypeString,
		Validator:     noValidation,
	})

	parser.registerTag(Definition{
		Name:          "bug",
		RequiresValue: true,
		ValueType:     ValueTypeString,
		Validator:     noValidation,
	})

	parser.registerTag(Definition{
		Name:          "epic",
		RequiresValue: true,
		ValueType:     ValueTypeString,
		Validator:     noValidation,
	})

	// Entity Management Tags
	parser.registerTag(Definition{
		Name:          "status",
		RequiresValue: true,
		ValueType:     ValueTypeEnum,
		AllowedValues: nil, // zavisit ot type entity, budet valid otdelno
		Validator:     noValidation,
	})

	parser.registerTag(Definition{
		Name:          "assignee",
		RequiresValue: false, // mozhet byt pustym (snyat assignee)
		ValueType:     ValueTypeUsername,
		Validator:     validateUsername,
	})

	parser.registerTag(Definition{
		Name:          "priority",
		RequiresValue: true,
		ValueType:     ValueTypeEnum,
		AllowedValues: []string{"High", "Medium", "Low"},
		Validator:     validatePriority,
	})

	parser.registerTag(Definition{
		Name:          "due",
		RequiresValue: false, // mozhet byt pustym (snyat due_date)
		ValueType:     ValueTypeDate,
		Validator:     validateISODate,
	})

	parser.registerTag(Definition{
		Name:          "title",
		RequiresValue: true,
		ValueType:     ValueTypeString,
		Validator:     noValidation,
	})

	// Bug-Specific Tags
	parser.registerTag(Definition{
		Name:          "severity",
		RequiresValue: true,
		ValueType:     ValueTypeEnum,
		AllowedValues: []string{"Critical", "Major", "Minor", "Trivial"},
		Validator:     validateSeverity,
	})

	return parser
}

// registerTag registriruet definition tega
func (p *Parser) registerTag(def Definition) {
	p.knownTags[def.Name] = def
}

// isKnownTag checks, is li teg izvestnym
func (p *Parser) isKnownTag(name string) bool {
	_, exists := p.knownTags[name]
	return exists
}

// GetTagDefinition returns definition tega po imeni
func (p *Parser) GetTagDefinition(name string) (Definition, bool) {
	def, exists := p.knownTags[name]
	return def, exists
}

// Parse parsit text messages and izvlekaet tags
// pravila parsinga:
// 1. String starts s # → parse tags before end stroki or before text
// 2. stroka does not start s # → regular text, tags not parsyatsya
// 3. after regular text New stroka s # → parse tags
// 4. pustye stroki ignoriruyutsya
func (p *Parser) Parse(content string) ParseResult {
	lines := strings.Split(content, "\n")
	result := ParseResult{Tags: []ParsedTag{}}

	inTagMode := true

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// pustye stroki propuskaem
		if trimmed == "" {
			continue
		}

		// Checking, nachinaetsya li stroka s #
		if strings.HasPrefix(trimmed, "#") && (inTagMode || i > 0) {
			// parsim tags on it is stroke
			tags, remaining := p.parseTagsFromLine(trimmed)
			result.Tags = append(result.Tags, tags...)

			// if est text after tegov on toy zhe stroke
			if remaining != "" {
				result.PlainText += remaining + "\n"
				inTagMode = false
			}
		} else {
			// obychnyy text
			result.PlainText += line + "\n"
			inTagMode = false
		}
	}

	result.PlainText = strings.TrimSpace(result.PlainText)
	return result
}

// parseTagsFromLine parsit all tags on one stroke
// returns list rasparsennyh tegov and ostavshiysya text
func (p *Parser) parseTagsFromLine(line string) ([]ParsedTag, string) {
	tags := []ParsedTag{}
	remaining := line

	for strings.HasPrefix(remaining, "#") {
		tag, rest := p.parseOneTag(remaining)

		if tag != nil {
			// Checking, is li teg izvestnym
			if p.isKnownTag(tag.Key) {
				tags = append(tags, *tag)
			}
			// neizvestnye tags ignoriruyutsya (MVP), no tozhe parsyatsya
		}

		remaining = strings.TrimSpace(rest)

		// if ostatok does not start s #, it is text - prekraschaem parsing tegov
		// Remaining budet vozvraschen as plaintext
		if remaining != "" && !strings.HasPrefix(remaining, "#") {
			break
		}
	}

	return tags, remaining
}

// parseOneTag parsit one teg from nachala stroki
// returns ParsedTag and ostavshuyusya chast stroki
func (p *Parser) parseOneTag(s string) (*ParsedTag, string) {
	// s nachinaetsya s #
	if !strings.HasPrefix(s, "#") {
		return nil, s
	}

	// ubiraem # in nachale
	withoutHash := s[1:]

	// nahodim konets imeni tega (before probela)
	parts := strings.SplitN(withoutHash, " ", maxTagParts)
	tagName := parts[0]

	// if teg in kontse stroki or next simvol not probel
	if len(parts) == 1 {
		return &ParsedTag{Key: tagName, Value: ""}, ""
	}

	rest := parts[1]

	// if srazu after probela idyot #, to value pustoe
	if strings.HasPrefix(rest, "#") {
		return &ParsedTag{Key: tagName, Value: ""}, rest
	}

	// Value tega - it is vsyo between name tega and sleduyuschim valid tegom (or kontsom stroki)
	// Spec stroka 114: "value tega — it is vsyo between name tega and sleduyuschim #"
	// validnyy teg: probel + # + lowercase bukva (a-z)

	var value string
	var remaining string

	// ischem next valid teg
	searchStart := 0
	for {
		nextHashIndex := strings.Index(rest[searchStart:], " #")
		if nextHashIndex == -1 {
			// no sleduyuschego #, vsyo ostalnoe — value
			value = strings.TrimSpace(rest)
			remaining = ""
			break
		}

		// Checking, that after # idet lowercase bukva a-z (valid imya tega)
		actualIndex := searchStart + nextHashIndex + hashAndSpaceOffset // +2 for probela and #
		if actualIndex < len(rest) {
			nextChar := rest[actualIndex]
			// Checking that first simvol - latinskaya bukva a-z
			if nextChar >= 'a' && nextChar <= 'z' {
				// eto valid teg
				value = strings.TrimSpace(rest[:searchStart+nextHashIndex])
				remaining = strings.TrimSpace(rest[searchStart+nextHashIndex:])
				break
			}
		}

		// eto not valid teg (naprimer #123), prodolzhaem search
		searchStart = actualIndex
		if searchStart >= len(rest) {
			// dostigli end stroki
			value = strings.TrimSpace(rest)
			remaining = ""
			break
		}
	}

	return &ParsedTag{Key: tagName, Value: value}, remaining
}
