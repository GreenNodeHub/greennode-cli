package login

import (
	"context"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"strings"
	"sync"
)

// ErrAuthzDenied is returned by Serve when the callback carries RFC 6749
// §4.1.2.1 error params (?error=…): the user denied consent or IAM rejected the
// authorize request.
type ErrAuthzDenied struct {
	Code        string
	Description string
}

func (e *ErrAuthzDenied) Error() string {
	return fmt.Sprintf("login: authorization denied (code=%s): %s", e.Code, e.Description)
}

const callbackPath = "/callback"

// Listener is a one-shot loopback HTTP server that captures the IAM callback.
// It binds 127.0.0.1:<random port>/callback, serves exactly one request, then
// shuts down (RFC 8252 loopback redirect URI).
type Listener struct {
	uri       string
	srv       *http.Server
	bound     net.Listener
	mu        sync.Mutex
	code      string
	state     string
	cbErr     error
	wantState string        // when non-empty, handle rejects callbacks whose appState differs
	received  chan struct{} // closed on first callback
}

// setExpectedState arms the state-mismatch check. When set to a non-empty nonce,
// handle serves a failure page and Serve returns ErrStateMismatch if the
// callback's appState differs from it. The zero value (empty) disables the
// check, leaving the listener nonce-agnostic (how the unit tests exercise it).
func (l *Listener) setExpectedState(s string) { l.wantState = s }

// NewListener binds 127.0.0.1:0 (random ephemeral port) and returns a Listener
// whose RedirectURI() is http://127.0.0.1:<port>/callback. Serve starts
// accepting.
func NewListener() (*Listener, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("login: bind loopback: %w", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	l := &Listener{
		uri:      fmt.Sprintf("http://127.0.0.1:%d/callback", port),
		bound:    ln,
		received: make(chan struct{}),
	}
	mux := http.NewServeMux()
	mux.HandleFunc(callbackPath, l.handle)
	l.srv = &http.Server{Handler: mux}
	return l, nil
}

// RedirectURI returns the URI to put on the IAM authorize URL.
func (l *Listener) RedirectURI() string { return l.uri }

// Close releases the bound listener. Safe to call multiple times (idempotent).
func (l *Listener) Close() error {
	if l.bound != nil {
		err := l.bound.Close()
		l.bound = nil
		return err
	}
	return nil
}

// Serve runs the server until the first callback arrives or ctx is cancelled.
// Returns the parsed code and appState; RFC 6749 error params → *ErrAuthzDenied.
func (l *Listener) Serve(ctx context.Context) (string, string, error) {
	serveErr := make(chan error, 1)
	go func() {
		serveErr <- l.srv.Serve(l.bound) // ErrServerClosed on Shutdown
	}()

	ctxCancel := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			_ = l.srv.Shutdown(context.Background())
			close(ctxCancel)
		case <-l.received:
			// callback arrived; Serve's <-received branch drains srv.Serve.
		}
	}()

	select {
	case <-l.received:
		// handle() captured code/state/cbErr and triggered Shutdown.
		<-serveErr // wait for the server goroutine to finish
		return l.code, l.state, l.cbErr
	case <-ctxCancel:
		<-serveErr
		return "", "", ctx.Err()
	case <-serveErr:
		// Server exited on its own (e.g. accept error) without a callback.
		// On ctx cancel, Shutdown (from the watcher) closes the listener and
		// srv.Serve returns here before ctxCancel is observed — surface the
		// honest ctx error so errors.Is(., context.Canceled) holds.
		if err := ctx.Err(); err != nil {
			return "", "", err
		}
		return "", "", fmt.Errorf("login: listener exited without callback: %w", l.cbErr)
	}
}

// handle is the /callback handler. Extracts code+appState (or error params),
// serves a static result page, and signals Serve to return. One-shot: after the
// first request it shuts the server down.
func (l *Listener) handle(w http.ResponseWriter, r *http.Request) {
	l.mu.Lock()
	defer l.mu.Unlock()
	// Shutdown in a goroutine: calling (*http.Server).Shutdown synchronously
	// from within the handler self-deadlocks (Shutdown waits for this in-flight
	// connection to drain, which can't happen until handle returns). Running it
	// async lets handle return, the response flush, and the connection go idle so
	// the Shutdown goroutine then completes. Triggered by handle (one-shot).
	defer func() { go l.srv.Shutdown(context.Background()) }()
	q := r.URL.Query()
	if e := q.Get("error"); e != "" {
		l.cbErr = &ErrAuthzDenied{Code: e, Description: q.Get("error_description")}
		writeResult(w, r, false, resultDenied)
		l.signal()
		return
	}
	l.code = q.Get("code")
	l.state = q.Get("appState")
	// When an expected nonce is armed (by Login), a mismatch is a stale or forged
	// callback: serve a failure page and report ErrStateMismatch. Empty wantState
	// disables the check (unit tests exercise the listener without one).
	if l.wantState != "" && l.state != l.wantState {
		l.cbErr = ErrStateMismatch
		writeResult(w, r, false, resultMismatch)
		l.signal()
		return
	}
	writeResult(w, r, true, resultSuccess)
	l.signal()
}

// signal closes l.received once (guard against duplicate close if a second
// request slips in before Shutdown takes effect).
func (l *Listener) signal() {
	select {
	case <-l.received:
	default:
		close(l.received)
	}
}

// resultKind identifies which callback outcome the page reports.
type resultKind int

const (
	resultSuccess resultKind = iota
	resultDenied
	resultMismatch
)

// titleDesc is the title for one language's result page.
type titleDesc struct{ Title string }

