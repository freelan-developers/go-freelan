package fscp

// Session represents an outgoing or incoming session.
type Session struct {
	SessionNumber SessionNumber
	CipherSuite   CipherSuite
	EllipticCurve EllipticCurve
}
