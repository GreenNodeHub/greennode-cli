package saasai

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/client"
	"github.com/spf13/cobra"
)

const (
	speechToTextPath = "/api/v1/speechtotext/sync"
	textToSpeechPath = "/api/v1/texttospeech/sync"
)

type saasAIAPI interface {
	RequestBytes(string, string, map[string]string, []byte, string) (client.BytesResponse, error)
	RequestStream(string, string, map[string]string, io.Reader, string) (client.BytesResponse, error)
}

type clientFactory func(*cobra.Command) (saasAIAPI, error)

var newClient clientFactory = func(cmd *cobra.Command) (saasAIAPI, error) {
	return cli.NewClientWithEndpoint(cmd, endpoint)
}

var _ saasAIAPI = (*client.GreennodeClient)(nil)

func newSpeechToTextCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "transcribe",
		Short: "Transcribe an audio file synchronously",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			audioFile, encodingType, err := speechToTextInput(cmd)
			if err != nil {
				return err
			}
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			if dryRun {
				cli.PrintDryRun("transcribe", "SaaS AI speech-to-text request", map[string]any{"method": http.MethodPost, "path": speechToTextPath, "audio_file": audioFile, "encoding_type": encodingType})
				return nil
			}
			body, contentType, err := speechToTextMultipart(audioFile, encodingType)
			if err != nil {
				return err
			}
			defer body.Close()
			apiClient, err := newClient(cmd)
			if err != nil {
				return err
			}
			response, err := apiClient.RequestStream(http.MethodPost, speechToTextPath, nil, body, contentType)
			if err != nil {
				return err
			}
			if err := validateResponse(response, "application/json"); err != nil {
				return err
			}
			var result any
			if err := json.Unmarshal(response.Data, &result); err != nil {
				return fmt.Errorf("failed to parse SaaS AI speech-to-text response JSON: %w", err)
			}
			if _, ok := result.(map[string]any); !ok {
				return errors.New("SaaS AI speech-to-text response must be a JSON object")
			}
			return cli.Output(cmd, result)
		},
	}
	cmd.Flags().String("audio-file", "", "Path to a WAV or MP3 audio file")
	cmd.Flags().String("encoding-type", "", "Audio encoding type (wav or mp3)")
	cmd.Flags().Bool("dry-run", false, "Preview the validated request without calling the API")
	_ = cmd.MarkFlagRequired("audio-file")
	_ = cmd.MarkFlagRequired("encoding-type")
	return cmd
}

func newTextToSpeechCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "synthesize",
		Short: "Synthesize an audio file synchronously",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			body, outputFile, err := textToSpeechInput(cmd)
			if err != nil {
				return err
			}
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			if dryRun {
				cli.PrintDryRun("synthesize", "SaaS AI text-to-speech request", map[string]any{"method": http.MethodPost, "path": textToSpeechPath, "body": body, "output_file": outputFile})
				return nil
			}
			force, _ := cmd.Flags().GetBool("force")
			overwrite := force
			if exists, err := outputExists(outputFile); err != nil {
				return err
			} else if exists {
				if !cli.Confirm(force, fmt.Sprintf("Overwrite existing output file %s?", outputFile)) {
					return cli.ConfirmationError()
				}
				overwrite = true
			}
			apiClient, err := newClient(cmd)
			if err != nil {
				return err
			}
			encodedBody, err := json.Marshal(body)
			if err != nil {
				return fmt.Errorf("failed to encode SaaS AI text-to-speech request: %w", err)
			}
			response, err := apiClient.RequestBytes(http.MethodPost, textToSpeechPath, nil, encodedBody, "application/json")
			if err != nil {
				return err
			}
			if err := validateResponse(response, "audio/mpeg"); err != nil {
				return err
			}
			if err := writeAudio(outputFile, response.Data, overwrite); err != nil {
				return fmt.Errorf("write synthesized audio to %s: %w", outputFile, err)
			}
			return cli.Output(cmd, map[string]any{"output_file": outputFile, "bytes": len(response.Data), "content_type": response.ContentType})
		},
	}
	cmd.Flags().String("input", "", "Text to synthesize")
	cmd.Flags().Float64("speed", 0, "Speech speed (0.8 to 1.2)")
	cmd.Flags().Int("speaker-id", 0, "Speaker ID: 0 SouthWomen, 1 NorthemWomen, 2 SouthMen, 3 NorthemMen")
	cmd.Flags().Int("encode-type", 0, "Audio encoding: 0 wav, 1 mp3")
	cmd.Flags().String("output-file", "", "Path for the generated audio file")
	cmd.Flags().Bool("dry-run", false, "Preview the validated request without calling the API or writing a file")
	cmd.Flags().Bool("force", false, "Skip confirmation before replacing an existing output file")
	_ = cmd.MarkFlagRequired("input")
	_ = cmd.MarkFlagRequired("output-file")
	return cmd
}