// resultCopy holds the bilingual (en/vi) title per outcome. The English titles
// mirror the GreenNode console's MCP login-callback page
// (aiplatform.console.greennode.ai/mcp/login-callback, i18n key
// mcpConnector.publicCallback.*) so the local `grn login` UX matches the
// console verbatim. Under the title, closeCaptions adds a one-line "you can
// close this tab" instruction (the tab is OS-launched, so a script
// window.close() button cannot close it — see resultPageTmpl). Vietnamese is
// taken from the console's vi.json.
var resultCopy = map[resultKind]struct{ En, Vi titleDesc }{
	resultSuccess: {
		En: titleDesc{Title: "Authorization successful"},
		Vi: titleDesc{Title: "Xác thực thành công"},
	},
	resultDenied: {
		En: titleDesc{Title: "Authorization failed"},
		Vi: titleDesc{Title: "Xác thực thất bại"},
	},
	resultMismatch: {
		En: titleDesc{Title: "Authorization failed"},
		Vi: titleDesc{Title: "Xác thực thất bại"},
	},
}

// closeCaptions is the bilingual one-line instruction shown under the title on
// every result page: the IAM authorize URL is opened by the OS browser launcher
// (not by window.open), so the callback tab is OS-launched and browsers only
// allow scripts to close script-opened windows — a "Close tab" button would
// silently do nothing. The page instead tells the user to close the tab
// manually. Identical across outcomes (success/denied/mismatch); the grn
// process in the terminal reports the result either way.
var closeCaptions = struct{ En, Vi string }{
	En: "You can close this tab.",
	Vi: "Bạn có thể đóng tab này.",
}

