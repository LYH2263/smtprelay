package main

import (
        "encoding/json"
        "flag"
        "log"
        "net/http"
        "os"
        "path/filepath"
        "sync"
        "time"

        smtprelay "github.com/LYH2263/go-smtprelay"
        "github.com/LYH2263/go-smtprelay/internal/smtpdial"
)

func main() {
        addr := flag.String("addr", ":8110", "listen address")
        web := flag.String("web", "web", "static web directory")
        data := flag.String("data", "data", "data directory")
        flag.Parse()

        _ = os.MkdirAll(*data, 0o755)
        dialer := &smtpdial.MemoryDialer{Code: 250, Response: "OK queued (lab)"}
        relay, err := smtprelay.New(smtprelay.Options{
                PersistPath: filepath.Join(*data, "queue.json"),
                AuditDir:    filepath.Join(*data, "audit"),
                Dialer:      dialer,
                DirectHost:  "127.0.0.1",
                DirectPort:  2525,
                MaxQueue:    2048,
        })
        if err != nil {
                log.Fatal(err)
        }
        defer relay.Close()

        var mu sync.Mutex
        mux := http.NewServeMux()
        mux.Handle("/", http.FileServer(http.Dir(*web)))

        mux.HandleFunc("/api/stats", func(w http.ResponseWriter, r *http.Request) {
                mu.Lock()
                defer mu.Unlock()
                writeJSON(w, relay.Stats())
        })
        mux.HandleFunc("/api/queue", func(w http.ResponseWriter, r *http.Request) {
                mu.Lock()
                defer mu.Unlock()
                writeJSON(w, relay.ListPending())
        })
        mux.HandleFunc("/api/bounces", func(w http.ResponseWriter, r *http.Request) {
                mu.Lock()
                defer mu.Unlock()
                writeJSON(w, relay.ListBounces(50))
        })
        mux.HandleFunc("/api/submit", func(w http.ResponseWriter, r *http.Request) {
                if r.Method != http.MethodPost {
                        http.Error(w, "POST only", 405)
                        return
                }
                var req struct {
                        MailFrom string   `json:"mail_from"`
                        To       []string `json:"to"`
                        Subject  string   `json:"subject"`
                        Body     string   `json:"body"`
                }
                if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
                        http.Error(w, err.Error(), 400)
                        return
                }
                env := &smtprelay.Envelope{
                        MailFrom:  req.MailFrom,
                        Subject:   req.Subject,
                        RawBody:   []byte(req.Body),
                        CreatedAt: time.Now(),
                }
                for _, a := range req.To {
                        env.Recipients = append(env.Recipients, smtprelay.Recipient{Address: a})
                }
                mu.Lock()
                id, err := relay.Submit(env)
                mu.Unlock()
                if err != nil {
                        http.Error(w, err.Error(), 400)
                        return
                }
                writeJSON(w, map[string]string{"id": id})
        })
        mux.HandleFunc("/api/deliver", func(w http.ResponseWriter, r *http.Request) {
                if r.Method != http.MethodPost {
                        http.Error(w, "POST only", 405)
                        return
                }
                mu.Lock()
                res, err := relay.DeliverOnce(r.Context())
                mu.Unlock()
                writeJSON(w, map[string]any{"result": res, "error": errString(err)})
        })

        log.Printf("maild listening on %s", *addr)
        log.Fatal(http.ListenAndServe(*addr, mux))
}

func writeJSON(w http.ResponseWriter, v any) {
        w.Header().Set("Content-Type", "application/json")
        _ = json.NewEncoder(w).Encode(v)
}

func errString(err error) string {
        if err == nil {
                return ""
        }
        return err.Error()
}
