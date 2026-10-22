package saasai

import (
	"github.com/greennodehub/greennode-cli/internal/cli"
	opengine "github.com/greennodehub/greennode-cli/internal/operation"
	"github.com/spf13/cobra"
)

const endpoint = "https://ai-speech-text.api.vngcloud.vn/speech-api"

var SaaSAICmd = newSaaSAICommand()

func newSaaSAICommand() *cobra.Command {
	root := opengine.NewGroup("saas-ai", "Transcribe speech and synthesize audio")
	root.Long = `Use the GreenNode SaaS AI Speech Text API for synchronous speech transcription and synthesis.

Both operations are billable remote requests and support an offline --dry-run.
Speech synthesis writes the returned audio only to an explicit output file.`
	speechToText := opengine.NewGroup("speech-to-text", "Transcribe an audio file")
	textToSpeech := opengine.NewGroup("text-to-speech", "Synthesize speech from text")
	speechToText.AddCommand(newSpeechToTextCommand())
	textToSpeech.AddCommand(newTextToSpeechCommand())
	root.AddCommand(speechToText, textToSpeech)
	return root
}

func init() {
	cli.RegisterService(SaaSAICmd)
}
