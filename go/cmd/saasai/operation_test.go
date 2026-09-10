package saasai

import (
	"bytes"
	"encoding/json"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/client"
	"github.com/spf13/cobra"
)

type recordedCall struct {
	method      string
	path        string
	body        []byte
	contentType string
	streamed    bool
}

type recordingAPI struct {
	calls    []recordedCall
	response client.BytesResponse
}

func (f *recordingAPI) RequestBytes(method, path string, _ map[string]string, body []byte, contentType string) (client.BytesResponse, error) {
	f.calls = append(f.calls, recordedCall{method: method, path: path, body: body, contentType: contentType})
	if f.response.Data == nil {
		f.response = client.BytesResponse{StatusCode: http.StatusOK, Data: []byte(`{"transcript":"hello"}`), ContentType: "application/json"}
	}
	return f.response, nil
}

func (f *recordingAPI) RequestStream(method, path string, _ map[string]string, body io.Reader, contentType string) (client.BytesResponse, error) {
	payload, err := io.ReadAll(body)
	if err != nil {
		return client.BytesResponse{}, err
	}
	f.calls = append(f.calls, recordedCall{method: method, path: path, body: payload, contentType: contentType, streamed: true})
	if f.response.Data == nil {
		f.response = client.BytesResponse{StatusCode: http.StatusOK, Data: []byte(`{"transcript":"hello"}`), ContentType: "application/json"}
	}
	return f.response, nil
}

func TestSpeechToTextSendsPublishedMultipartFields(t *testing.T) {
	dir := t.TempDir()
	audioFile := filepath.Join(dir, "sample.wav")
	if err := os.WriteFile(audioFile, []byte("audio-data"), 0o600); err != nil {
		t.Fatal(err)
	}
	fake := installRecordingClient(t)
	runCommand(t, newSpeechToTextCommand(), map[string]string{"audio-file": audioFile, "encoding-type": "wav"})
	if len(fake.calls) != 1 {
		t.Fatalf("calls = %#v, want one call", fake.calls)
	}
	call := fake.calls[0]
	if call.method != http.MethodPost || call.path != speechToTextPath || !call.streamed || !strings.HasPrefix(call.contentType, "multipart/form-data;") {
		t.Fatalf("call = %#v, want speech-to-text multipart POST", call)
	}
	_, params, err := mime.ParseMediaType(call.contentType)
	if err != nil {
		t.Fatalf("parse content type: %v", err)
	}
	reader := multipart.NewReader(bytes.NewReader(call.body), params["boundary"])
	fields := map[string]string{}
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("read multipart: %v", err)
		}
		value, _ := io.ReadAll(part)
		fields[part.FormName()] = string(value)
	}
	if fields["encoding_type"] != "wav" || fields["audio_file"] != "audio-data" {
		t.Fatalf("multipart fields = %#v, want encoding type and audio bytes", fields)
	}
}

func TestTextToSpeechSendsOptionalPublishedFieldsAndWritesAudio(t *testing.T) {
	dir := t.TempDir()
	outputFile := filepath.Join(dir, "speech.mp3")
	fake := installRecordingClient(t)
	fake.response = client.BytesResponse{StatusCode: http.StatusOK, Data: []byte{0x49, 0x44, 0x33}, ContentType: "audio/mpeg"}
	runCommand(t, newTextToSpeechCommand(), map[string]string{"input": "hello", "speed": "1", "speaker-id": "1", "encode-type": "1", "output-file": outputFile})
	if len(fake.calls) != 1 {
		t.Fatalf("calls = %#v, want one call", fake.calls)
	}
	var body map[string]any
	if err := json.Unmarshal(fake.calls[0].body, &body); err != nil {
		t.Fatalf("decode request: %v", err)
	}
	if fake.calls[0].method != http.MethodPost || fake.calls[0].path != textToSpeechPath || fake.calls[0].contentType != "application/json" || body["input"] != "hello" || body["speed"] != float64(1) || body["speaker_id"] != float64(1) || body["encode_type"] != float64(1) {
		t.Fatalf("call = %#v body = %#v, want published JSON request", fake.calls[0], body)
	}
	got, err := os.ReadFile(outputFile)
	if err != nil || string(got) != string([]byte{0x49, 0x44, 0x33}) {
		t.Fatalf("output = %q, %v; want audio bytes", got, err)
	}
}

func TestDryRunSkipsClientAndFileWrite(t *testing.T) {
	dir := t.TempDir()
	outputFile := filepath.Join(dir, "speech.wav")
	clientCalls := 0
	previous := newClient
	newClient = func(*cobra.Command) (saasAIAPI, error) {
		clientCalls++
		return &recordingAPI{}, nil
	}
	t.Cleanup(func() { newClient = previous })
	runCommand(t, newTextToSpeechCommand(), map[string]string{"input": "hello", "output-file": outputFile, "dry-run": "true"})
	if clientCalls != 0 {
		t.Fatalf("client factory calls = %d, want 0", clientCalls)
	}
	if _, err := os.Stat(outputFile); !os.IsNotExist(err) {
		t.Fatalf("output file exists after dry run: %v", err)
	}
}