// brandMarkImg is the GreenNode brand mark, emitted as a trusted template.HTML
// so the data:image/png URI is not rewritten by html/template's safe-URL filter
// (which only allows http(s)/mailto in a `src` attribute and would otherwise
// replace it with the #ZgotmplZ sentinel). It is a package CONSTANT — the PNG is
// the official GreenNode brand mark (provided at /Users/lap15626/Downloads/
// greennode.png, 447×447 palette PNG; downscaled to 256×256 for a crisp inline
// data-URI at the rendered 72px height) — and NO query param or user input
// reaches it, so marking it template.HTML is not an XSS sink. It is the page's
// sole brand element (no wordmark text alongside). The page stays fully
// self-contained (renders offline, no CDN/asset fetch).
const brandMarkImg = template.HTML(`<img class="brand-mark" src="data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAQAAAAEACAIAAADTED8xAAAAAXNSR0IArs4c6QAAAERlWElmTU0AKgAAAAgAAYdpAAQAAAABAAAAGgAAAAAAA6ABAAMAAAABAAEAAKACAAQAAAABAAABAKADAAQAAAABAAABAAAAAABn6hpJAAAmxUlEQVR4Ae2dCXxVxb3H75bkZl+AQAgQQtgXAUHccFcQqbVVpCpUcXlWX2utS9Wq1bb61GprqfbVpep7FbVat4dU3KioqCjFgMi+r4mQjYSsN3d53+TUk5Nzl9wll1yZ//3wCXNm+c/8l98s/zMzx+rz+SzyEwmoKgGbqowL3yKBNgkIAMQOlJaAAEBp9QvzAgCxAaUlIABQWv3CvABAbEBpCQgAlFa/MC8AEBtQWgICAKXVL8wLAMQGlJaAAEBp9QvzAgCxAaUlIABQWv3CvABAbEBpCQgAlFa/MC8AEBtQWgICAKXVL8wLAMQGlJaAAEBp9QvzAgCxAaUlIABQWv3CvABAbEBpCQgAlFa/MC8AEBtQWgICAKXVL8wLAMQGlJaAAEBp9QvzAgCxAaUlIABQWv3CvABAbEBpCQgAlFa/MC8AEBtQWgICAKXVL8wLAMQGlJaAAEBp9QvzAgCxAaUlIABQWv3CvABAbEBpCQgAlFa/MC8AEBtQWgICAKXVL8wLAMQGlJaAAEBp9QvzAgCxAaUlIABQWv3CvABAbEBpCQgAlFa/MC8AEBtQWgICAKXVL8wLAMQGlJaAAEBp9QvzAgCxAaUlIABQWv3CvABAbEBpCQgAlFa/MC8AEBtQWgICAKXVL8wLAMQGlJaAAEBp9QvzAgCxAaUlIABQWv3CvABAbEBpCQgAlFa/MC8AEBtQWgICAKXVL8wLAMQGlJaAAEBp9QvzAgCxAaUlIABQWv3CvABAbEBpCQgAlFa/MC8AEBtQWgICAKXVL8wLAMQGlJaAAEBp9QvzAgCxAaUlIABQWv3CvABAbEBpCQgAlFa/MC8AEBtQWgICAKXVL8wLAMQGlJaAAEBp9QvzAgCxAaUlIABQWv3CvCNhRdDY2Hiw/dfS0hJOI30+n91uHzVqVHJysn/+/fv3l5WV+ceHiLHZbOnp6RkZGXl5eQFphiirJ9XV1cEEf1tbW/XIcAKwM2DAgPz8/BCZIVtVVXXo0CGPxxMim3+S1+vt379/QUGBlkQLt23bBr/+ObuMSUpKys3NzczMzMrK6jJzAmZIRAB8+eWXS5YsWb169a5du4CB2+0OR3AoNSUlZdGiRf369fPP/9577917771Op9M/KViM1WrF7gFAcXHxscceO23atKKiomCZTfE0ZsWKFUuXLv38888rKiqam5sjtVGK3HrrrZdffrmJMo8QX7lyJRyVlpaCajoIYvyzhYhxuVw333zzFVdcoeX54osvrr76aiwY1IUoFTCJTictLa2wsHDChAlnn302fwNmS9jIxAIAtvLHP/7xn//8Z21tLZJ1OBx0SwTCFB/6C6ZC7C8iUlqNdNvV1dWMHsuXL3/uuecuueSSefPmAbPQ7dm3bx9cfPDBB/TNdJBwAZbC50IjTqmAvOzZs+cPf/gDxJuamqImrsnWyAUx/ALWaMwWMNzQ0LB+/fo1a9a89tprZ5xxxk9+8hPwEDBnAkYmEAC2bt1622230f0zntKpRCosjIxfsFJaaogMAQtq+bEMUsEkloem77zzzj59+gTMT+TatWvpueGFoYMZVLBsXcZrDTZl60biJsoBqzPlCfZIWYZKfozVr776KkP3Qw89NHbs2GD5Eyo+mmlfPBigY2NQ3rBhQ05OTnST0Xi0ykiTLhmbfuedd+666y4mZsYkPcyc7Y477tixYwfTCQ08elLsAWY7v/zlL+NEPPbmQQHFZWdn7969+5ZbbqGd3UIz3kQSAgBMSX/zm99s2rQpNTXVNArzGNEvInl1SdmfGhhgZv/kk09S1pQKF/R8W7ZsoeM3pXZZUcAMRvpMxhh/EBFjY7cTN1akhwM2KWCkXoQAGVDizp076QhYxhiTEjOcEFOgxYsXs2Q0TRiYtdPRatNck8oDipKFYERDBzS1hWkw4nTh2DSp2JyxO0fBf//7388880zTKM86YdmyZSYMMytgss70INIZNm0zLm1BHYMPxI28QxwRsSaJiHEoQBzWjKT8w6GFo+VHLDSSVTgNY4TUiSCxVatWvfHGG7Nnz9YjEzPQ0eieah/2weLJqGxagmoZTBHfcccdF+Z6AEvFyHBZhsMI2mWW8otf/AIXXoj8OBk//PBDjI8OWFsJkBlNszJ+5ZVXxowZowODBj///PNkA7E6QYwMl9TMmTOPPvpoMKDHhxNAIAMHDtRyQvzFF1/kr5EIdeEkPe+882gG9qe3JEzi+FiD5dR6hLvvvhtXabA8WjyVgkA8XXjtDhw4YOSdpqLW6dOno8fQRHo2tecBwJxh48aNRtmh6d69ez/44IPYTZykg45R3qRJk0IsZ7WqzznnHBZ2999/v1aESAIYHMCoqanR8Yb6cYMY3ayageIO4tVEjFzgHNu8ebPR+4SIaPnDDz88bty4GIkHLE4Hwfg2bNiwgKmmyFNPPRUpaesTXY90E7QZn8Hxxx9vyp9Qjz2/BsBu6uvrjYM4Hee1114bP+vXFAAAMKNwlHHBBRfgADXOaGktg4BxnYeyTVxgQ1deeWXs1k8LWVvjatSHIGJAF8TjZP2aTLqcIxlFR0vwfiIWbfQgCfEyNVq3bp0xWwKGex4A27dvx1aQlyYdwozOZ511VkIJ69xzz2WJos/TNPDQ6+uNxOWvhwnABYNDd2GYoQaL1+lDnBEy0V45TZ06lSmTsZ00mNWw3uzEDPQ8AOhp9G4DGWna1UfSBJEaCwb8P6Z2Go0eSPAzttY/xpgaUdhYr1aQuVaYS6OIKoolM01CSnofoZEKc4yNpd4Yy/Y8AEx2Az8I0V/lMfIZY3F23TDDMTaVCUnPmmCiiQgJG+WjCdw/JkZFdHvxngeAiSX0moCqffPNN5mF6wsVWsgir2/fvqbGGx/RvXFNbEyKPQzxRBskY2eqRyj0vBcoIra/+uor3hmj+y5BwjDCyiz0phSj6zpYM6gI63/22WeNThhtij9kyJBgpTBQGvDpp5+yODbNCoIVIZ5SzBmY3Ov7NANmJhvOx48++ghHUPjEIUWzJ0+e3KXjK2ClR2rktwwAuMOxRTYLhQYAqVjS/PnzgwEAGyIPuyBZTQYjRR5mPmw74z0d6te7f+J5d/H973+/V69ewcyCzIwY99xzT0QGCmUmWk899VRoAGjE2dwaEXHYxC3Dlj4BgFFrCQcAjICfsYnGMK4YXqyYXogaM2hhlM0vxCSBGTwGiuvav6wxhgU6vSbvdHTrJ5UYNizxBsqY0z8MF1EsEmiYsS5/slpMFMQRCHIz+lKDEVcqPuEAgJ5CKKDNrtt/IfKQ1F15gJA/inghMGfOnPHjx4dug9aMLvOYMtByU0ywx/BzahTCkUmwuo7g+IQDALKOVLVRqyfEUBOMJn5uHN7XXHNNFGWD0ZT4HpRAIgIghDi0biwgQg6bRYKBgA3wb3aY2YwFw5/WR0qc/BCPtJSxbUdk+FsGAObHuG5ME1nN9JmaR6ohrCG0QUBZI65TZj3Ark+2Q99www2mJD2PHjC1U48PEYDB0E3Sy0ZKvJ3XNgzoFCSABBIOAFhVsFUgKuSM7Pnnn2+yPPJzWITzNGjXlBRax2ApdH46e2b8xkUwbWAp+fLLL3P2L8QyADTiquKQA26oiGwO+l3uwdRW4RxMGzRoUETEkUaITaChZXWkpiYcALCAYErFWPEPBnQR4qQPViqg5rAhHErs+OVlVrCCVMcmnPfff5+N+LiD9B4X2LAT7vXXXw8BACqlOPuZS0pKAjYgxkj4HTFihFhzjGKkeMIBIDqWghlxMGrADPcOr5y6dIqz15czCRyDpAptaNIGAbZDc1ogxKsAqo5iVhaswaZ42pD422xMbU7Mx57fCoEujaKh48TO+GuM7DJs2oRIfsh2aX9h2hCb3efOnWvcDs1owF06xq2OAeuKlIsu2dQzQDl+xPVaVAj0PADYZWkUNBOM8vJyXvUbI7sMc3OJyQRBkYlyl0RCZJgxYwZTJh2rGB+Tor179+pFqF1PJZIMjBimJumZIw0YKWvEeYvHL1I6kt9fAj0PgKFDh2Iuuo4xXKyZA4f+bQ0WQ/e/cOFCrE3vFDE+3sJ24xSZV7+mvb5Ux84CvUkcfdRrJxIuWCewjULPEEuA/RrG93EQZ/bFRqNYaEpZTQI9DwCO3nEwF5PVVcImygULFrz11lvh9KB0hOz54Zo0fDU6BQriIQl9r6CeOcyA0b61IsaYkSNHcgLGyAXt4ZQwS4Uw6YfIxq47RGScsLEIfvrpp7sLYCGqPuKTen4RXFxczGL0448/xr2ojQPMsNkThovm3Xff5UQpnZ8+Phj1QUeIl4aLGNjTZnJoAoCTTz65G6dAxnoDhjFQ1spczKjv/4EL9tLdfvvt2v0RtDAgFwGpEQmW6BrAFWFGAI5W4o+CiJYf4mzIgzjeWDa9BhNRMOLIB5mHeeQ3GJEjI77nAUBnxu0Pn332GSrX+1Q0jZLo1/mFFjQwMPb9ZGZGhKu0y81qoclGmkrLL7zwQi5HgAuapBXHLjFTztQzo9NZC4cyUKG/582GBgDKXnTRRUDdSBwRsS7HGwt9vcZwiJOHyRvgEQAgip4HAI046aST6Ca5RoZ5tq5CtB7FgRJMB+3+6Ec/CnhFrk48HgH6VPxFXBkEF3pnj2lGwQXNgwu9v+eRkZCz+S+88ALv1/TGR02cgowhOh2VAz2/BkD69JRc0cMms0idPybN0UHSKXLpMZv1TUmH4RF7ZX8E90hzVjii/j6ctkH8Zz/7GTAwncwMp6zkCSGBhAAA7eON0n333QcGsB7jai9E041J9LiYPvMN9kpcf/31xr7TmC3eYdbB3CDEYiA6LkI3j4MQiAjiYID5YejMkhqmBBJiCqS1lUnLI488wnkoJrWVlZV059hxl7NbTEHLyXT5qquu4or6gJyDEHLy01IJUCpgzmCR5NeLk4ewPs8xFmHzD5dhwQX3ovGyjKEALqIYEALSZ5vQo48++thjj+H25bZq6o2OOF2MsfG6cLRI7dGYwchgiLAmIl1KBCIVcgjicUpKIADAIdNl7ldisssnArhckkvGNTWHYB6bGD16NEddcfvoHhj//CyUmZrrL7OwABbfEdklxOmD9akz0zYo+FdEDNluuumm733ve/ht+IYFh5jpswPmDBHJi7aAiwfWANy9PGvWLFxkXDvF5VxRzLgYKo2NhxfI0mzd6JFPFKMoDYaI3myq6PLsXggJHJ6kjjdQh6e+8Guh/0C1rAV1rfiXxYKxS+O60D+PFsPrAqgZUxlbmLGEqWYMghdbxv6MVvF2rEsF036qxpqNVYcZhi8QGyIzraKD8N8GEqKInoSPWHcTM3tksNKTtAC+1zCFoxfkCkeaZOxWwEDo21f1sj0VSFwA9JREpF6lJJAoi2ClhC7MJo4EBACJowtpSQ9IILEWwT0ggIiq9FncPq/Hyl5rr8Nncdgclsh2bUdUmWQ+HBIQAHQhZQ4r7Gr4el3NjtK67eX1+yub61qdXODmtbkt2Y60ftn54zOHjMksGpozIMUqwuxCmAmY3POL4MV7P1tRvc5KbxrLz2rJSckallk4MbukwBn0wraIaihrqlpyoPTN/cs/rVpf2VLX7G212ujx2brNfn8Lf3xtI4Ev2eLISUodkzlkZv/jZhYcOyJzUESjwjvlK5ZXrrVao9mYQAOSHUl9UnJHZgwckTEgPzXU126MvO9prlyw/W0XHFniOAf2WbwT84Z/t/8JCM1Ye0KFex4AV3xy///s+4clqdPXr6KRkceblpQ2JK3vcTmj55XMODFvdDRE2svsa658euubL5Z/sLW+3GPxpNiS2o+oBdZiGwx8PpfP7fH6Bjh7fSf/2GuGnXdU9pAwa79uxfw/7X7NYu/Yyx1mwX9n82Fcttyk1H4pvU/rfdR5hVNPz59g7wpOn1dvOOWjG1u8LRYwHb+fp3nO4HOem3x7/GqInXJs/W7s9bMRyEEf6sywR/ANd/9qsU2f3eK1+LY07Ft7aNeL5R9dNfCsm0ZdNMDZ2z9ziBj69ed2vfPg1lfW1u5MYp+p3WHtar9g25jAvj1rMp1pRWvt47sXL/z6sx8P++51JednOrpGdZLNYbM706IGQDszTd7WrU37Nu7a/cyeJdN7T7xj9NzJucNDsAmeU+3J/LXHEwD1Fl/bMimxf/HsAA4j5x66Yc5hWazYU4bD6fW1zt/x+sXL/2vdwR3ht6Kqpe7WNU9cuXr+xkO7Mxx8ezEp0rHbYbWnO1KqPbV3rH/m0hUPrK/bFX7tseTEjpOtjnQ777a9Cw98ev6ndz+19R9eX2R7PWJpwLe37BECAEYA4wTFZrWlO1KX1667dOVvNx/aHY56NtbvvXjFPb/f+iqenVRbx5cewylryuNos0Xnwq8/OeeTOz6q/MqUGr9HrQug6v3umh+v/fO96xa4PGF9BC1+TUp8ykcMAIz23yZ2ntNsyaWHtl1b+qfa1i624myu33f5ygeWVJSmOpLpTbEk/x+RrT5Ps6+10dvS6Glp8vFpJw8LAP+cxDB0pDuce5v3X/jZfy05sCpgnvhFMhrYrZZ7tzx//8bnZRwILecEnaJp1tbmaAnvh8FhuHZLm4dG/xFmVvB+VekTWxbdMvpiPd4U2N9cc23p/M+qN2YEmrJj4i4vC1wvw0JRSv5AR6/e6TkOu6OyrnpPa2WZ52C9u5FVQDJT+U6DUFslqfaUanfNVV/8/rkpt0/tFcGinEoBm6mdAR9h3ME9Mp0ZJyfSsNqS7tv8wuCMgsuKppnKttN3s3C3x9MLZMExkPDTsEQEAGvZFEtScVLf9uEJN4e5dzepkw7XZffsba6kY2ZtZ0ylpNOe9Oedi34w6LSijH7GJC3c7HHd9OXjSytXsXLwT8X0qX5C9tCzciae3Xfy6F5DshxpLFoxO7fH3ehr2XaobPmBda+ULVtZv7nJ46IuExGnLWlXU/n1ax5ddPw9/cNbkWOdqdbkIY5c464yE1nt0We1NFndZc2VDZ6WZHsSSDBmY43Lx5Dv3fD8KX0mDE7LNyal2VOPShlc72nqsgpjqUjDHk/rwKTucUlHWnX4+RMRAK1e9+isopePuTPV7sQawmDG2uJrXX9wx+M7/vHmgRWpDqywAzP4BPe2VC3c9+lPR5zvT+qvO99+pfyjNId5YzO1AqdhqQU/H/aDCwaclJvScVZTI8JOySyLY2LeMP7NLZm+dH/pg5tfXH5wg9OeYu+MWAaW1Qe33bP22T9N+hkds38bjDG0u9HTWpwx6PXj7siwp4Vmn1RMf13tjlfKPv6/8k9dXleKjTfUHT8etzWV/e/2xXePndchEYtleMbA10++J8Q22w4SMYRYgzOljIHA4SiaiACA7ySLLT+td0TOwaL0vif0GXfTmsf+uue9FDsOnH//CGAo71eUXlVyTlrnbn7Nwe0PbnmJVHr0b7K3/U9Mi9t9bsFxD429enjmQGNSwHBWUup5A06c2nfcb9e98Kedi3h1YPLEY4gvlH8ws/z47/Tv+rPp1J5ktRem9QFLAaszRQ7NLPxO4QnvVnxx4+o/b2ssw3llzIBbbEHZ+3OLpw1N76/HJ9vsfVPz9EeVA110SD0pmsinj9nJ6b8df80xWcNbvJ28H5wr29Swt8bVaSnM6vDxrW9sb9yPdZrYpO+/avDZCybfFo7162V7JWU9OP6a+0fOs/lsXt5JGH50/A3upns2PnewcxsMWczBiGbP0J+Rf8yCSbcOSe7r8nZaPCRb7dvry1/dudRcgTy3SyBxAdDJgsLWVq+kjIv6n2Ia3OlQ97tq9zRWGMl8dWjnq+Ufs2YwzhnIUO9u/n6/E3877uqspFCHUYykOsJWy3UjLvj50NmNHJjsPHljQVxat21R2fKOzN0dmtxr5B2jfwizpqrtNuuS/as4qtLdFR4J9BIXAJ0mJZGIenRuMUtPkxG4PK6D7k4nwv62Y8kB90HTpBzHyJScEfMn/DQa629vJM6XW0ZdNKfg5BZvq7nVPt+ze/7Z7I3mdJiZVJDnWUWn0f5m0wBote1qOVDeXBOkkNLRiQuAqNXitDuZgpv6dcYEo5uyqrn2vf2lvLEy1kIR3Ki3Db9oYGoo3wWnHDlDaCxoCtPZ3znmh31T8oCTMSnJbl9Vs5mNpcbI7g2nWh1n9z/G29mFCiarXId4O9a9dR0Z1DpZQEKxZLLg8Ntm6vsDFsRzsrWJ9WLHHkwGnEZ38xn9Jp/b/4SARZhCcH8jt3ft2rWLMBePTps2bfr06cbLvPSCIzMHzRs47b5NLyQndUgYp3u1u+6DA6sm9Rqh5+z2QEnGICvTxw7OWOBbW6zu2pZOA2C31/stJdihnkRjIK5j06raLQ3eFqe9g33wZrPY5hachpvfXxTc8vnAAw9g/RzV1y6G2LJlC99q53qSX/3qV1xw7V9kVuEpz+x+p6a1jg1CWioYozNeVr3hBkun4ci/bCwx6T6HvX0GqLu2qNfj9dQ1CAACyLXDAgIk9lwU5sj+tjjVz6cSv6zdgbPSirv1mx/7Gkoy+p+aP+GbiI7/ua+OmzS595M7GkxvjriX97rrrnviiSe4jLqjQHtoZObASZklb1WtbF+VtkXBD++qGHkOug7lJZtfLJiKR/3IqsZ/+YTf0/huZH/LwYX7P3V3uKrathLqpTS5Gx+NYRpmfNTDGoOdHn2+0elFp/Xu+oPKUTMbe8EEBQByZF9D7OwFpFDvad5ctxe7NKa6fZ6j0ocUpAXYPs0FtNzRG/DyFSDBzTxcg8U4YLrDy+lIPj5v1OLKlUYcMwJUNFbta6iMHwBgxMiXFubdHC+G9fid9WU//dcfeXvILdTtwGR9ZLO0FSQPI0d7zn8/EkHSNz457UWeOekb+HTkbI/xuOYUzxQA6GKPIID84jcCNLY21bY2YBLGBtEHDsvsb0IFGVjvvv3228ZrpIylCHM10NKlS7mQdPDgwaakkrRC04jBY4PH1eXmPBOdbn8Eq6nJTpvXbnKCdW9F9Z62b9p2L81up9bJCLqdetQE6YKMTpuo6QQseKi1sdbT0N7LdaSzb6aXM8C0hEsa+WKNqXfvKEY7bTZup9q9e7cxUgv3yc5z2pKZcelJVIobtKY1sg9A6cUl0O0SSFAAwKfJQLuRc9xE2ptadt1pZImhOqdhSaBXxwjA1WumjlxP1QOma+e0eCY8OJoMU4+26YXH4q1zd3xbSacggR6RQIICIK5ToGRbUlLnNwDYJRiodXfaK6Hpg6sXuTrT9GrZpCr8QgE/x9TQ3NB+8LwDy9SCU6hXSpqJgjz2lAQSdIqGycRvEZyRlJplTytrrTHOsuinKzwBHIVY9vDhw/kSfbDpLO8EuBHafwGARvdVH2j2tDoNO7QBNqeHs2xdnxWOt0Ew+jEWxbkWL4CPcxWxkk9QAMBW/JST6nDi7VnXuNti/EqK1bKtrow9/aYTBdg9VzHj8udm3IArAQBw7rnnBvzg9ubmfSYu2IHX25mV39M7MRnvMnwpSRwXML4wi9WWTOU51GFL9XY4mk3JCfKYuADomDd0t6jS7ClD0we8U7GSKvQOinnRv2o2bavbOzZ3iKlCPhZ/5ZVX/uUvf2Gqw3RIT+WlGPeM85m6Sy+9VI/UA3WuhlX121g8GGthQZybkpufmqNnOywBa11ri3GX6Kison9MvQ/m4ydk+GKQyU7ORMJxrSVGASYuAGJkLERx9DEhe7Dd5kBD+utSZlwHXLVLKkv9AQApvlrAN2wef/xx/RpxVgVchH/JJZfwQZqAl5ivOrjty7rtXBenY4x6uVlxnHNQpiOOawCuCfJbsfg4JJSW3HG6gBsDOMcTQkTqJCUuAHS7iYcyjs4emmdPa/A26+4drJP9CU/vfvuHg6exs99UKX3/3LlzTz/99M8//3zz5s34hYqKivhaUbAPLbq9nse2LKxureeWFJ0UHPEm+MyCiXHtEWu8jR6L23idEfVyRCEtJfLd3XrTj9xA4gIgrv6pkdmDudNz6cE1nL7VlIuVpNqSN9TteW7XkuuHBjg8STa+RhPm5/c+rPzyzaoVpuUE59z7peRN6dX26d/4/dYe3MrnZo30WYmmWBycFjJGSliTQFzNLHohY47xexNMs9KSnOcVnmj84guRWA23wT286e/LK9ZF33ScP40Vv16/gBdext0HEGQWfmqvcSUZhbEQD132QFPN4q9XmN5nc7isMDVvYEqAXR6hqamQmqAAwBbj5wbV9HpewYklaf2Nt4+AOvZH7HVVX736kR31ZdGpv7618cel/72seq3pdi2Wv8kW+7yi6VygEh3lcEr9z+63uBnSdMiT+djgzMLezuxwKKiWJ0EBEO8RADUPzMi/csB01zffjSRGQx0n8dfX77iy9PfrandGag0Vrrp5/3p44YGPuVkRFoy/Jk/L2fnHTs0/yhjZveG/7n73d1teZRAzkWWh/52+U/TVjilV8UezsBJEHIdhBIDTK4acc2zW8Ca/M4pg4IOqr2at+PUb+z42HW8PIZ/Sms0XL//Na19/yJ1wpmyMM31Tcn85Zo7pDbQpm/6IyXKbg/7YZeDrxqp71z9/3apH6z2Npv1tnMwcnTpoRhi3UXRZyxGZIQIpH2b+Ta+Q4lF7flruXWMu/cGKe9hCbLKbdHvytoayS7948JKvT/vhoLOm9B5luunE2J7Ndbv/d+e7T+56u9ZTZ3T7aHlYg3Ie5aYRsyflhOV5pKJDnsb3vl7ptKbwAQJjRaYwnfohV+MX1VsWV64oPbQ91c4XiTvt8SY/L2N/MuS7hc5Ol6BwOUVp1SbjLj0T5W55xMs8IC1/dE6RiVp5efm2bdvYXs73bY3vVUzZDs9j4gLg8PA/o2DKnSMuuWvDs5wiNu6MoHYu2OFQ75N73nqlfNn4nJIZfY45uveo/km5OcmZHG6sc9cfaK5Zc3DbsqqvllV/tbO5gldp7P30b3aD2zV3wGk/Lv6uf5J/DBMnp81R1rz/4n/db1qj+2cmhu0MLT43975kBLqCivtdTs2dcHHxmaaym+v3APtGj8u0J9yULcZHt7dlXvGMx4++UafDq8Nnnnlmw4YNxcXFbKHlINGNN94Y8DydXiTegcQFwGGbnP102AX7Gisf2bko3WG+D51hAT8pB2iWVnz5fsXqJGtyQUputj3V5rU2WFv4hEyTr8VrsbLlM9j3DSh7Wt5RD42/hpPy4esSNyYH1sLZpgBo0y0BUMcckk/a5Nqzfz3msswk83s3hg6XjfuDfPZOl6mG38CwcrZYPE2WTldj/O1vf6PvnzJlCtgeP348zZg/f/5dd90VcCthWHXEnOmwmVnELQ019kdMLFQBNqvdfdTlswtPpUcMOONvg4E9mQ0UfCim3FW1oWnPupbdu5oPMGQQyWTJf+JBffTlDe6WU/PGPT75Rtz/oVoQKA3LDvOff2msn09mpNpSHh1/7dQ+Y/0zEBMm8ViytTnVDNsg+NL46tWrZ8+e/cknn/CJ8gULFnCfwMiRI5ctWxawhYcnMnEBgBYP2y/PkfH05Bt+PvRCu8VhulXO1AZc7Fw+zj8CtNDk6tEyE8+ign2gs/qf1Ha9XMYAE5F4PzZ6XXmOnKcn/fziojPiXVf49GtqajhYxwfoOTuxfPlyvlNfUFAwceLEPXv2hE+k23MmLgC6ndXQBDPsqfeN+49Hx/1nfnI2Lsv2bUKhSwROBRLYH4uEG4fOemryzQPS+wbOF59Y0NvY2nxS7pjXp/5qVuFJ8akkSqpYPHsH6+vr+/XrxywIGHCadOfOnX37HlYRmVqfAGuATjcSdDQvYOfakRyHEK/eLis+mwsGH9708kt7P6j3cW9K253j4bSEXh/M0OvbLPYze028aeTss/InhddGyIdTQ1BiTOXdXnerz8ugNDataO6A0y8vmeF/nXWw8sHGsWD5o47H0EtKSl566aUTTjhh5syZ0Fm8eDF7q26++eaoacZesOcBgIvQ4m5pMkwWPd5WXl5GPTbhdnS5XVx5oO9EIMbm4yxiWHY2Jmvwk5NvOr/wpOd3LXmvalW166CNs918hKL9uhHjpBayUGSLP3sNPD4Pl7mfkDd2XuFZs4tO5cxNmLqhrNfjagqraQFJ2jLtqUOc+WOzBp/d79hpBcf0D3mtnUaCSluo1MMljczkgED3/zzIhjVV55t6L7vsst/97nelpaWctGYE2Lp165w5c/xvlOn+1gSn2POfSV1ZuXFX49edv4Lhy0nKOiV/fMA7qoLz8u+UiuaDn1R+hV0ajQrzPb732H7OcL+kCy2shHdbS/atfK9mzfaGPbWuRu7S8lrxnrfZPf9AhdOSnJ2c1je51xk542YMPv6YnBEBPzMTos1fVG3a2Vje/ho6RK5gSZxqSeubkjPQ2ad3Slb44qpuPfRRxZfeti+IxcsN1Daw+LwDM/pNyRtlbD3+HwCwb98+5j8sAAIeJDLmj3e45wEQbw5jp9/kbSlrqNxxqJxvsVQ01dS5m5ntZKWkcRN1fnJucVb/gWn5UV+mG3vzhEIsEhAAxCI9Kfutl0DUM+1vPefCgEgACQgAxAyUloAAQGn1C/MCALEBpSUgAFBa/cK8AEBsQGkJCACUVr8wLwAQG1BaAgIApdUvzAsAxAaUloAAQGn1C/MCALEBpSUgAFBa/cK8AEBsQGkJCACUVr8wLwAQG1BaAgIApdUvzAsAxAaUloAAQGn1C/MCALEBpSUgAFBa/cK8AEBsQGkJCACUVr8wLwAQG1BaAgIApdUvzAsAxAaUloAAQGn1C/MCALEBpSUgAFBa/cK8AEBsQGkJCACUVr8wLwAQG1BaAgIApdUvzAsAxAaUloAAQGn1C/MCALEBpSUgAFBa/cK8AEBsQGkJCACUVr8wLwAQG1BaAgIApdUvzAsAxAaUloAAQGn1C/MCALEBpSUgAFBa/cK8AEBsQGkJCACUVr8wLwAQG1BaAgIApdUvzAsAxAaUloAAQGn1C/MCALEBpSUgAFBa/cK8AEBsQGkJCACUVr8wLwAQG1BaAgIApdUvzAsAxAaUloAAQGn1C/MCALEBpSUgAFBa/cK8AEBsQGkJCACUVr8wLwAQG1BaAgIApdUvzAsAxAaUloAAQGn1C/MCALEBpSUgAFBa/cK8AEBsQGkJCACUVr8wLwAQG1BaAgIApdUvzAsAxAaUloAAQGn1C/MCALEBpSUgAFBa/cK8AEBsQGkJCACUVr8wLwAQG1BaAgIApdUvzAsAxAaUloAAQGn1C/MCALEBpSUgAFBa/cK8AEBsQGkJ/D+ZbgD955N+tgAAAABJRU5ErkJggg==" alt="GreenNode">`)

