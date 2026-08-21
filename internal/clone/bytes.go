package clone

// Bytes returns an independent copy of b.
func Bytes(b []byte) []byte {
        if b == nil {
                return nil
        }
        out := make([]byte, len(b))
        copy(out, b)
        return out
}

// HeaderMap returns a shallow copy of header key/values.
func HeaderMap(in map[string]string) map[string]string {
        if in == nil {
                return nil
        }
        out := make(map[string]string, len(in))
        for k, v := range in {
                out[k] = v
        }
        return out
}

// Strings copies a string slice.
func Strings(in []string) []string {
        if in == nil {
                return nil
        }
        out := make([]string, len(in))
        copy(out, in)
        return out
}