func TestSpeechToTextDryRunSkipsClientAndAudioRead(t *testing.T) {
	audioFile := filepath.Join(t.TempDir(), "sample.wav")
	if err := os.WriteFile(audioFile, []byte("audio-data"), 0o000); err != nil {
		t.Fatal(err)
	}
	clientCalls := 0
	previous := newClient
	newClient = func(*cobra.Command) (saasAIAPI, error) {
		clientCalls++
		return &recordingAPI{}, nil
	}
	t.Cleanup(func() { newClient = previous })
	runCommand(t, newSpeechToTextCommand(), map[string]string{"audio-file": audioFile, "encoding-type": "wav", "dry-run": "true"})
	if clientCalls != 0 {
		t.Fatalf("client factory calls = %d, want 0", clientCalls)
	}
}

func TestTextToSpeechExistingOutputNonInteractiveDoesNotCallClient(t *testing.T) {
	dir := t.TempDir()
	outputFile := filepath.Join(dir, "speech.wav")
	const existing = "existing audio"
	if err := os.WriteFile(outputFile, []byte(existing), 0o600); err != nil {
		t.Fatal(err)
	}
	clientCalls := 0
	previous := newClient
	newClient = func(*cobra.Command) (saasAIAPI, error) {
		clientCalls++
		return &recordingAPI{}, nil
	}
	t.Cleanup(func() { newClient = previous })
	cli.SetNonInteractive(true)
	t.Cleanup(func() { cli.SetNonInteractive(false) })

	cmd := newTextToSpeechCommand()
	setFlags(t, cmd, map[string]string{"input": "hello", "output-file": outputFile})
	if err := cmd.RunE(cmd, nil); err == nil {
		t.Fatal("expected non-interactive confirmation refusal")
	}
	if clientCalls != 0 || cli.ConfirmationError() == nil {
		t.Fatalf("client calls = %d, confirmation error = %v; want no client and non-interactive refusal", clientCalls, cli.ConfirmationError())
	}
	assertFileContents(t, outputFile, existing)
}

func TestTextToSpeechExistingOutputInteractiveDeclineDoesNotCallClient(t *testing.T) {
	dir := t.TempDir()
	outputFile := filepath.Join(dir, "speech.wav")
	const existing = "existing audio"
	if err := os.WriteFile(outputFile, []byte(existing), 0o600); err != nil {
		t.Fatal(err)
	}
	clientCalls := 0
	previous := newClient
	newClient = func(*cobra.Command) (saasAIAPI, error) {
		clientCalls++
		return &recordingAPI{}, nil
	}
	t.Cleanup(func() { newClient = previous })
	cli.SetNonInteractive(false)
	setStdin(t, "n\n")

	cmd := newTextToSpeechCommand()
	setFlags(t, cmd, map[string]string{"input": "hello", "output-file": outputFile})
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatalf("RunE() error = %v, want no-op after declining confirmation", err)
	}
	if clientCalls != 0 || cli.ConfirmationError() != nil {
		t.Fatalf("client calls = %d, confirmation error = %v; want no client and no interactive confirmation error", clientCalls, cli.ConfirmationError())
	}
	assertFileContents(t, outputFile, existing)
}

func TestTextToSpeechRejectsOutOfRangeValuesBeforeClientConstruction(t *testing.T) {
	clientCalls := 0
	previous := newClient
	newClient = func(*cobra.Command) (saasAIAPI, error) {
		clientCalls++
		return &recordingAPI{}, nil
	}
	t.Cleanup(func() { newClient = previous })
	cmd := newTextToSpeechCommand()
	setFlags(t, cmd, map[string]string{"input": "hello", "speed": "1.3", "output-file": filepath.Join(t.TempDir(), "speech.wav")})
	if err := cmd.RunE(cmd, nil); err == nil || !strings.Contains(err.Error(), "speed") {
		t.Fatalf("RunE() error = %v, want speed validation error", err)
	}
	if clientCalls != 0 {
		t.Fatalf("client factory calls = %d, want 0", clientCalls)
	}
}

func installRecordingClient(t *testing.T) *recordingAPI {
	t.Helper()
	fake := &recordingAPI{}
	previous := newClient
	newClient = func(*cobra.Command) (saasAIAPI, error) { return fake, nil }
	t.Cleanup(func() { newClient = previous })
	return fake
}

func runCommand(t *testing.T, cmd *cobra.Command, flags map[string]string) {
	t.Helper()
	setFlags(t, cmd, flags)
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatalf("RunE() error = %v", err)
	}
}

func setFlags(t *testing.T, cmd *cobra.Command, flags map[string]string) {
	t.Helper()
	for name, value := range flags {
		if err := cmd.Flags().Set(name, value); err != nil {
			t.Fatalf("set --%s: %v", name, err)
		}
	}
}

func assertFileContents(t *testing.T, path, want string) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if string(got) != want {
		t.Fatalf("file %s = %q, want %q", path, got, want)
	}
}

func setStdin(t *testing.T, input string) {
	t.Helper()
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdin pipe: %v", err)
	}
	if _, err := writer.WriteString(input); err != nil {
		t.Fatalf("write stdin pipe: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close stdin pipe writer: %v", err)
	}
	previous := os.Stdin
	os.Stdin = reader
	t.Cleanup(func() {
		os.Stdin = previous
		_ = reader.Close()
	})
}