// resultPageTmpl renders the local PKCE callback result page. It mirrors the
// GreenNode console's MCP login-callback page so local `grn login` shares the
// same UX: GreenNode brand mark, the console's animated result-icon (a green
// check on success, a red X on failure), a title, and a one-line "you can
// close this tab" instruction. The console's button is "Go to home"; the local
// page has nowhere to go. A "Close tab" (window.close()) button was tried but
// cannot work: `grn login` opens the IAM authorize URL via the OS browser
// launcher, so the callback tab is OS-launched (not window.open'd), and
// browsers only permit scripts to close script-opened windows. So the page
// instead tells the user to close the tab manually (closeCaptions) — no button.
//
// The result-icon SVG geometry (ring/halo circle r=24, check path
// M14 27l8 8 16-16, X bars 16,16↔32,32) is copied VERBATIM from the console's
// compiled mcp-connector-callback chunk. Colors are the console's: primary
// #309036, text-success #40c646, text-danger #ff2633.
//
// The ANIMATION intentionally diverges from the console (which runs the
// per-stroke `stroke-dashoffset` draw-in as an infinite loop ending at
// opacity:0). That design hides each stroke behind a dashoffset and only
// reveals it mid-cycle, so the icon spends most of its time partially or
// fully invisible — on a one-shot callback page that read as "the X is
// missing, just a red circle" and "the V is covered in a box." Here the
// strokes are ALWAYS fully drawn (no dasharray/dashoffset): the only motion
// is a single whole-SVG entrance (`result-icon-cycle`: scale .5→1.06→1 with
// a fade, `forwards`, ending at opacity:1) plus a faint halo fade-in. The
// icon is therefore visible from the first frame and never depends on
// dashoffset timing to become legible. The <svg> carries `overflow:visible`
// and a padded viewBox ("-2 -2 52 52") so the ring's 3px stroke (reaching
// r=25.5) is not clipped to a square at the four cardinal points.
//
// The page is fully self-contained (no CDN/fonts/JS deps) so it renders offline;
// Inter is preferred but falls back to the system font. Bilingual en/vi; the
// language is chosen from the request's Accept-Language (vi preferred → vi),
// defaulting to English.
//
// Security: the template is static — only Title/Close/Lang from resultCopy
// (package constants) reach it, NEVER query params, so there is no XSS sink.
var resultPageTmpl = template.Must(template.New("result").Parse(`<!doctype html>
<html lang="{{.Lang}}">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>grn login</title>
<style>
  :root{--primary:#309036;--primary-dark:#297f2e;--success:#40c646;--danger:#ff2633;--ink:#121613;--muted:#6b7280;--bg:#f5f7fa;--card:#fff;--rule:#e5e7eb}
  *{box-sizing:border-box}
  html,body{margin:0;padding:0;height:100%}
  body{background:var(--bg);color:var(--ink);font:14px/1.5 'Inter',system-ui,-apple-system,"Segoe UI",Roboto,sans-serif;-webkit-font-smoothing:antialiased;display:flex;align-items:center;justify-content:center;min-height:100vh;padding:24px}
  .card{background:var(--card);border:1px solid var(--rule);border-radius:12px;box-shadow:0 1px 2px rgba(0,0,0,.04),0 8px 24px -16px rgba(0,0,0,.12);width:100%;max-width:440px;padding:40px 32px 28px;text-align:center}
  .brand{display:flex;align-items:center;justify-content:center;margin-bottom:28px}
  .brand-mark{height:72px;width:auto;max-width:220px;display:block;-webkit-user-drag:none;user-select:none}
  .text-success{color:var(--success)}
  .text-danger{color:var(--danger)}
  .result-icon{--result-icon-size:72px}.result-icon__svg{overflow:visible;width:var(--result-icon-size);height:var(--result-icon-size);animation:result-icon-cycle 3s forwards}.result-icon__halo{fill:currentColor;opacity:0;animation:result-halo-cycle 3s forwards}.result-icon__ring{fill:none;stroke:currentColor;stroke-width:3;stroke-linecap:round}{{if .Ok}}.result-icon__check{fill:none;stroke:currentColor;stroke-width:4;stroke-linecap:round;stroke-linejoin:round}{{else}}.result-icon__bar{fill:none;stroke:currentColor;stroke-width:4;stroke-linecap:round}{{end}}@keyframes result-icon-cycle{0%{transform:scale(.5);opacity:0;animation-timing-function:cubic-bezier(.34,1.56,.64,1)}12%{transform:scale(1.06);opacity:1}18%{transform:scale(1);opacity:1;animation-timing-function:ease}86%{transform:scale(1);opacity:1;animation-timing-function:ease-out}to{transform:scale(1);opacity:1}}@keyframes result-halo-cycle{0%{opacity:0}20%{opacity:.08}to{opacity:.08}}
  .result-icon{width:72px;height:72px;margin:0 auto 24px;display:flex;align-items:center;justify-content:center}
  h1{font-size:20px;font-weight:700;margin:0 0 24px;letter-spacing:-.01em;color:var(--ink)}
  .close-caption{margin:0;color:var(--muted);font-size:14px}
  @media (max-width:480px){.card{padding:32px 20px 22px}}
</style>
</head>
<body>
  <div class="card">
    <div class="brand">{{.BrandImg}}</div>
    <div class="result-icon {{if .Ok}}text-success{{else}}text-danger{{end}}">
      <svg class="result-icon__svg" viewBox="-2 -2 52 52" aria-hidden="true">
        <circle class="result-icon__halo" cx="24" cy="24" r="24"></circle>
        <circle class="result-icon__ring" cx="24" cy="24" r="24"></circle>
        {{if .Ok}}<path class="result-icon__check" d="M14 27l8 8 16-16"></path>{{else}}<line class="result-icon__bar--1" x1="16" y1="16" x2="32" y2="32"></line><line class="result-icon__bar--2" x1="32" y1="16" x2="16" y2="32"></line>{{end}}
      </svg>
    </div>
    <h1>{{.Title}}</h1>
    <p class="close-caption">{{.Close}}</p>
  </div>
</body>
</html>`))

