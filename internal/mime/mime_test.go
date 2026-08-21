package mime

import "testing"

func TestSplitSerializeRoundTrip(t *testing.T) {
        raw := Serialize(map[string]string{"From": "a@b", "Subject": "x"}, []byte("body\r\nline"))
        h, b, err := SplitMessage(raw)
        if err != nil {
                t.Fatal(err)
        }
        if h["From"] != "a@b" || h["Subject"] != "x" {
                t.Fatalf("headers=%v", h)
        }
        if string(b) != "body\r\nline" {
                t.Fatalf("body=%q", b)
        }
}
