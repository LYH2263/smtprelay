package smtprelay_test

import (
        "bytes"
        "context"
        "errors"
        "path/filepath"
        "testing"
        "time"

        smtprelay "github.com/LYH2263/go-smtprelay"
        "github.com/LYH2263/go-smtprelay/internal/smtpdial"
)

func TestSubmitPeekAck_Smoke(t *testing.T) {
        r, err := smtprelay.New(smtprelay.Options{MaxQueue: 16})
        if err != nil {
                t.Fatal(err)
        }
        defer r.Close()

        body := []byte("hello body")
        id, err := r.Submit(&smtprelay.Envelope{
                MailFrom:   "a@example.com",
                Recipients: []smtprelay.Recipient{{Address: "b@example.com"}},
                Subject:    "hi",
                RawBody:    body,
        })
        if err != nil {
                t.Fatal(err)
        }
        // caller mutates original body — queued copy must stay intact
        body[0] = 'X'
        got, err := r.Get(id)
        if err != nil {
                t.Fatal(err)
        }
        if got.RawBody[0] != 'h' {
                t.Fatalf("queued body polluted: %q", got.RawBody)
        }

        peek, err := r.Peek()
        if err != nil {
                t.Fatal(err)
        }
        if peek.ID != id {
                t.Fatalf("peek id=%s want %s", peek.ID, id)
        }
        if err := r.Ack(id); err != nil {
                t.Fatal(err)
        }
        if _, err := r.Peek(); !errors.Is(err, smtprelay.ErrNotFound) {
                t.Fatalf("want ErrNotFound, got %v", err)
        }
}

func TestListPending_RecipientIsolation(t *testing.T) {
        r, err := smtprelay.New(smtprelay.Options{})
        if err != nil {
                t.Fatal(err)
        }
        defer r.Close()
        _, err = r.Submit(&smtprelay.Envelope{
                MailFrom:   "a@example.com",
                Recipients: []smtprelay.Recipient{{Address: "b@example.com"}},
                RawBody:    []byte("x"),
        })
        if err != nil {
                t.Fatal(err)
        }
        list := r.ListPending()
        if len(list) != 1 {
                t.Fatalf("len=%d", len(list))
        }
        list[0].Recipients[0].Address = "evil@example.com"
        again := r.ListPending()
        if again[0].Recipients[0].Address != "b@example.com" {
                t.Fatalf("internal recipients shared: %v", again[0].Recipients)
        }
}

func TestDeliverOnce_MemoryDialer(t *testing.T) {
        dialer := &smtpdial.MemoryDialer{Code: 250, Response: "OK"}
        r, err := smtprelay.New(smtprelay.Options{
                Dialer:     dialer,
                DirectHost: "mx.example.com",
                DirectPort: 25,
        })
        if err != nil {
                t.Fatal(err)
        }
        defer r.Close()
        _, err = r.Submit(&smtprelay.Envelope{
                MailFrom:   "a@example.com",
                Recipients: []smtprelay.Recipient{{Address: "b@example.com"}},
                Subject:    "s",
                MimeParts:  []smtprelay.MimePart{{ContentType: "text/plain", Charset: "utf-8", Data: []byte("part")}},
        })
        if err != nil {
                t.Fatal(err)
        }
        res, err := r.DeliverOnce(context.Background())
        if err != nil {
                t.Fatal(err)
        }
        if res.Code != 250 {
                t.Fatalf("code=%d", res.Code)
        }
        if !bytes.Contains(dialer.Last.Data, []byte("part")) {
                t.Fatalf("DATA missing body: %s", dialer.Last.Data)
        }
        st := r.Stats()
        if st.Pending != 0 || st.Total != 0 {
                t.Fatalf("stats after ack: %+v", st)
        }
}

func TestDeliverOnce_NoDialer(t *testing.T) {
        r, err := smtprelay.New(smtprelay.Options{DirectHost: "x"})
        if err != nil {
                t.Fatal(err)
        }
        defer r.Close()
        _, _ = r.Submit(&smtprelay.Envelope{
                MailFrom: "a@example.com", Recipients: []smtprelay.Recipient{{Address: "b@example.com"}}, RawBody: []byte("z"),
        })
        _, err = r.DeliverOnce(context.Background())
        if !errors.Is(err, smtprelay.ErrNoDialer) {
                t.Fatalf("want ErrNoDialer, got %v", err)
        }
}