// preferVietnamese returns true when the request's Accept-Language prefers
// Vietnamese (the first comma-separated tag starts with "vi"). Defaults to
// English otherwise. This is a deliberately simple heuristic — the browser
// signals the user's language and we honor it; no CLI flag needed.
func preferVietnamese(r *http.Request) bool {
	al := r.Header.Get("Accept-Language")
	if al == "" {
		return false
	}
	first := al
	if i := strings.IndexByte(al, ','); i >= 0 {
		first = al[:i]
	}
	if i := strings.IndexByte(first, ';'); i >= 0 {
		first = first[:i]
	}
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(first)), "vi")
}

func writeResult(w http.ResponseWriter, r *http.Request, ok bool, kind resultKind) {
	status := http.StatusOK
	if !ok {
		status = http.StatusBadRequest
	}
	cp := resultCopy[kind]
	lang := "en"
	title, closeLine := cp.En.Title, closeCaptions.En
	if preferVietnamese(r) {
		lang = "vi"
		title, closeLine = cp.Vi.Title, closeCaptions.Vi
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_ = resultPageTmpl.Execute(w, struct {
		Lang     string
		Ok       bool
		Title    string
		Close    string
		BrandImg template.HTML
	}{Lang: lang, Ok: ok, Title: title, Close: closeLine, BrandImg: brandMarkImg})
}
