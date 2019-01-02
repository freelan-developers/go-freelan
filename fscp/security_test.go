package fscp

import "testing"

func TestCipherSuiteSliceFindCommon(t *testing.T) {
	a := CipherSuiteSlice{
		ECDHERSAAES128GCMSHA256,
		ECDHERSAAES256GCMSHA384,
	}
	b := CipherSuiteSlice{
		ECDHERSAAES256GCMSHA384,
		ECDHERSAAES128GCMSHA256,
	}
	c := CipherSuiteSlice{
		CipherSuite(0xff),
	}

	value := a.FindCommon(b)
	expected := ECDHERSAAES128GCMSHA256

	if value != expected {
		t.Errorf("expected %s but got %s", expected, value)
	}

	value = b.FindCommon(a)
	expected = ECDHERSAAES256GCMSHA384

	if value != expected {
		t.Errorf("expected %s but got %s", expected, value)
	}

	if value = a.FindCommon(c); value != NullCipherSuite {
		t.Fatalf("a null cipher suite was expected")
	}
}

func TestElliptiCurveSliceFindCommon(t *testing.T) {
	a := EllipticCurveSlice{
		SECT571K1,
		SECP384R1,
	}
	b := EllipticCurveSlice{
		SECP384R1,
		SECT571K1,
	}
	c := EllipticCurveSlice{
		SECP521R1,
	}

	value := a.FindCommon(b)
	expected := SECT571K1

	if value != expected {
		t.Errorf("expected %s but got %s", expected, value)
	}

	value = b.FindCommon(a)
	expected = SECP384R1

	if value != expected {
		t.Errorf("expected %s but got %s", expected, value)
	}

	if value = a.FindCommon(c); value != NullEllipticCurve {
		t.Fatalf("a null elliptic curve was expected")
	}
}
