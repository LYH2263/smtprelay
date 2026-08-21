package dkim

import "testing"

func TestSignHasTags(t *testing.T) {
        pem, err := GenerateTestKey()
        if err != nil {
                t.Fatal(err)
        }
        sig, err := Sign(Params{
                Domain: "example.com", Selector: "mail", PrivateKeyPEM: pem,
                Headers: map[string]string{"From": "a@example.com", "To": "b@example.com", "Subject": "s", "Date": "x"},
                Body:    []byte("hello\r\n"),
        })
        if err != nil {
                t.Fatal(err)
        }
        tags := ParseSignatureTags(sig)
        if !HasRequiredTags(tags) {
                t.Fatalf("tags=%v", tags)
        }
}
