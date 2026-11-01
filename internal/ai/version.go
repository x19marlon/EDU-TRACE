package ai

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

func promptVersion(syllabus string) string {
	h := sha256.New()
	for _, part := range []string{systemPromptFormal, systemPromptInformal, systemPromptChat, systemPromptReview,
		systemPromptAnalysis, verifyBlock, rulesBlock, string(reviewSchema), strings.Join(sectionTitles[:], "|"), syllabus} {
		h.Write([]byte(part))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))[:8]
}

func (o *OllamaClient) Model() string { return o.cfg.OllamaModel }

func (o *OllamaClient) PromptVersion() string { return o.prompt }