func TestPermanentBounce_WrapsErrPermanent(t *testing.T) {
        dialer := &smtpdial.MemoryDialer{Code: 550, Response: "user unknown", Err: errors.New("rcpt failed")}
        r, err := smtprelay.New(smtprelay.Options{Dialer: dialer, DirectHost: "mx", MaxAttempts: 3})
        if err != nil {
                t.Fatal(err)
        }
        defer r.Close()
        _, _ = r.Submit(&smtprelay.Envelope{
                MailFrom: "a@example.com", Recipients: []smtprelay.Recipient{{Address: "b@example.com"}}, RawBody: []byte("z"),
        })
        res, err := r.DeliverOnce(context.Background())
        if !errors.Is(err, smtprelay.ErrPermanent) {
                t.Fatalf("want ErrPermanent, got %v", err)
        }
        if smtprelay.ClassifyBounce(res) != smtprelay.BouncePermanent {
                t.Fatalf("class=%s", smtprelay.ClassifyBounce(res))
        }
        if len(r.ListBounces(10)) != 1 {
                t.Fatalf("bounces=%d", len(r.ListBounces(10)))
        }
}

func TestCloseRejectsSubmit(t *testing.T) {
        r, err := smtprelay.New(smtprelay.Options{})
        if err != nil {
                t.Fatal(err)
        }
        if err := r.Close(); err != nil {
                t.Fatal(err)
        }
        _, err = r.Submit(&smtprelay.Envelope{
                MailFrom: "a@example.com", Recipients: []smtprelay.Recipient{{Address: "b@example.com"}}, RawBody: []byte("z"),
        })
        if !errors.Is(err, smtprelay.ErrClosed) {
                t.Fatalf("want ErrClosed, got %v", err)
        }
}

func TestPersistRoundTrip(t *testing.T) {
        dir := t.TempDir()
        path := filepath.Join(dir, "q.json")
        r, err := smtprelay.New(smtprelay.Options{PersistPath: path})
        if err != nil {
                t.Fatal(err)
        }
        id, err := r.Submit(&smtprelay.Envelope{
                MailFrom: "a@example.com", Recipients: []smtprelay.Recipient{{Address: "b@example.com"}}, RawBody: []byte("persist-me"),
        })
        if err != nil {
                t.Fatal(err)
        }
        if err := r.Close(); err != nil {
                t.Fatal(err)
        }
        r2, err := smtprelay.New(smtprelay.Options{PersistPath: path})
        if err != nil {
                t.Fatal(err)
        }
        defer r2.Close()
        got, err := r2.Get(id)
        if err != nil {
                t.Fatal(err)
        }
        if string(got.RawBody) != "persist-me" {
                t.Fatalf("body=%q", got.RawBody)
        }
}

func TestSubmitContextCanceled(t *testing.T) {
        r, err := smtprelay.New(smtprelay.Options{})
        if err != nil {
                t.Fatal(err)
        }
        defer r.Close()
        ctx, cancel := context.WithCancel(context.Background())
        cancel()
        _, err = r.SubmitContext(ctx, &smtprelay.Envelope{
                MailFrom: "a@example.com", Recipients: []smtprelay.Recipient{{Address: "b@example.com"}}, RawBody: []byte("z"),
        })
        if !errors.Is(err, context.Canceled) {
                t.Fatalf("want canceled, got %v", err)
        }
}

func TestDialAndSendHonorsContext(t *testing.T) {
        dialer := &smtpdial.MemoryDialer{Delay: 200 * time.Millisecond, Code: 250}
        r, err := smtprelay.New(smtprelay.Options{Dialer: dialer})
        if err != nil {
                t.Fatal(err)
        }
        defer r.Close()
        ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
        defer cancel()
        _, err = r.DialAndSend(ctx, dialer, "h", 25, &smtprelay.Envelope{
                MailFrom: "a@example.com", Recipients: []smtprelay.Recipient{{Address: "b@example.com"}}, RawBody: []byte("z"),
        })
        if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
                t.Fatalf("want ctx error, got %v", err)
        }
}

func TestBuildMIME_Multipart(t *testing.T) {
        raw, err := smtprelay.BuildMIME(&smtprelay.Envelope{
                MailFrom:   "a@example.com",
                Recipients: []smtprelay.Recipient{{Address: "b@example.com"}},
                Subject:    "mix",
                MimeParts: []smtprelay.MimePart{
                        {ContentType: "text/plain", Data: []byte("hello")},
                        {ContentType: "application/octet-stream", Disposition: "attachment", Filename: "f.bin", Data: []byte{1, 2, 3}},
                },
        })
        if err != nil {
                t.Fatal(err)
        }
        if !bytes.Contains(raw, []byte("multipart/mixed")) {
                t.Fatalf("not multipart: %s", raw)
        }
}