func speechToTextInput(cmd *cobra.Command) (string, string, error) {
	audioFile, _ := cmd.Flags().GetString("audio-file")
	if audioFile == "" {
		return "", "", errors.New("audio-file must not be empty")
	}
	info, err := os.Stat(audioFile)
	if err != nil {
		return "", "", fmt.Errorf("audio-file %s: %w", audioFile, err)
	}
	if !info.Mode().IsRegular() {
		return "", "", fmt.Errorf("audio-file %s must be a regular file", audioFile)
	}
	encodingType, _ := cmd.Flags().GetString("encoding-type")
	if encodingType != "wav" && encodingType != "mp3" {
		return "", "", fmt.Errorf("encoding-type must be wav or mp3, got %q", encodingType)
	}
	return audioFile, encodingType, nil
}

func speechToTextMultipart(audioFile, encodingType string) (io.ReadCloser, string, error) {
	file, err := os.Open(audioFile)
	if err != nil {
		return nil, "", fmt.Errorf("open audio-file %s: %w", audioFile, err)
	}

	bodyReader, bodyWriter := io.Pipe()
	writer := multipart.NewWriter(bodyWriter)
	go func() {
		defer file.Close()
		if err := writeSpeechToTextMultipart(writer, file, audioFile, encodingType); err != nil {
			_ = bodyWriter.CloseWithError(err)
			return
		}
		_ = bodyWriter.Close()
	}()
	return bodyReader, writer.FormDataContentType(), nil
}

func writeSpeechToTextMultipart(writer *multipart.Writer, file *os.File, audioFile, encodingType string) error {
	if err := writer.WriteField("encoding_type", encodingType); err != nil {
		return fmt.Errorf("encode speech-to-text encoding type: %w", err)
	}
	part, err := writer.CreateFormFile("audio_file", filepath.Base(audioFile))
	if err != nil {
		return fmt.Errorf("create speech-to-text audio part: %w", err)
	}
	if _, err := io.Copy(part, file); err != nil {
		return fmt.Errorf("read audio-file %s: %w", audioFile, err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("finish speech-to-text multipart request: %w", err)
	}
	return nil
}

func textToSpeechInput(cmd *cobra.Command) (map[string]any, string, error) {
	input, _ := cmd.Flags().GetString("input")
	if strings.TrimSpace(input) == "" {
		return nil, "", errors.New("input must not be empty")
	}
	outputFile, _ := cmd.Flags().GetString("output-file")
	if outputFile == "" {
		return nil, "", errors.New("output-file must not be empty")
	}
	if info, err := os.Lstat(outputFile); err == nil && !info.Mode().IsRegular() {
		return nil, "", fmt.Errorf("output-file %s must be a regular file", outputFile)
	} else if err != nil && !os.IsNotExist(err) {
		return nil, "", fmt.Errorf("inspect output-file %s: %w", outputFile, err)
	}
	if dir := filepath.Dir(outputFile); dir != "." {
		if info, err := os.Stat(dir); err != nil {
			return nil, "", fmt.Errorf("output directory %s: %w", dir, err)
		} else if !info.IsDir() {
			return nil, "", fmt.Errorf("output directory %s is not a directory", dir)
		}
	}

	body := map[string]any{"input": input}
	if cmd.Flags().Changed("speed") {
		speed, _ := cmd.Flags().GetFloat64("speed")
		if math.IsNaN(speed) || math.IsInf(speed, 0) || speed < 0.8 || speed > 1.2 {
			return nil, "", fmt.Errorf("speed must be between 0.8 and 1.2, got %v", speed)
		}
		body["speed"] = speed
	}
	if cmd.Flags().Changed("speaker-id") {
		speakerID, _ := cmd.Flags().GetInt("speaker-id")
		if speakerID < 0 || speakerID > 3 {
			return nil, "", fmt.Errorf("speaker-id must be between 0 and 3, got %d", speakerID)
		}
		body["speaker_id"] = speakerID
	}
	if cmd.Flags().Changed("encode-type") {
		encodeType, _ := cmd.Flags().GetInt("encode-type")
		if encodeType != 0 && encodeType != 1 {
			return nil, "", fmt.Errorf("encode-type must be 0 (wav) or 1 (mp3), got %d", encodeType)
		}
		body["encode_type"] = encodeType
	}
	return body, outputFile, nil
}

func outputExists(path string) (bool, error) {
	_, err := os.Lstat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, fmt.Errorf("inspect output-file %s: %w", path, err)
}

func validateResponse(response client.BytesResponse, contentType string) error {
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("SaaS AI returned HTTP %d; expected 200", response.StatusCode)
	}
	if len(response.Data) == 0 {
		return errors.New("SaaS AI returned an empty response")
	}
	mediaType, _, err := mime.ParseMediaType(response.ContentType)
	if err != nil || mediaType != contentType {
		return fmt.Errorf("SaaS AI returned content type %q; expected %s", response.ContentType, contentType)
	}
	return nil
}

func writeAudio(path string, data []byte, overwrite bool) error {
	file, err := os.CreateTemp(filepath.Dir(path), ".grn-audio-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	if overwrite {
		return os.Rename(file.Name(), path)
	}
	return os.Link(file.Name(), path)
}
