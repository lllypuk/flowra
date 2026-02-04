package tag

import "strings"

const (
	// maxTagParts is the maximum number of parts when splitting tag (name and value)
	maxTagParts = 2
	// hashAndSpaceOffset is the offset for space and hash characters
	hashAndSpaceOffset = 2
)

// Parser parses tags from text messages
type Parser struct {
	knownTags map[string]Definition
}

// NewParser creates a new parser with registered system tags
//
//nolint:funlen // Function registers all system tags and needs to be comprehensive
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
		AllowedValues: nil, // depends on entity type, validated separately
		Validator:     noValidation,
	})

	parser.registerTag(Definition{
		Name:          "assignee",
		RequiresValue: false, // can be empty (remove assignee)
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
		RequiresValue: false, // can be empty (remove due_date)
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

	// Participant Management Tags (Task 007a)
	parser.registerTag(Definition{
		Name:          "invite",
		RequiresValue: true,
		ValueType:     ValueTypeUsername,
		Validator:     validateUsername,
	})

	parser.registerTag(Definition{
		Name:          "remove",
		RequiresValue: true,
		ValueType:     ValueTypeUsername,
		Validator:     validateUsername,
	})

	// Chat Lifecycle Tags (Task 007a)
	parser.registerTag(Definition{
		Name:          "close",
		RequiresValue: false,
		ValueType:     ValueTypeNone,
		Validator:     noValidation,
	})

	parser.registerTag(Definition{
		Name:          "reopen",
		RequiresValue: false,
		ValueType:     ValueTypeNone,
		Validator:     noValidation,
	})

	parser.registerTag(Definition{
		Name:          "delete",
		RequiresValue: false,
		ValueType:     ValueTypeNone,
		Validator:     noValidation,
	})

	return parser
}

// registerTag registers a tag definition
func (p *Parser) registerTag(def Definition) {
	p.knownTags[def.Name] = def
}

// isKnownTag checks if the tag is known
func (p *Parser) isKnownTag(name string) bool {
	_, exists := p.knownTags[name]
	return exists
}

// GetTagDefinition returns tag definition by name
func (p *Parser) GetTagDefinition(name string) (Definition, bool) {
	def, exists := p.knownTags[name]
	return def, exists
}

// Parse parses text message and extracts tags
// parsing rules:
// 1. Line starts with # → parse tags until end of line or text
// 2. Line does not start with # → regular text, tags not parsed
// 3. After regular text, new line with # → parse tags
// 4. Empty lines are ignored
func (p *Parser) Parse(content string) ParseResult {
	lines := strings.Split(content, "\n")
	result := ParseResult{Tags: []ParsedTag{}}

	inTagMode := true

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// skip empty lines
		if trimmed == "" {
			continue
		}

		// check if line starts with #
		if strings.HasPrefix(trimmed, "#") && (inTagMode || i > 0) {
			// parse tags on this line
			tags, remaining := p.parseTagsFromLine(trimmed)
			result.Tags = append(result.Tags, tags...)

			// if there is text after tags on the same line
			if remaining != "" {
				result.PlainText += remaining + "\n"
				inTagMode = false
			}
		} else {
			// regular text
			result.PlainText += line + "\n"
			inTagMode = false
		}
	}

	result.PlainText = strings.TrimSpace(result.PlainText)
	return result
}

// parseTagsFromLine parses all tags on one line
// returns list of parsed tags and remaining text
func (p *Parser) parseTagsFromLine(line string) ([]ParsedTag, string) {
	tags := []ParsedTag{}
	remaining := line

	for strings.HasPrefix(remaining, "#") {
		tag, rest := p.parseOneTag(remaining)

		if tag != nil {
			// check if tag is known
			if p.isKnownTag(tag.Key) {
				tags = append(tags, *tag)
			}
			// unknown tags are ignored (MVP), but still parsed
		}

		remaining = strings.TrimSpace(rest)

		// if remainder does not start with #, it is text - stop parsing tags
		// remaining will be returned as plaintext
		if remaining != "" && !strings.HasPrefix(remaining, "#") {
			break
		}
	}

	return tags, remaining
}

// parseOneTag parses one tag from beginning of string
// returns ParsedTag and remaining part of string
func (p *Parser) parseOneTag(s string) (*ParsedTag, string) {
	// s starts with #
	if !strings.HasPrefix(s, "#") {
		return nil, s
	}

	// remove # at the beginning
	withoutHash := s[1:]

	// find end of tag name (before space)
	parts := strings.SplitN(withoutHash, " ", maxTagParts)
	tagName := parts[0]

	// if tag is at end of line or next char is not space
	if len(parts) == 1 {
		return &ParsedTag{Key: tagName, Value: ""}, ""
	}

	rest := parts[1]

	// if # follows immediately after space, value is empty
	if strings.HasPrefix(rest, "#") {
		return &ParsedTag{Key: tagName, Value: ""}, rest
	}

	// Tag value is everything between tag name and next valid tag (or end of line)
	// Spec line 114: "tag value is everything between tag name and next #"
	// valid tag: space + # + lowercase letter (a-z)

	var value string
	var remaining string

	// search for next valid tag
	searchStart := 0
	for {
		nextHashIndex := strings.Index(rest[searchStart:], " #")
		if nextHashIndex == -1 {
			// no next #, everything else is value
			value = strings.TrimSpace(rest)
			remaining = ""
			break
		}

		// check that after # follows lowercase letter a-z (valid tag name)
		actualIndex := searchStart + nextHashIndex + hashAndSpaceOffset // +2 for space and #
		if actualIndex < len(rest) {
			nextChar := rest[actualIndex]
			// check that first character is Latin letter a-z
			if nextChar >= 'a' && nextChar <= 'z' {
				// this is a valid tag
				value = strings.TrimSpace(rest[:searchStart+nextHashIndex])
				remaining = strings.TrimSpace(rest[searchStart+nextHashIndex:])
				break
			}
		}

		// this is not a valid tag (e.g. #123), continue searching
		searchStart = actualIndex
		if searchStart >= len(rest) {
			// reached end of line
			value = strings.TrimSpace(rest)
			remaining = ""
			break
		}
	}

	return &ParsedTag{Key: tagName, Value: value}, remaining
}
