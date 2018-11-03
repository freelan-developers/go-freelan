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

	value, err := a.FindCommon(b)

	if err != nil {
		t.Fatalf("no error was expected but got: %s", err)
	}

	expected := ECDHERSAAES128GCMSHA256

	if value != expected {
		t.Errorf("expected %s but got %s", expected, value)
	}

	value, err = b.FindCommon(a)

	if err != nil {
		t.Fatalf("no error was expected but got: %s", err)
	}

	expected = ECDHERSAAES256GCMSHA384

	if value != expected {
		t.Errorf("expected %s but got %s", expected, value)
	}

	if _, err = a.FindCommon(c); err == nil {
		t.Fatalf("an error was expected")
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

	value, err := a.FindCommon(b)

	if err != nil {
		t.Fatalf("no error was expected but got: %s", err)
	}

	expected := SECT571K1

	if value != expected {
		t.Errorf("expected %s but got %s", expected, value)
	}

	value, err = b.FindCommon(a)

	if err != nil {
		t.Fatalf("no error was expected but got: %s", err)
	}

	expected = SECP384R1

	if value != expected {
		t.Errorf("expected %s but got %s", expected, value)
	}

	if _, err = a.FindCommon(c); err == nil {
		t.Fatalf("an error was expected")
	}
}
