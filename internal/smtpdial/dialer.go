package smtpdial

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"
)

// Request is one SMTP DATA transaction.
type Request struct {
	Host     string
	Port     int
	MailFrom string
	RcptTo   []string
	Data     []byte
	Helo     string
}

// Dialer sends mail over SMTP (real or fake).
type Dialer interface {
	Send(ctx context.Context, req Request) (code int, response string, err error)
}

// MemoryDialer records transactions and returns a fixed result.
type MemoryDialer struct {
	Code     int
	Response string
	Err      error
	Delay    time.Duration
	Last     Request
	History  []Request
}

func (m *MemoryDialer) Send(ctx context.Context, req Request) (int, string, error) {
	if m.Delay > 0 {
		// Honor ctx so a cancelled caller does not wait out the full Delay.
		timer := time.NewTimer(m.Delay)
		select {
		case <-timer.C:
		case <-ctx.Done():
			timer.Stop()
			return 0, "", ctx.Err()
		}
	}
	m.Last = req
	cp := req
	cp.RcptTo = append([]string(nil), req.RcptTo...)
	cp.Data = append([]byte(nil), req.Data...)
	m.History = append(m.History, cp)
	code := m.Code
	if code == 0 && m.Err == nil {
		code = 250
	}
	resp := m.Response
	if resp == "" && m.Err == nil {
		resp = "OK"
	}
	return code, resp, m.Err
}

// NetDialer is a minimal real SMTP client (no STARTTLS).
type NetDialer struct {
	Timeout time.Duration
	Helo    string
}

func (d NetDialer) Send(ctx context.Context, req Request) (int, string, error) {
	if req.Port <= 0 {
		req.Port = 25
	}
	timeout := d.Timeout
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	dialer := net.Dialer{Timeout: timeout}
	conn, err := dialer.DialContext(ctx, "tcp", fmt.Sprintf("%s:%d", req.Host, req.Port))
	if err != nil {
		return 0, "", err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(timeout))

	// Honor ctx during the SMTP exchange: closing the conn cancels any
	// in-flight read/write so Send returns promptly on cancellation.
	stop := make(chan struct{})
	defer close(stop)
	go func() {
		select {
		case <-ctx.Done():
			_ = conn.Close()
		case <-stop:
		}
	}()

	sess := &session{conn: conn}
	if code, msg, err := sess.read(); err != nil {
		return code, msg, err
	} else if code != 220 {
		return code, msg, fmt.Errorf("smtpdial: greeting %d %s", code, msg)
	}
	helo := d.Helo
	if helo == "" {
		helo = req.Helo
	}
	if helo == "" {
		helo = "localhost"
	}
	if code, msg, err := sess.cmd("HELO " + helo); err != nil || code != 250 {
		return code, msg, errOr(code, msg, err, "HELO")
	}
	if code, msg, err := sess.cmd("MAIL FROM:<" + stripAngles(req.MailFrom) + ">"); err != nil || code != 250 {
		return code, msg, errOr(code, msg, err, "MAIL")
	}
	for _, rcpt := range req.RcptTo {
		if code, msg, err := sess.cmd("RCPT TO:<" + stripAngles(rcpt) + ">"); err != nil || (code != 250 && code != 251) {
			return code, msg, errOr(code, msg, err, "RCPT")
		}
	}
	if code, msg, err := sess.cmd("DATA"); err != nil || code != 354 {
		return code, msg, errOr(code, msg, err, "DATA")
	}
	payload := dotStuff(req.Data)
	if _, err := conn.Write(payload); err != nil {
		return 0, "", err
	}
	if _, err := conn.Write([]byte("\r\n.\r\n")); err != nil {
		return 0, "", err
	}
	code, msg, err := sess.read()
	if err != nil {
		return code, msg, err
	}
	_, _, _ = sess.cmd("QUIT")
	if code != 250 {
		return code, msg, fmt.Errorf("smtpdial: DATA result %d %s", code, msg)
	}
	return code, msg, nil
}

func errOr(code int, msg string, err error, what string) error {
	if err != nil {
		return err
	}
	return fmt.Errorf("smtpdial: %s failed: %d %s", what, code, msg)
}

func stripAngles(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.LastIndex(s, "<"); i >= 0 {
		j := strings.LastIndex(s, ">")
		if j > i {
			return s[i+1 : j]
		}
	}
	return s
}

func dotStuff(data []byte) []byte {
	s := strings.ReplaceAll(string(data), "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	lines := strings.Split(s, "\n")
	var b strings.Builder
	for i, line := range lines {
		if strings.HasPrefix(line, ".") {
			b.WriteByte('.')
		}
		b.WriteString(line)
		if i != len(lines)-1 {
			b.WriteString("\r\n")
		}
	}
	return []byte(b.String())
}

type session struct {
	conn net.Conn
	buf  []byte
}

func (s *session) read() (int, string, error) {
	var lines []string
	for {
		line, err := s.readLine()
		if err != nil {
			return 0, "", err
		}
		if len(line) < 4 {
			return 0, line, fmt.Errorf("smtpdial: short response")
		}
		lines = append(lines, line)
		if line[3] == ' ' {
			var code int
			fmt.Sscanf(line[:3], "%d", &code)
			return code, strings.Join(lines, "\n"), nil
		}
	}
}

func (s *session) readLine() (string, error) {
	for {
		if i := indexCRLF(s.buf); i >= 0 {
			line := string(s.buf[:i])
			s.buf = s.buf[i+2:]
			return line, nil
		}
		tmp := make([]byte, 1024)
		n, err := s.conn.Read(tmp)
		if n > 0 {
			s.buf = append(s.buf, tmp[:n]...)
		}
		if err != nil {
			return "", err
		}
	}
}

func (s *session) cmd(line string) (int, string, error) {
	if _, err := s.conn.Write([]byte(line + "\r\n")); err != nil {
		return 0, "", err
	}
	return s.read()
}

func indexCRLF(b []byte) int {
	for i := 0; i+1 < len(b); i++ {
		if b[i] == '\r' && b[i+1] == '\n' {
			return i
		}
	}
	return -1
}
