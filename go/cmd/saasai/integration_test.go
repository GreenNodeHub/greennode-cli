package saasai

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/greennodehub/greennode-cli/internal/client"
	"github.com/greennodehub/greennode-cli/internal/testutil"
	"github.com/spf13/cobra"
)

type fixtureToken struct{}

func (fixtureToken) GetToken() (string, error)     { return "fixture-access-token", nil }
func (fixtureToken) RefreshToken() (string, error) { return "fixture-refreshed-token", nil }

func TestMultipartAndBinaryThroughHTTPTransport(t *testing.T) {
	dir := t.TempDir()
	audioFile := filepath.Join(dir, "fixture.wav")
	outputFile := filepath.Join(dir, "fixture.mp3")
	audio := []byte("fixture-audio-bytes")
	result := []byte{0x49, 0x44, 0x33, 0x00, 0xff}
	if err := os.WriteFile(audioFile, audio, 0600); err != nil {
		t.Fatal(err)
	}
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Method != http.MethodPost || r.Header.Get("Authorization") != "Bearer fixture-access-token" || r.Header.Get("portal-user-id") != "" {
			t.Error("unexpected method/authentication")
		}
		switch r.URL.Path {
		case "/api/v1/speechtotext/sync":
			reader, err := r.MultipartReader()
			if err != nil {
				t.Error(err)
				w.WriteHeader(400)
				return
			}
			fields := map[string]string{}
			for {
				part, err := reader.NextPart()
				if err == io.EOF {
					break
				}
				if err != nil {
					t.Error(err)
					return
				}
				raw, err := io.ReadAll(part)
				if err != nil {
					t.Error(err)
				}
				if _, exists := fields[part.FormName()]; exists {
					t.Error("duplicate multipart field")
				}
				fields[part.FormName()] = string(raw)
				if part.FormName() == "audio_file" && part.FileName() != "fixture.wav" {
					t.Error("incorrect audio filename")
				}
			}
			if !reflect.DeepEqual(fields, map[string]string{"encoding_type": "wav", "audio_file": string(audio)}) {
				t.Errorf("multipart fields = %#v", fields)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"transcript":"Fixture transcription."}`)
		case "/api/v1/texttospeech/sync":
			if r.Header.Get("Content-Type") != "application/json" {
				t.Error("incorrect JSON content type")
			}
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			want := map[string]any{"input": "Fixture text.", "speed": float64(1), "speaker_id": float64(1), "encode_type": float64(1)}
			if !reflect.DeepEqual(body, want) {
				t.Errorf("body = %#v", body)
			}
			w.Header().Set("Content-Type", "audio/mpeg")
			_, _ = w.Write(result)
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(404)
		}
	}))
	defer srv.Close()
	previous := newClient
	newClient = func(*cobra.Command) (saasAIAPI, error) {
		return client.NewGreennodeClient(srv.URL, fixtureToken{}, time.Second, time.Second, true, false), nil
	}
	defer func() { newClient = previous }()
	testutil.CaptureStdout(t, func() {
		runCommand(t, newSpeechToTextCommand(), map[string]string{"audio-file": audioFile, "encoding-type": "wav"})
		runCommand(t, newTextToSpeechCommand(), map[string]string{"input": "Fixture text.", "speed": "1", "speaker-id": "1", "encode-type": "1", "output-file": outputFile})
	})
	got, err := os.ReadFile(outputFile)
	if err != nil || !bytes.Equal(got, result) {
		t.Fatalf("audio output = %x, %v", got, err)
	}
	info, err := os.Stat(outputFile)
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatalf("audio permissions: %v, %v", info, err)
	}
	if calls.Load() != 2 {
		t.Fatalf("API calls = %d, want 2", calls.Load())
	}
}

func TestHTTPFailureDoesNotRetryOrWriteAudio(t *testing.T) {
	for _, status := range []int{401, 429, 503} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			output := filepath.Join(t.TempDir(), "fixture.mp3")
			var calls atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(status) }))
			defer srv.Close()
			previous := newClient
			newClient = func(*cobra.Command) (saasAIAPI, error) {
				return client.NewGreennodeClient(srv.URL, fixtureToken{}, time.Second, time.Second, true, false), nil
			}
			defer func() { newClient = previous }()
			cmd := newTextToSpeechCommand()
			setFlags(t, cmd, map[string]string{"input": "Fixture text.", "output-file": output})
			if err := cmd.RunE(cmd, nil); err == nil {
				t.Fatal("expected request failure")
			}
			if calls.Load() != 1 {
				t.Fatalf("calls = %d, want 1", calls.Load())
			}
			if _, err := os.Stat(output); !os.IsNotExist(err) {
				t.Fatalf("output created after failure: %v", err)
			}
		})
	}
}

func TestAudioCommitDoesNotClobberNewFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "fixture.mp3")
	if err := os.WriteFile(path, []byte("existing"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := writeAudio(path, []byte("replacement"), false); err == nil {
		t.Fatal("expected no-clobber error")
	}
	assertFileContents(t, path, "existing")
	if err := writeAudio(path, []byte("replacement"), true); err != nil {
		t.Fatal(err)
	}
	assertFileContents(t, path, "replacement")
}

func TestAudioOutputRejectsSymlinksAndNonFiniteSpeed(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "fixture-target")
	link := filepath.Join(dir, "fixture-link")
	if err := os.WriteFile(target, []byte("existing"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Skip(err)
	}
	cmd := newTextToSpeechCommand()
	setFlags(t, cmd, map[string]string{"input": "Fixture text.", "output-file": link, "force": "true"})
	if err := cmd.RunE(cmd, nil); err == nil {
		t.Fatal("expected symlink refusal")
	}
	assertFileContents(t, target, "existing")
	for _, speed := range []string{"NaN", "+Inf", "-Inf"} {
		cmd := newTextToSpeechCommand()
		setFlags(t, cmd, map[string]string{"input": "Fixture text.", "output-file": filepath.Join(dir, "fixture.mp3"), "speed": speed, "dry-run": "true"})
		if err := cmd.RunE(cmd, nil); err == nil {
			t.Fatalf("accepted speed %s", speed)
		}
	}
}

func TestInvalidResponsesDoNotWriteAudio(t *testing.T) {
	for _, response := range []client.BytesResponse{
		{StatusCode: 201, ContentType: "audio/mpeg", Data: []byte("fixture-audio")},
		{StatusCode: 200, ContentType: "application/json", Data: []byte("{}")},
		{StatusCode: 200, ContentType: "audio/mpeg", Data: []byte{}},
	} {
		fake := installRecordingClient(t)
		fake.response = response
		output := filepath.Join(t.TempDir(), "fixture.mp3")
		cmd := newTextToSpeechCommand()
		setFlags(t, cmd, map[string]string{"input": "Fixture text.", "output-file": output})
		if err := cmd.RunE(cmd, nil); err == nil {
			t.Fatal("expected response validation failure")
		}
		if _, err := os.Stat(output); !os.IsNotExist(err) {
			t.Fatalf("output created: %v", err)
		}
	}
}

func TestGlobalFactoryUsesBothProfileModes(t *testing.T) {
	for _, mode := range []string{"machine", "user"} {
		t.Run(mode, func(t *testing.T) {
			var calls atomic.Int32
			server := testutil.ProfileServer(t, mode, "", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.Method != http.MethodPost || r.URL.Path != "/api/v1/texttospeech/sync" {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL)
				}
				w.Header().Set("Content-Type", "audio/mpeg")
				_, _ = io.WriteString(w, "fixture-audio")
			})
			output := filepath.Join(t.TempDir(), "fixture.mp3")
			root := testutil.ProfileRoot(newSaaSAICommand())
			root.SetArgs([]string{"saas-ai", "text-to-speech", "synthesize", "--input", "Fixture text.", "--output-file", output, "--endpoint-url", server.URL, "--allow-untrusted-endpoint"})
			testutil.CaptureStdout(t, func() {
				if err := root.Execute(); err != nil {
					t.Fatal(err)
				}
			})
			assertFileContents(t, output, "fixture-audio")
			if calls.Load() != 1 {
				t.Fatalf("API calls = %d, want 1", calls.Load())
			}
		})
	}
}
