package ssm

import (
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
)

// The value of a secure parameter is not decrypted when the parameters or
// their history are listed, so that scrolling through them decrypts nothing.
// What is listed for one is its encrypted form, which is never shown.
const (
	// secureValueHint is shown where the value would be read
	secureValueHint = "Secure value, not decrypted. Press Enter on the row to decrypt and show it."
	// secureValueCell is shown in a table cell
	secureValueCell = "(secure)"
)

func isSecure(paramType types.ParameterType) bool {
	return paramType == types.ParameterTypeSecureString
}

// valueVersion is one version of a parameter
type valueVersion struct {
	name    string
	version int64
}

// decryptedValues are the secure values the user has asked to see, which are
// kept while the page is, so they are only decrypted once. A version never
// changes, so a value that is kept is never out of date. They are only used on
// the UI goroutine.
type decryptedValues map[valueVersion]string

// text is what to show for the value of the version: the value as it was
// listed, unless it is secure, which is shown once it was decrypted
func (inst decryptedValues) text(name string, version int64, paramType types.ParameterType, listed string) string {
	if !isSecure(paramType) {
		return listed
	}
	if value, shown := inst[valueVersion{name, version}]; shown {
		return value
	}
	return secureValueHint
}
