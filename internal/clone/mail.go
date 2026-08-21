package clone

// Addr is a mailbox with optional display name.
type Addr struct {
        Address string
        Name    string
}

// Part is one MIME part payload.
type Part struct {
        ContentType string
        Charset     string
        Disposition string
        Filename    string
        ContentID   string
        Data        []byte
}

// Addrs copies address slices.
func Addrs(in []Addr) []Addr {
        if in == nil {
                return nil
        }
        out := make([]Addr, len(in))
        copy(out, in)
        return out
}

// Parts deep-copies MIME parts including Data.
func Parts(in []Part) []Part {
        if in == nil {
                return nil
        }
        out := make([]Part, len(in))
        for i, p := range in {
                out[i] = p
                out[i].Data = Bytes(p.Data)
        }
        return out
}
