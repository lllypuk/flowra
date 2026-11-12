package leadfeed_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lllypuk/flowra/internal/application/leadfeed"
)

func TestAnswerFingerprint(t *testing.T) {
	t.Parallel()

	a, b := "a", "b"
	base := leadfeed.AnswerFingerprint("Q-0001", &a, "text")
	assert.Equal(t, base, leadfeed.AnswerFingerprint("Q-0001", &a, "text"))
	assert.Len(t, base, 64)

	for name, other := range map[string]string{
		"qid":         leadfeed.AnswerFingerprint("Q-0002", &a, "text"),
		"option":      leadfeed.AnswerFingerprint("Q-0001", &b, "text"),
		"no option":   leadfeed.AnswerFingerprint("Q-0001", nil, "text"),
		"text":        leadfeed.AnswerFingerprint("Q-0001", &a, "other"),
		"field shift": leadfeed.AnswerFingerprint("Q-000", &a, "1text"),
	} {
		assert.NotEqual(t, base, other, name)
	}
	empty := ""
	assert.NotEqual(t,
		leadfeed.AnswerFingerprint("Q-0001", nil, "x"),
		leadfeed.AnswerFingerprint("Q-0001", &empty, "x"), "absent and empty option differ")
}
