package pinyin

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConvert(t *testing.T) {
	args := Args{StrictMatch: true}
	s1 := textToSentence(args, "")
	assert.Empty(t, s1.terms)

	s1 = textToSentence(args, "hello ")
	assert.Equal(t, 1, len(s1.terms))
	assert.Equal(t, "hello", string(s1.terms[0].alias[0]))

	s1 = textToSentence(args, "hello ,中国")
	assert.Equal(t, 3, len(s1.terms))
	assert.Equal(t, "hello", string(s1.terms[0].alias[0]))
	assert.Equal(t, "zhong", string(s1.terms[1].alias[0]))
	assert.Equal(t, "guo", string(s1.terms[2].alias[0]))

	s1 = textToSentence(args, "重要,OK?")
	assert.Equal(t, 3, len(s1.terms))
	var alias []string
	for _, w := range s1.terms[0].alias {
		alias = append(alias, string(w))
	}
	assert.ElementsMatch(t, []string{"zhong", "chong", "tong"}, alias)
	assert.Equal(t, "yao", string(s1.terms[1].alias[0]))
	assert.Equal(t, "ok", string(s1.terms[2].alias[0]))

	args.StrictMatch = false
	s1 = textToSentence(args, "重要,OK?")
	assert.Equal(t, 3, len(s1.terms))
	alias = nil
	for _, w := range s1.terms[0].alias {
		alias = append(alias, string(w))
	}
	assert.ElementsMatch(t, []string{"zhong", "chong", "tong", "zh", "z", "ch", "c", "t"}, alias)
	assert.Equal(t, "yao", string(s1.terms[1].alias[0]))
	assert.Equal(t, "ok", string(s1.terms[2].alias[0]))

	s1 = textToSentence(Args{StrictMatch: true}, "hello 100，总共100人")
	assert.Equal(t, 6, len(s1.terms))
	assert.Equal(t, "hello", string(s1.terms[0].alias[0]))
	assert.Equal(t, "100", string(s1.terms[1].alias[0]))
	assert.Equal(t, "zong", string(s1.terms[2].alias[0]))
	assert.Equal(t, "共", s1.getRaw(s1.terms[3]))
	assert.Equal(t, "gong", string(s1.terms[3].alias[0]))
	assert.Equal(t, "100", string(s1.terms[4].alias[0]))
	assert.Equal(t, "ren", string(s1.terms[5].alias[0]))

}

func TestMatch(t *testing.T) {
	var substr string
	var idx int
	substr, idx = Contain(`hello, 世界`, "shijie")
	assert.NotEqual(t, -1, idx)
	assert.Equal(t, "世界", substr)

	substr, idx = FuzzyContain(`hello, 世界`, "sj")
	assert.NotEqual(t, -1, idx)
	assert.Equal(t, "世界", substr)

	substr, idx = FuzzyContain(`hello, 世界,`, "hellosj")
	assert.NotEqual(t, -1, idx)
	assert.Equal(t, "hello, 世界", substr)

	substr, idx = FuzzyContain(`hello, 世界,`, "hello世j")
	assert.NotEqual(t, -1, idx)
	assert.Equal(t, "hello, 世界", substr)

	substr, idx = FuzzyContain(`不允许,`, "yx")
	assert.NotEqual(t, -1, idx)
	assert.Equal(t, "允许", substr)

	substr, idx = FuzzyContain(`网站,`, "wz")
	assert.NotEqual(t, -1, idx)
	assert.Equal(t, "网站", substr)
}
