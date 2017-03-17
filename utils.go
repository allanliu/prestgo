package prestgo

import "bytes"

// escapeBytesBackslash escapes []byte with backslashes (\)
func escapeBytesBackslash(buf *bytes.Buffer, v []byte) {
	for _, c := range v {
		switch c {
		case '\x00':
			buf.Write([]byte{'\\', '0'})

		case '\n':
			buf.Write([]byte{'\\', 'n'})

		case '\r':
			buf.Write([]byte{'\\', 'r'})

		case '\x1a':
			buf.Write([]byte{'\\', 'Z'})

		case '\'':
			buf.Write([]byte{'\\', '\''})

		case '"':
			buf.Write([]byte{'\\', '"'})

		case '\\':
			buf.Write([]byte{'\\', '\\'})

		default:
			buf.WriteByte(c)
		}
	}
}

// escapeStringBackslash is similar to escapeBytesBackslash but for string.
func escapeStringBackslash(buf *bytes.Buffer, v string) {
	for i := 0; i < len(v); i++ {
		switch v[i] {
		case '\x00':
			buf.Write([]byte{'\\', '0'})

		case '\n':
			buf.Write([]byte{'\\', 'n'})

		case '\r':
			buf.Write([]byte{'\\', 'r'})

		case '\x1a':
			buf.Write([]byte{'\\', 'Z'})

		case '\'':
			buf.Write([]byte{'\\', '\''})

		case '"':
			buf.Write([]byte{'\\', '"'})

		case '\\':
			buf.Write([]byte{'\\', '\\'})

		default:
			buf.WriteByte(v[i])
		}
	}
}
