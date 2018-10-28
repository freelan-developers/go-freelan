// Code generated by "stringer -type EllipticCurve"; DO NOT EDIT.

package fscp

import "strconv"

const _EllipticCurve_name = "SECT571K1SECP384R1SECP521R1"

var _EllipticCurve_index = [...]uint8{0, 9, 18, 27}

func (i EllipticCurve) String() string {
	i -= 1
	if i >= EllipticCurve(len(_EllipticCurve_index)-1) {
		return "EllipticCurve(" + strconv.FormatInt(int64(i+1), 10) + ")"
	}
	return _EllipticCurve_name[_EllipticCurve_index[i]:_EllipticCurve_index[i+1]]
}