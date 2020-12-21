package session

type SessionStorable interface {
	/**
	 * Encode byte [ ].
	 *
	 * @return the byte [ ]
	 */
	Encode() ([]byte, error)

	/**
	 * Decode.
	 *
	 * @param src the src
	 */
	Decode(src []byte)
}
